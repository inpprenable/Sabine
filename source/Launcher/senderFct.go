package Launcher

import (
	"encoding/gob"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net"
	"pbftnode/source/Blockchain"
	"sync"
	"syscall"
)

const nbMessChan = 128

type contacts struct {
	listContact []string
	idContact   int
	conn        *net.Conn
	encoder     *gob.Encoder
	sync.WaitGroup
}

func newContacts(Contact string, listContact []string) *contacts {
	conn, _ := connectWith(Contact)
	encoder := gob.NewEncoder(conn)
	return &contacts{
		listContact: listContact,
		idContact:   0,
		conn:        &conn,
		encoder:     encoder,
	}
}

type sender interface {
	close()
	send(transac Blockchain.Transaction, priority bool, broadcastType Blockchain.BroadcastType)
	Wait()
	Add(int)
}

func (contact *contacts) send(transac Blockchain.Transaction, priority bool, _ Blockchain.BroadcastType) {
	sendTx(transac, contact.encoder, contact, priority)
	contact.Done()
}

func (contact *contacts) close() {
	conn := contact.conn
	_ = (*conn).Close()
}

func sendTx(transaction Blockchain.Transaction, encoder *gob.Encoder, listContact *contacts, priority bool) {
	var err error
	err = encoder.Encode(Blockchain.Message{
		Priority:    priority,
		Flag:        Blockchain.TransactionMess,
		Data:        transaction,
		ToBroadcast: Blockchain.AskToBroadcast,
	})
	log.Debug().Msgf("Transaction send %s", transaction.GetHashPayload())

	if err != nil {
		operr, ok := err.(*net.OpError)
		if ok || operr.Err.Error() == syscall.ECONNRESET.Error() {
			if listContact.idContact+1 < len(listContact.listContact) {
				log.Warn().Msg("Changement du noeud adressé")
				listContact.idContact++
				*listContact.conn, _ = connectWith(listContact.listContact[listContact.idContact])
				*encoder = *gob.NewEncoder(*listContact.conn)
			}
		} else {
			check(err)
		}
	}
}

type singleSocket struct {
	contact    string
	id         int
	conn       net.Conn
	encoder    *gob.Encoder
	chanMsg    chan Blockchain.Message
	chanRemove chan query
	closeTx    chan chan struct{}
}

func newSingleSocket(contact string, wait *sync.WaitGroup, chanRemove chan query) *singleSocket {
	conn, id := connectWith(contact)
	socket := &singleSocket{
		contact:    contact,
		id:         id,
		conn:       conn,
		chanMsg:    make(chan Blockchain.Message, nbMessChan),
		closeTx:    make(chan chan struct{}),
		chanRemove: chanRemove,
	}
	socket.encoder = gob.NewEncoder(socket.conn)
	go socket.loopSocket(wait)
	return socket
}

type query struct {
	contact string
	channel chan struct{}
}

func newQuery(contact string) query {
	return query{contact, make(chan struct{})}
}

func (query *query) wait() {
	<-query.channel
	close(query.channel)
}

func (socket *singleSocket) loopSocket(wait *sync.WaitGroup) {
	for {
		select {
		case msg := <-socket.chanMsg:
			err := socket.encoder.Encode(msg)
			wait.Done()
			if err != nil {
				operr, ok := err.(*net.OpError)
				if ok || operr.Err.Error() == syscall.ECONNRESET.Error() {
					log.Warn().Msg("Supression du lien ")
					query := newQuery(socket.contact)
					socket.chanRemove <- query
					for range socket.chanMsg {
						wait.Done()
					}
					query.wait()
					return
				}
			} else {
				log.Trace().Msgf("Message send to %d", socket.id)
			}
		case channel := <-socket.closeTx:
			for range socket.chanMsg {
				wait.Done()
			}
			err := socket.conn.Close()
			check(err)
			channel <- struct{}{}
			return
		}
	}
}

type multiSend struct {
	chanMsg        chan Blockchain.Message
	closeChan      chan chan struct{}
	waiter         sync.WaitGroup
	uniformDistrib bool
}

func (send *multiSend) close() {
	channel := make(chan struct{})
	send.closeChan <- channel
	close(send.closeChan)
	close(send.chanMsg)
}

func (send *multiSend) send(transac Blockchain.Transaction, priority bool, broadcastType Blockchain.BroadcastType) {
	msg := Blockchain.Message{
		Priority:    priority,
		Flag:        Blockchain.TransactionMess,
		Data:        transac,
		ToBroadcast: broadcastType,
	}
	send.chanMsg <- msg
}

func (send *multiSend) Wait() {
	send.waiter.Wait()
}

func (send *multiSend) Add(id int) {
	send.waiter.Add(id)
}

func newMultiSend(listContact []string, uniformDistrib bool) *multiSend {
	multisend := &multiSend{make(chan Blockchain.Message), make(chan chan struct{}), sync.WaitGroup{}, uniformDistrib}
	go multisend.loopSendTxMulti(listContact, multisend.chanMsg, multisend.closeChan, &multisend.waiter)
	return multisend
}

func (send *multiSend) loopSendTxMulti(listContact []string, chanMsg <-chan Blockchain.Message, closeChan chan chan struct{}, wait *sync.WaitGroup) {
	n := len(listContact)
	listSocket := make([]*singleSocket, n)
	chanRemove := make(chan query)
	for i := 0; i < n; i++ {
		listSocket[i] = newSingleSocket(listContact[i], wait, chanRemove)
	}
	for {
		select {
		case query := <-chanRemove:
			var size int = -1
			for i, socket := range listSocket {
				if socket.contact == query.contact {
					size = i
					break
				}
			}
			if size > 0 {
				listSocket = append(listSocket[:n], listSocket[n+1:]...)
				n--
			}
		case msg := <-chanMsg:
			if send.uniformDistrib {
				uniformDistribution(listSocket, msg)
			} else {
				randomDistribution(n, listSocket, msg)
			}
		case channel := <-closeChan:
			listChannel := make([]chan struct{}, n)
			for i, socket := range listSocket {
				listChannel[i] = make(chan struct{})
				socket.closeTx <- listChannel[i]
			}
			for _, closeQuery := range listChannel {
				<-closeQuery
				close(closeQuery)
			}
			channel <- struct{}{}
			return
		}
	}
}

func randomDistribution(n int, listSocket []*singleSocket, msg Blockchain.Message) {
	i := rand.Intn(n)
	listSocket[i].chanMsg <- msg
}

func uniformDistribution(listSocket []*singleSocket, msg Blockchain.Message) {
	msg.ToBroadcast = Blockchain.DontBroadcast
	log.Trace().Msgf("Try to send the message msg %s", msg.Data.GetHashPayload())
	for _, socket := range listSocket {
		log.Trace().Msgf("Try to send a msg to  %d", socket.id)
		socket.chanMsg <- msg
	}
}
