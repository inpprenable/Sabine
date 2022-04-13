package Launcher

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Blockchain/Consensus"
	"pbftnode/source/Blockchain/Socket"
	"pbftnode/source/config"
	"time"
)

type NodeArg struct {
	BaseArg
	BootAddr      string
	NodeId        string
	NodeNumber    int
	SaveFile      string
	MultiSaveFile bool
	ListeningPort string
	AvgDelay      int
	StdDelay      int
	HttpChain     string
	HttpMetric    string
	Param         Blockchain.ConsensusParam
	PPRof         bool
	Control       bool
	ControlType   string
	Sleep         int
	RegularSave   int
	DelayType     string
}

func Node(arg NodeArg) {
	var srv *http.Server
	arg.init()
	var Saver *Blockchain.Saver
	defer arg.close()

	if arg.Sleep != 0 {
		time.Sleep(time.Duration(arg.Sleep) * time.Millisecond)
	}

	if arg.PPRof {
		go func() {
			log.Print(http.ListenAndServe("[::]:6060", nil))
		}()
	}

	var wallet = Blockchain.NewWallet(fmt.Sprintf("NODE%s", arg.NodeId))
	consensus := Consensus.NewPBFTStateConsensus(wallet, arg.NodeNumber, arg.Param)
	delay := Socket.NewNodeDelay(createDelay(arg), true)
	var comm = Socket.NewNetSocketBoot(consensus, arg.BootAddr, arg.ListeningPort, delay)
	consensus.SetSocketHandler(comm)
	comm.InitBootstrapedCo()
	if arg.Control {
		consensus.SetControlInstruction(true)
	}
	if arg.HttpMetric != "" {
		consensus.SetHTTPViewer(arg.HttpMetric)
	}
	log.Printf("the expected first proposer is %d\n", consensus.BlockChain.GetProposerNumber())
	if arg.HttpChain != "" {
		srv = Blockchain.HttpBlockchainViewer(consensus.BlockChain, arg.HttpChain)
	}
	Saver = Blockchain.NewSaver(arg.SaveFile, consensus)
	if arg.MultiSaveFile {
		consensus.BlockChain.Save = func() {
			Saver.AskMultiSave()
		}
	} else {
		consensus.BlockChain.Save = func() {
			Saver.AskToSave()
		}
	}
	if arg.RegularSave > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(arg.RegularSave) * time.Minute)
			for _ = range ticker.C {
				consensus.BlockChain.Save()
			}
		}()
	}

	<-arg.interruptChan
	consensus.BlockChain.Save()
	Saver.Wait()
	end(srv, comm)
	consensus.Close()
}

func openChainFile(saveFile string) (chainFile *os.File) {
	var err error
	try := config.CreateTest
	for try > 0 && chainFile == nil {
		chainFile, err = os.Create(saveFile)
		try--
	}
	if try == 0 && err != nil {
		log.Error().Msgf("Error opening the file %s : %s", saveFile, err.Error())
		return nil
	}
	return chainFile
}

func createDelay(arg NodeArg) Socket.ProbaDelay {

	delayType := Socket.StrToDelayType(arg.DelayType)
	switch delayType {
	case Socket.NoDelaySt:
		return Socket.NoDelay{}
	case Socket.PoissonDelaySt:
		return Socket.NewPoissonDelay(float64(arg.AvgDelay))
	case Socket.NormalDelaySt:
		return Socket.NewNormalDelay(float64(arg.AvgDelay), float64(arg.StdDelay))
	case Socket.FixeDelaySt:
		return Socket.NewFixeDelay(float64(arg.AvgDelay))
	default:
		return Socket.NoDelay{}
	}
}

func end(srv *http.Server, comm *Socket.NetSocketBoot) {
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
	comm.Close()
	//fmt.Println("adios")
}
