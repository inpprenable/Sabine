package Launcher

import (
	"encoding/gob"
	"fmt"
	"github.com/rs/zerolog/log"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Blockchain/Socket"
	"sync"
	"time"
)

type ZombieRunArg struct {
	ZombieArg
	Throughput     int
	DelayPerChange int
	NbValPerChange int
}

func ZombieRun(arg ZombieRunArg) {
	var err error
	var listContact contacts
	var continu = true
	channelLoop := make(chan bool)

	arg.init()

	Contact := arg.Contact
	if arg.ByBootstrap {
		Contact, _, listContact.listContact = getAContact(arg.Contact)
	}

	conn, _ := connectWith(Contact)
	listContact.conn = &conn
	Socket.GobLoader()
	encoder := gob.NewEncoder(conn)
	wallet := Blockchain.NewWallet("NODE" + arg.NodeID)
	go stopSleep(arg.interruptChan, &continu, conn, channelLoop)

	sendTx(*wallet.CreateTransaction(Blockchain.Commande{
		Order:     Blockchain.VarieValid,
		Variation: -arg.Reducing,
	}), encoder, &listContact, true)

	toKill := make(chan chan<- struct{})
	log.Info().Msg("Start of the continu throughput")
	go sendTxAtThroughput(arg.Throughput, wallet, encoder, &listContact, toKill)

	tick := time.Duration(arg.DelayPerChange) * time.Second
	ticker := time.NewTicker(tick)

	nbStep := (arg.NbOfNode - 4) / arg.NbValPerChange
	log.Info().Msgf("Tick %d/%d s", 0, (nbStep+1)*arg.DelayPerChange)
	for i := 0; i < nbStep && continu; i++ {
		select {
		case <-ticker.C:
			log.Info().Msgf("Tick %d/%d s", (i+1)*arg.DelayPerChange, (nbStep+1)*arg.DelayPerChange)
			sendTx(*wallet.CreateTransaction(Blockchain.Commande{
				Order:     Blockchain.VarieValid,
				Variation: -arg.NbValPerChange,
			}), encoder, &listContact, true)
		case <-channelLoop:

		}

	}
	if continu {
		<-ticker.C
	}
	log.Info().Msgf("Tick %d/%d s", (nbStep+1)*arg.DelayPerChange, (nbStep+1)*arg.DelayPerChange)

	killer := make(chan struct{})
	toKill <- killer
	<-killer
	err = conn.Close()
	arg.close()
	check(err)
}

func sendTxAtThroughput(throughput int, wallet *Blockchain.Wallet, encoder *gob.Encoder, listContact *contacts, toKill <-chan chan<- struct{}) {
	tick := time.Duration(1e9/float64(throughput)) * time.Nanosecond
	ticker := time.NewTicker(tick)
	var waitGroup sync.WaitGroup
	var i int
	for {
		select {
		case <-ticker.C:
			transac := wallet.CreateBruteTransaction([]byte(fmt.Sprintf("Transaction number n°%d", i)))
			i++
			go func() {
				waitGroup.Add(1)
				sendTx(*transac, encoder, listContact, false)
				waitGroup.Done()
			}()
		case channel := <-toKill:
			waitGroup.Wait()
			channel <- struct{}{}
			return
		}
	}

}
