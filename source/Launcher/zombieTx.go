package Launcher

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Blockchain/Socket"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ZombieTxArg struct {
	ZombieArg
	Scenario         []Scenarii
	DelayScenarioStr string
	DelayScenario    []Scenarii
	Multi            bool
}

type Scenarii struct {
	NbTxPS   int
	Duration int
}

func ZombieTx(arg ZombieTxArg) {
	var continu = true
	var waitGroup sync.WaitGroup
	var err error
	var mySender sender
	var listContact []string
	var nContact int

	arg.init()

	if len(arg.DelayScenarioStr) != 0 {
		var err error
		arg.DelayScenario, err = handleDelayScenario(arg.DelayScenarioStr)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
	}

	Contact := arg.Contact
	if arg.ByBootstrap {
		for nContact < arg.NbOfNode {
			Contact, nContact, listContact = getAContact(arg.Contact)
			fmt.Printf("%d nodes are connected to the bootstrap server", nContact)
			if nContact < arg.NbOfNode {
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
	Socket.GobLoader()

	if arg.Multi {
		mySender = newMultiSend(listContact, false)
	} else {
		mySender = newContacts(Contact, listContact)
	}

	//conn := connectWith(Contact)
	//listContact.conn = &conn

	go stopSleep(arg.interruptChan, &continu, nil, nil)
	//encoder := gob.NewEncoder(conn)
	wallet := Blockchain.NewWallet("NODE" + arg.NodeID)

	mySender.Add(1)
	mySender.send(*wallet.CreateTransaction(Blockchain.Commande{
		Order:     Blockchain.VarieValid,
		Variation: -arg.Reducing,
	}), true, Blockchain.AskToBroadcast)
	mySender.Wait()

	//sendTx(*wallet.CreateTransaction(Blockchain.Commande{
	//	Order:     Blockchain.VarieValid,
	//	Variation: -arg.Reducing,
	//}), encoder, &listContact, false)
	time.Sleep(500 * time.Millisecond)
	fmt.Println("The number of validator should be reduced")
	timeStart := time.Now()
	go delayRoutine(mySender, arg.DelayScenario, wallet)
	for y, scenarii := range arg.Scenario {
		fmt.Printf("Experiment %d, %d Tx/s during %d s\n", y, scenarii.NbTxPS, scenarii.Duration)
		if scenarii.NbTxPS == 0 {
			fmt.Printf("Wait %d s", scenarii.Duration)
			time.Sleep(time.Duration(scenarii.Duration) * time.Second)
		} else {
			var waitSleep = time.Duration(1e9/float64(scenarii.NbTxPS)) * time.Nanosecond
			var nbTx = scenarii.Duration * scenarii.NbTxPS
			onePercent := nbTx / 100
			fmt.Printf("\rSend of transactions :\t%d\t/%d", 0, nbTx)
			ticker := time.NewTicker(waitSleep)
			for i := 0; continu && i < nbTx; i++ {
				if (i+1)%onePercent == 0 {
					fmt.Printf("\rSend of transactions :\t%d\t/%d", i+1, nbTx)
				}
				transac := wallet.CreateBruteTransaction([]byte(fmt.Sprintf("Transaction number n°%d", i)))
				mySender.Add(1)
				go func() {
					mySender.send(*transac, false, Blockchain.AskToBroadcast)
				}()
				<-ticker.C
			}
			ticker.Stop()
		}
		fmt.Println("")
	}

	fmt.Println("\nThe experiment is finished")
	t1 := time.Now()
	fmt.Printf("It took %f\n", t1.Sub(timeStart).Seconds())
	waitGroup.Wait()
	fmt.Printf("It took %f additional seconds\n", time.Now().Sub(t1).Seconds())
	//err = conn.Close()
	mySender.close()
	arg.close()
	check(err)
}

func connectWith(Contact string) (net.Conn, int) {
	log.Print("Try to connect with ", Contact)
	var conn net.Conn
	var err error
	var cmptErr int
	for conn == nil && cmptErr < 10 {
		conn, err = net.Dial("tcp", Contact)
		if err != nil {
			log.Error().Msgf("Error connecting: %s", err.Error())
			time.Sleep(100 * time.Millisecond)
			cmptErr++
		}
	}
	if cmptErr == 3 {
		log.Fatal().Msgf("Cannot connect with: %s", err.Error())
	}
	log.Debug().Msg("Connected")
	id := Socket.ExchangeIdClient(nil, conn)
	log.Debug().Msg("Id are exchanged")
	return conn, id
}

func handleDelayScenario(scenarioStr string) ([]Scenarii, error) {
	var delayScenarii []Scenarii
	scenarioStr = strings.ReplaceAll(scenarioStr, "\"", "")
	for _, word := range strings.Split(scenarioStr, " ") {
		if len(word) != 0 {
			subworld := strings.Split(word, ":")
			if len(subworld) > 2 {
				errorString := fmt.Sprintf("the world %s doesn't respect this format \"delay:duration\"", word)
				return nil, errors.New(errorString)
			}
			var scenarii Scenarii
			var err error
			if len(subworld) == 1 {
				scenarii.NbTxPS, err = strconv.Atoi(subworld[0])
				if err != nil {
					return nil, errors.New(fmt.Sprintf("the world %s is not an int", word))
				}
				scenarii.Duration = -1
			} else {
				scenarii.NbTxPS, err = strconv.Atoi(subworld[0])
				if err != nil {
					return nil, errors.New(fmt.Sprintf("the world %s is not at the format \"int:int\"", word))
				}
				scenarii.Duration, err = strconv.Atoi(subworld[1])
				if err != nil {
					return nil, errors.New(fmt.Sprintf("the world %s is not at the format \"int:int\"", word))
				}
			}
			delayScenarii = append(delayScenarii, scenarii)
		}
	}
	return delayScenarii, nil
}

func delayRoutine(mySender sender, delayScenario []Scenarii, wallet *Blockchain.Wallet) {
	for _, scenario := range delayScenario {
		mySender.Add(1)
		mySender.send(*wallet.CreateTransaction(Blockchain.Commande{
			Order:     Blockchain.ChangeDelay,
			Variation: scenario.NbTxPS,
		}), true, Blockchain.AskToBroadcast)
		if scenario.Duration < 0 {
			return
		}
		time.Sleep(time.Duration(scenario.Duration) * time.Second)
	}
}
