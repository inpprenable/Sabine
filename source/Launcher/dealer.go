package Launcher

import (
	"encoding/gob"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Blockchain/Socket"
	"time"
)

const nbTry int = 180

type DealerArg struct {
	BaseArg
	Contact       string
	NbOfNode      int
	IncomePort    string
	RandomDistrib bool
}

func Dealer(arg DealerArg) {
	var continu = true
	var err error
	var mySender sender
	var listContact []string
	var nContact int

	arg.init()
	go stopSleep(arg.interruptChan, &continu, nil, nil)

	if arg.NbOfNode > 0 {
		var try int = nbTry
		for try > 0 && continu && nContact < arg.NbOfNode {
			_, nContact, listContact = getAContact(arg.Contact)
			fmt.Printf("%d nodes are connected to the bootstrap server", nContact)
			if nContact < arg.NbOfNode {
				time.Sleep(1 * time.Second)
				log.Info().Msgf("Not enough validator : get %d instead of %d", nContact, arg.NbOfNode)
			}
			try--
			if try == 0 && nContact < arg.NbOfNode {
				log.Fatal().Msg("Too much test to boot to bootstrap")
			}
		}
		log.Info().Msgf("Received enough validator : get %d on %d", nContact, arg.NbOfNode)
	} else {
		_, nContact, listContact = getAContact(arg.Contact)
		fmt.Printf("%d nodes are connected to the bootstrap server", nContact)
	}
	Socket.GobLoader()

	mySender = newMultiSend(listContact, !arg.RandomDistrib)

	go createSocker(&arg, mySender)

	<-arg.interruptChan

	mySender.close()
	arg.close()
	check(err)
}

func createSocker(arg *DealerArg, send sender) {
	listenSocket, err := net.Listen(connType, addr+":"+arg.IncomePort)
	if err != nil {
		log.Panic().Msgf("Socket Error init: %s", err.Error())
	}
	for {
		conn, err := listenSocket.Accept()
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				log.Error().Msg("Cannot close the connection")
			}
		}(conn)
		if err != nil {
			log.Error().Msgf("Socket Error init: %s", err.Error())
		}
		idIcoming := Socket.ExchangeIdServer(nil, conn)
		if idIcoming != -1 {
			err := conn.Close()
			if err != nil {
				log.Error().Msg("A node which is not a client tried to connect")
			}
		} else {
			var message Blockchain.Message
			decoder := gob.NewDecoder(conn)
			for {
				err = decoder.Decode(&message)
				if err == io.EOF {
					log.Info().Msg("Deconnection of the client")
					break
				}
				check(err)
				transaction := message.Data.(Blockchain.Transaction)
				if arg.RandomDistrib {
					send.Add(1)
				} else {
					send.Add(arg.NbOfNode)
				}
				go func() {
					send.send(transaction, message.Priority, Blockchain.DontBroadcast)
				}()
			}
		}
	}
}
