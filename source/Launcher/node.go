package Launcher

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Blockchain/Consensus"
	"pbftnode/source/Blockchain/Socket"
	"pbftnode/source/config"
	"strconv"
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
	HttpChain     string
	HttpMetric    string
	Param         Blockchain.ConsensusParam
	PPRof         bool
	Control       bool
	ControlType   string
	Sleep         int
	RegularSave   int
	DelayParam    DelayParam
}

// DelayParam Contains all the parameters referencing the configuration of the delay between nodes
type DelayParam struct {
	DelayType  string
	AvgDelay   int
	StdDelay   int
	matAdj     [][]int
	MatAdjPath string
}

func Node(arg NodeArg) {
	var srv *http.Server
	arg.init()
	var Saver *Blockchain.Saver
	defLoggerPanic()
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

	arg.loadMatrix()
	consensus := Consensus.NewPBFTStateConsensus(wallet, arg.NodeNumber, arg.Param)

	delay := Socket.NewNodeDelay(createDelay(arg.DelayParam, consensus.GetId()), true)
	var comm = Socket.NewNetSocketBoot(consensus, arg.BootAddr, arg.ListeningPort, delay)
	consensus.SetSocketHandler(comm)
	comm.InitBootstrapedCo()
	if arg.Control {
		consensus.SetControlInstruction(true)
	}
	if arg.HttpMetric != "" {
		consensus.SetHTTPViewer(arg.HttpMetric)
	}
	log.Printf("the expected first proposer is %d\n", consensus.BlockChain.GetProposerId())
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
			for range ticker.C {
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

func createDelay(arg DelayParam, nodeId int) Socket.DelayConfig {
	delayType := Socket.ParseDelayType(arg.DelayType)
	var matrix []int = nil
	if arg.matAdj != nil {
		matrix = arg.matAdj[nodeId]
	}

	return Socket.DelayConfig{
		DelayType: delayType,
		AvgDelay:  arg.AvgDelay,
		StdDelay:  arg.StdDelay,
		Matrix:    matrix,
	}
}

// loadMatrix load the csv file containing the delay between nodes
func (param *DelayParam) loadMatrix() {
	if param.MatAdjPath != "" {
		param.matAdj = readFullMatrix(param.MatAdjPath)
	}
}

// loadMatrix load the csv file containing the delay between nodes
func (param *NodeArg) loadMatrix() {
	if param.DelayParam.MatAdjPath != "" {
		param.DelayParam.matAdj = readFullMatrix(param.DelayParam.MatAdjPath)
		param.Param.SelectorArgs.MatAdj = param.DelayParam.matAdj
	}
}

func readFullMatrix(path string) [][]int {
	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatal().Msgf("Cannot open the CSV file, %s", err.Error())
	}
	csvlines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		log.Fatal().Msgf("Cannot read the CSV file, %s", err.Error())
	}
	nb_node := -1
	var lineMatrix [][]int
	for index, csvline := range csvlines {
		if nb_node == -1 {
			nb_node = len(csvline)
			lineMatrix = make([][]int, nb_node)
		}
		lineMatrix[index] = make([]int, nb_node)
		for i, s := range csvline {
			delay, errVal := strconv.Atoi(s)
			if errVal != nil {
				log.Fatal().Msgf("One of the value cannot be cast : %s", s)
			}
			lineMatrix[index][i] = delay
		}
	}
	err = csvFile.Close()
	check(err)
	log.Debug().Msg("Successfully import the adjacency matrix")
	return lineMatrix
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

func defLoggerPanic() {
	defer func() {
		if r := recover(); r != nil {
			log.Panic().Msgf("Panic: %v", r)
			os.Exit(1)
		}
	}()
}
