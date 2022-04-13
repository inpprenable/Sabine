package Consensus

import (
	"encoding/base64"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"pbftnode/source/Blockchain"
	"time"
)

const ChannelSize = 64

type PBFTStateConsensus struct {
	CoreConsensus
	state      consensusState
	stateFonct stateInterf
	//checkStateChange func(message *Blockchain.Message)
	currentHash      []byte
	BlockPoolNV      *Blockchain.BlockPool
	RamOpt           bool
	Control          *Blockchain.ControlFeedBack
	toKill           chan chan struct{}
	chanReceivMsg    chan Blockchain.Message
	chanPrioInMsg    chan Blockchain.Message
	chanTxMsg        chan Blockchain.Message
	chanUpdateStatus chan Blockchain.Message
}

func (consensus *PBFTStateConsensus) GetBlockchain() *Blockchain.Blockchain {
	return consensus.BlockChain
}

func (consensus *PBFTStateConsensus) GetTransactionPool() Blockchain.TransactionPoolInterf {
	return consensus.TransactionPool
}

func NewPBFTStateConsensus(wallet *Blockchain.Wallet, numberOfNode int, param Blockchain.ConsensusParam) *PBFTStateConsensus {
	consensus := &PBFTStateConsensus{
		CoreConsensus: *NewCoreConsensus(wallet, numberOfNode, ConsensusArgs{
			param.MetricSaveFile,
			param.TickerSave,
			param.ControlType,
			param.Behavior,
			param.RefreshingPeriod,
		}),
		state:            NewRoundSt,
		toKill:           make(chan chan struct{}),
		chanReceivMsg:    make(chan Blockchain.Message, ChannelSize),
		chanTxMsg:        make(chan Blockchain.Message, ChannelSize),
		chanPrioInMsg:    make(chan Blockchain.Message, 128),
		chanUpdateStatus: make(chan Blockchain.Message, 1),
		BlockPoolNV:      Blockchain.NewBlockPool(),
	}
	if consensus.IsProposer() {
		consensus.state = NewRoundProposerSt
	}
	consensus.stateFonct = consensus.updateStateFct()
	consensus.Broadcast = param.Broadcast
	consensus.PoANV = param.PoANV
	consensus.RamOpt = !param.Broadcast && param.RamOpt
	go consensus.MsgHandlerGoroutine(consensus.chanUpdateStatus, consensus.chanReceivMsg, consensus.toKill)
	go consensus.transactionHandler(consensus.chanUpdateStatus, consensus.chanTxMsg, consensus.TransactionPool.GetStopingChan())
	consensus.Control = Blockchain.NewControlFeedBack(consensus.Metrics, consensus, param.ModelFile, param.ControlPeriod)
	return consensus
}

func (consensus *PBFTStateConsensus) SetControlInstruction(instruct bool) {
	consensus.Control.SetInstruction(instruct)
}

func (consensus *PBFTStateConsensus) updateState(state consensusState) {
	if zerolog.GlobalLevel() <= zerolog.TraceLevel {
		log.Trace().
			Int("ID", consensus.GetId()).
			Str("Old State", consensus.state.String()).
			Str("New State", state.String()).
			Int("Proposer", consensus.GetProposerId()).
			Str("ref", base64.StdEncoding.EncodeToString(consensus.currentHash)).
			Int64("at", time.Now().UnixNano()).
			Msg("Consensus State Change")
	}
	consensus.state = state
}

func (consensus *PBFTStateConsensus) MessageHandler(message Blockchain.Message) {
	if message.Priority {
		consensus.chanPrioInMsg <- message
	} else {
		if message.Flag == Blockchain.TransactionMess {
			consensus.chanTxMsg <- message
		} else {
			consensus.chanReceivMsg <- message
		}
	}
}

func (consensus *PBFTStateConsensus) transactionHandler(chanUpdateStatus chan<- Blockchain.Message, channelTx <-chan Blockchain.Message, stopAddTxChan <-chan bool) {
	var addTx = true
	var cmpt int
	for {
		select {
		case msg := <-channelTx:
			transac := msg.Data.(Blockchain.Transaction)
			if addTx || transac.IsCommand() {
				consensus.receiveTransacMess(msg)
				select {
				case chanUpdateStatus <- msg:
				default:
				}
			} else {
				cmpt++
			}
		case addTx = <-stopAddTxChan:
			if addTx {
				log.Warn().Int64("at", time.Now().UnixNano()).
					Msgf("%d messages were dropped", cmpt)
				cmpt = 0
			} else {
				log.Warn().Int64("at", time.Now().UnixNano()).
					Msgf("All new Transactions will be dropped")
			}
		}
	}
}

func (consensus *PBFTStateConsensus) MsgHandlerGoroutine(chanUpdateStatus chan Blockchain.Message, chanReceivMsg <-chan Blockchain.Message, toKill <-chan chan struct{}) {
	for {
		select {
		case message := <-consensus.chanPrioInMsg:
			consensus.handleOneMsg(message, chanUpdateStatus)
		default:
		}
		select {
		case message := <-consensus.chanPrioInMsg:
			consensus.handleOneMsg(message, chanUpdateStatus)
		case message := <-chanReceivMsg:
			consensus.handleOneMsg(message, chanUpdateStatus)
		case message := <-chanUpdateStatus:
			consensus.updateStatusWithMsg(message)
		case channel := <-toKill:
			channel <- struct{}{}
			return
		}
	}
}

func (consensus *PBFTStateConsensus) handleOneMsg(message Blockchain.Message, chanUpdateStatus chan<- Blockchain.Message) {
	consensus.receivedMessage(message)
	select {
	case chanUpdateStatus <- message:
	default:
	}
}

func (consensus *PBFTStateConsensus) updateStatusWithMsg(message Blockchain.Message) {
	oldState := consensus.state
	oldSize := consensus.GetSeqNb()
	admess := &message
	for ok := true; ok; {
		admess = consensus.checkStateChange(admess)
		ok = oldState != consensus.state || oldSize != consensus.GetSeqNb()
		oldState = consensus.state
		oldSize = consensus.GetSeqNb()
	}
}

type stateInterf interface {
	testState(Blockchain.Message) bool
	giveMeNext(*Blockchain.Message) Blockchain.Payload
	update(payload Blockchain.Payload) *Blockchain.Message
	//RemoveOld(blockSeq int)
}

type consensusState int8

const (
	NewRoundSt consensusState = iota
	NewRoundProposerSt
	PrePreparedSt
	PreparedSt
	CommittedSt
	FinalCommittedSt
	RoundChangeSt
	NVRoundSt
)

func (id consensusState) String() string {
	switch id {
	case NewRoundSt:
		return "New Round"
	case NewRoundProposerSt:
		return "New Round Proposer"
	case PrePreparedSt:
		return "Pre-Prepared"
	case PreparedSt:
		return "Prepared"
	case CommittedSt:
		return "Committed"
	case FinalCommittedSt:
		return "Final Committed"
	case RoundChangeSt:
		return "Round Change"
	case NVRoundSt:
		return "NV Round Change"
	default:
		return "Unknown state"
	}
}

func (consensus *PBFTStateConsensus) updateStateFct() stateInterf {
	switch consensus.state {
	case NewRoundSt:
		return &NewRound{consensus}
	case NewRoundProposerSt:
		return &NewRoundProposer{consensus}
	case PreparedSt:
		return &Prepared{consensus}
	case CommittedSt:
		return &Committed{consensus}
	case RoundChangeSt:
		return nil
	case NVRoundSt:
		return &NewRoundNV{consensus}
	default:
		log.Error().Msgf("Unknown State")
		return nil
	}
}

func (consensus *PBFTStateConsensus) receivedMessage(message Blockchain.Message) {
	if !consensus.Validators.IsValidator(message.Data.GetProposer()) {
		return
	}
	consensus.logTrace(message, consensus.isActiveValidator())
	switch message.Flag {
	case Blockchain.TransactionMess:
		consensus.receiveTransacMess(message)
		break
	case Blockchain.PrePrepare:
		consensus.receivePrePrepareMessage(message)
		break
	case Blockchain.PrepareMess:
		consensus.receivePrepareMessage(message)
		break
	case Blockchain.CommitMess:
		consensus.receiveCommitMessage(message)
		break
	case Blockchain.RoundChangeMess:
		consensus.receiveRCMessage(message)
		break
	case Blockchain.BlocMsg:
		consensus.receiveBlockMsg(message)
		break
	default:
		log.Warn().Msg("The Message is not recognize")
	}
}

func (consensus *PBFTStateConsensus) receiveTransacMess(message Blockchain.Message) {
	transac := message.Data.(Blockchain.Transaction)
	if !consensus.TransactionPool.ExistingTransaction(transac) && !consensus.BlockChain.ExistTx(transac.Hash) &&
		transac.VerifyTransaction() &&
		(!transac.IsCommand() || transac.VerifyAsCommandShort(consensus.Validators)) {
		consensus.ReceiveTrustedMess(message)
	}
}

func (consensus *PBFTStateConsensus) ReceiveTrustedMess(message Blockchain.Message) {
	transac := message.Data.(Blockchain.Transaction)
	success := consensus.TransactionPool.AddTransaction(transac)
	if !success {
		log.Error().Msgf("The transactionPool is full")
	}
	if message.ToBroadcast == Blockchain.AskToBroadcast {
		consensus.SocketHandler.BroadcastMessage(Blockchain.Message{
			Flag:        Blockchain.TransactionMess,
			Data:        transac,
			ToBroadcast: Blockchain.DontBroadcast,
			Priority:    message.Priority,
		})
	} else if (message.ToBroadcast == Blockchain.DefaultBehavour) && !consensus.IsProposer() {
		consensus.SocketHandler.TransmitTransaction(message)
	}
}

func (consensus *PBFTStateConsensus) receivePrePrepareMessage(message Blockchain.Message) {
	var block = message.Data.(Blockchain.Block)
	if !consensus.BlockPool.ExistingBlock(block) && block.VerifyBlock() && block.SequenceNb > consensus.BlockChain.GetCurrentSeqNb() {
		// add block to pool
		consensus.BlockPool.AddBlock(block)
		//send to other nodes
		if consensus.Broadcast {
			consensus.SocketHandler.BroadcastMessage(message)
		}
	}
}

func (consensus *PBFTStateConsensus) receivePrepareMessage(message Blockchain.Message) {
	var prepare = message.Data.(Blockchain.Prepare)
	// check if the prepare Message is valid
	// The existence of the block in the blockpool is equivalent to the existence of an assumed Prepare Message from the node
	if !consensus.PreparePool.ExistingPrepare(prepare) && prepare.IsValidPrepare() {
		// add prepare Message to the pool
		consensus.PreparePool.AddPrepare(prepare)
		// send to other nodes
		if consensus.Broadcast {
			consensus.SocketHandler.BroadcastMessage(message)
		}
		if consensus.RamOpt && consensus.PreparePool.IsFullValidated(string(prepare.BlockHash)) &&
			(consensus.CommitPool.HaveCommitFrom(prepare.BlockHash, &consensus.Wallet) ||
				consensus.BlockChain.ExistBlockOfHash(prepare.BlockHash)) {
			consensus.PreparePool.Remove(prepare.BlockHash)
		}
	}
}

func (consensus *PBFTStateConsensus) receiveCommitMessage(message Blockchain.Message) {
	var commit = message.Data.(Blockchain.Commit)
	// check the validity commit messages
	if !consensus.CommitPool.ExistingCommit(commit) && commit.IsValidCommit() {
		// add to pool
		consensus.CommitPool.AddCommit(commit)

		//send to other nodes
		if consensus.Broadcast {
			consensus.SocketHandler.BroadcastMessage(message)
		}
		if consensus.RamOpt && consensus.CommitPool.IsFullValidated(string(commit.BlockHash)) &&
			consensus.BlockChain.ExistBlockOfHash(commit.BlockHash) {
			consensus.CommitPool.Remove(commit.BlockHash)
		}
	}
}

func (consensus *PBFTStateConsensus) receiveRCMessage(message Blockchain.Message) {
	var round = message.Data.(Blockchain.RoundChange)
	// check the validity of the round change Message
	if !consensus.MessagePool.ExistingMessage(round) && round.IsValidMessage() {
		// add to pool
		consensus.MessagePool.AddMessage(round)
		//send to other nodes
		if consensus.Broadcast {
			consensus.SocketHandler.BroadcastMessage(message)
		}
	}
}

func (consensus *PBFTStateConsensus) receiveBlockMsg(message Blockchain.Message) {
	var block = message.Data.(Blockchain.BlockMsg)
	//log.WithFields(log.Fields{"SeqNb": block.Block.SequenceNb}).Debug("Receive The MsgBlock")
	log.Debug().
		Int("SeqNb", block.Block.SequenceNb).
		Msg("Receive The MsgBlock")
	if !consensus.BlockPoolNV.ExistingBlock(block.Block) && block.Block.VerifyBlock() {
		// add block to pool
		consensus.BlockPoolNV.AddBlock(block.Block)
	}
}

func (consensus *PBFTStateConsensus) checkStateChange(message *Blockchain.Message) *Blockchain.Message {
	stateChangePayload := consensus.stateFonct.giveMeNext(message)
	var emitted *Blockchain.Message
	if stateChangePayload != nil {
		emitted = consensus.stateFonct.update(stateChangePayload)
		consensus.stateFonct = consensus.updateStateFct()
	}
	return emitted
}

func (consensus *PBFTStateConsensus) updateStateAfterCommit() {
	if !(consensus.state == FinalCommittedSt || consensus.state == NVRoundSt) {
		log.Error().Msgf("Wrong state expected, get :%s", consensus.state)
	}
	if consensus.IsPoANV() && !consensus.isActiveValidator() {
		consensus.updateState(NVRoundSt)
	} else if consensus.IsProposer() {
		consensus.updateState(NewRoundProposerSt)
	} else {
		consensus.updateState(NewRoundSt)
	}
}

func (consensus *PBFTStateConsensus) Close() {
	consensus.Metrics.Close()
	query := make(chan struct{})
	consensus.toKill <- query
	<-query
	close(query)
	close(consensus.toKill)
	close(consensus.chanReceivMsg)
}

func (consensus PBFTStateConsensus) GetIncQueueSize() int {
	return len(consensus.chanReceivMsg) + len(consensus.chanPrioInMsg)
}
