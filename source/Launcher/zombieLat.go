package Launcher

import (
	"encoding/gob"
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Blockchain/Socket"
	"time"
)

type ZombieArg struct {
	ClientArg
	Reducing int
	NbOfNode int
}

type ZombieLatArg struct {
	ZombieArg
	NbTx int
}

func ZombieLat(arg ZombieLatArg) {
	var channel_loop chan bool
	var toKill chan bool
	var continu = true

	arg.init()

	Contact := arg.Contact
	if arg.ByBootstrap {
		var nbOfNode int
		Contact, nbOfNode, _ = getAContact(arg.Contact)
		if arg.NbOfNode != 0 && nbOfNode != arg.NbOfNode {
			log.Panic().Msgf("Wrong number of node connected, expected %d, got %d", arg.NbOfNode, nbOfNode)
			arg.close()
			return
		}
	}

	log.Print("Try to connect with ", Contact)
	conn, err := net.Dial("tcp", Contact)
	if err != nil {
		log.Fatal().Msgf("Error connecting: %s", err.Error())
	}
	log.Debug().Msg("Connected")
	Socket.ExchangeIdClient(nil, conn)
	log.Debug().Msg("Id are exchanged")
	Socket.GobLoader()

	channel_loop = make(chan bool, 10)
	toKill = make(chan bool, 1)
	go stopSleep(arg.interruptChan, &continu, conn, channel_loop)
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	wallet := Blockchain.NewWallet("NODE" + arg.NodeID)
	go handleCommitReturn(decoder, arg.interruptChan, channel_loop, toKill)

	sendTxWaitCommit(wallet.CreateTransaction(Blockchain.Commande{
		Order:     Blockchain.VarieValid,
		Variation: -arg.Reducing,
	}), encoder, channel_loop)
	time.Sleep(500 * time.Millisecond)
	fmt.Println("The number of validator should be reduced")
	timeStart := time.Now()
	var toSleep time.Duration
	onePercent := arg.NbTx / 100
	for i := 0; continu && i < arg.NbTx; i++ {
		//fmt.Printf("\rSend of the transaction :\t%d\t/%d", i+1, arg.NbTxPS)
		if (i+1)%onePercent == 0 {
			fmt.Printf("\rSend of the transaction :\t%d\t/%d", i+1, arg.NbTx)
		}
		toSleep = sendTxWaitCommit(wallet.CreateBruteTransaction([]byte(fmt.Sprintf("Transaction number n°%d", i))), encoder, channel_loop)
	}
	fmt.Println("\nThe experiment is finished")
	fmt.Printf("It took %f\n", time.Now().Sub(timeStart).Seconds())
	time.Sleep(10 * toSleep)
	err = conn.Close()
	arg.close()
	<-toKill
	check(err)
	close(channel_loop)
}

func stopSleep(channel <-chan os.Signal, continu *bool, conn net.Conn, channelLoop chan<- bool) {
	<-channel
	if conn != nil {
		log.Debug().Msg("I will close")
		err := conn.Close()
		if _, ok := err.(net.Error); !ok {
			check(err)
		}
	}
	*continu = false
	log.Debug().Msgf("Continu : ", *continu)
	if channelLoop != nil {
		channelLoop <- true
	}
	log.Debug().Msg("Everything should stop")
}

func sendTxWaitCommit(transaction *Blockchain.Transaction, encoder *gob.Encoder, channel_loop chan bool) time.Duration {
	var t0, t1 time.Time
	var diff time.Duration
	var err error
	t0 = time.Now()

	err = encoder.Encode(Blockchain.Message{
		Flag:        Blockchain.TransactionMess,
		Data:        *transaction,
		ToBroadcast: Blockchain.AskToBroadcast,
	})
	log.Debug().Msgf("Transaction send %s", transaction.GetHashPayload())
	check(err)
	<-channel_loop
	t1 = time.Now()
	diff = t1.Sub(t0)
	log.Debug().Msgf("I will wait ", diff)
	//	time.Sleep(1 * diff)
	return diff
}

func handleCommitReturn(decoder *gob.Decoder, c chan os.Signal, channel_loop chan bool, toKill chan<- bool) {
	var hashMap map[string]struct{} = make(map[string]struct{})
	for len(c) == 0 {
		var message Blockchain.Message
		err := decoder.Decode(&message)
		if _, ok := err.(*net.OpError); ok {
			toKill <- true
			return
		}
		check(err)
		if message.Flag == Blockchain.CommitMess {
			log.Debug().Msgf("Commit received \t%s", message.Data.GetHashPayload())
			commitMess, _ := message.Data.(Blockchain.Commit)
			_, ok := hashMap[string(commitMess.BlockHash)]
			if !ok {
				hashMap[string(commitMess.BlockHash)] = struct{}{}
				log.Debug().Msg("I send on the channel")
				channel_loop <- true
			}
		} else {
			log.Trace().Msgf("Received : %s", message.Flag)
		}
	}
	toKill <- true
}

func check(err error) {
	if err != nil {
		log.Error().Msgf("error %t \n-> %s", err, err)
		panic(err)
	}
}
