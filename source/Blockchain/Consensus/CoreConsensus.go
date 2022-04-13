package Consensus

import (
	"bytes"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"pbftnode/source/Blockchain"
	"time"
)

type CoreConsensus struct {
	BlockChain      *Blockchain.Blockchain
	TransactionPool Blockchain.TransactionPoolInterf
	Wallet          Blockchain.Wallet
	BlockPool       *Blockchain.BlockPool
	PreparePool     Blockchain.PreparePool
	CommitPool      Blockchain.CommitPool
	MessagePool     Blockchain.MessagePool
	Validators      Blockchain.ValidatorInterf
	SocketHandler   Blockchain.Sockets
	Metrics         *Blockchain.MetricHandler
	Broadcast       bool
	PoANV           bool
	controlType     Blockchain.ControlType
}

type ConsensusArgs struct {
	MetricSaveFile   string
	TickerSave       int
	ControlType      Blockchain.ControlType
	Beahavior        Blockchain.OverloadBehavior
	RefreshingPeriod int
}

func NewCoreConsensus(wallet *Blockchain.Wallet, nbNode int, metricOpt ConsensusArgs) (retour *CoreConsensus) {
	var validator Blockchain.ValidatorInterf = &Blockchain.Validators{}
	var transactionpool = Blockchain.NewTransactionPool(validator, metricOpt.Beahavior)
	var preparePool = Blockchain.NewPreparePool()
	var commitPool = Blockchain.NewCommitPool()
	var messagePool = Blockchain.NewMessagePool()
	var consensus = CoreConsensus{
		TransactionPool: transactionpool,
		Wallet:          *wallet,
		BlockPool:       Blockchain.NewBlockPool(),
		PreparePool:     preparePool,
		CommitPool:      commitPool,
		MessagePool:     messagePool,
		Validators:      validator,
		controlType:     metricOpt.ControlType,
	}
	retour = &consensus
	var blockchain = Blockchain.NewBlockchain(validator, nil)
	retour.BlockChain = blockchain
	consensus.Validators.GenerateAddresses(nbNode)
	consensus.Metrics = Blockchain.NewMetricHandler(retour.BlockChain, validator, metricOpt.MetricSaveFile, metricOpt.TickerSave, metricOpt.RefreshingPeriod)
	consensus.BlockChain.SetMetricHandler(consensus.Metrics)
	consensus.TransactionPool.SetMetricHandler(consensus.Metrics)
	return retour
}

func (consensus CoreConsensus) GetProposerId() int {
	return consensus.BlockChain.GetProposerNumber()
}

func (consensus CoreConsensus) GetNumberOfValidator() int {
	return consensus.Validators.GetNumberOfValidator()
}

func (consensus CoreConsensus) IsPoANV() bool {
	return consensus.PoANV
}

func (consensus CoreConsensus) MinApprovals() int {
	return (consensus.Validators.GetNumberOfValidator() * 2 / 3) + 1
}

func (consensus CoreConsensus) GetId() int {
	return consensus.Validators.GetIndexOfValidator(consensus.Wallet.PublicKey())
}

// IsProposer return if the actual node is the actual proposer of the chain
func (consensus *CoreConsensus) IsProposer() bool {
	return bytes.Equal(consensus.BlockChain.GetProposer(), consensus.Wallet.PublicKey())
}

// Log when an interesting message is arriving
func (consensus CoreConsensus) logTrace(message Blockchain.Message, isValidator bool) {
	if zerolog.GlobalLevel() <= zerolog.TraceLevel {
		log.Trace().
			Str("TypeMsg", message.Flag.String()).
			Int64("at", time.Now().UnixNano()).
			Str("ref", message.Data.GetHashPayload()).
			Bool("IsValidator", isValidator).
			Msg("Message Received")
	}
}

func (consensus CoreConsensus) isActiveValidator() bool {
	return consensus.Validators.IsActiveValidator(consensus.Wallet.PublicKey())
}

func (consensus CoreConsensus) getBlockchain() *Blockchain.Blockchain {
	return consensus.BlockChain
}

func (consensus CoreConsensus) GetValidator() Blockchain.ValidatorInterf {
	return consensus.Validators
}

func (consensus CoreConsensus) getTransactionPool() Blockchain.TransactionPoolInterf {
	return consensus.TransactionPool
}

func (consensus CoreConsensus) GetSeqNb() int {
	return consensus.BlockChain.GetLenght()
}

func (consensus CoreConsensus) GetBlockPool() *Blockchain.BlockPool {
	return consensus.BlockPool
}

func (consensus CoreConsensus) GetPreparePool() *Blockchain.PreparePool {
	return &consensus.PreparePool
}

func (consensus CoreConsensus) GetWallet() *Blockchain.Wallet {
	return &consensus.Wallet
}

func (consensus CoreConsensus) GetCommitPool() *Blockchain.CommitPool {
	return &consensus.CommitPool
}

func (consensus CoreConsensus) GetMessagePool() *Blockchain.MessagePool {
	return &consensus.MessagePool
}

func (consensus *CoreConsensus) setWallet(newWallet Blockchain.Wallet) {
	consensus.Wallet = newWallet
}

func (consensus *CoreConsensus) SetHTTPViewer(port string) {
	consensus.Metrics.SetHTTPViewer(port)
}

func (consensus *PBFTStateConsensus) setWallet(newWallet Blockchain.Wallet) {
	consensus.Wallet = newWallet
	consensus.updateStateAfterCommit()
	consensus.stateFonct = consensus.updateStateFct()
}

func (consensus CoreConsensus) MakeTransaction(commande Blockchain.Commande) *Blockchain.Transaction {
	return consensus.Wallet.CreateTransaction(commande)
}

type testConsensus interface {
	Blockchain.Consensus
	getBlockchain() *Blockchain.Blockchain
	GetValidator() Blockchain.ValidatorInterf
	getTransactionPool() Blockchain.TransactionPoolInterf
	GetBlockPool() *Blockchain.BlockPool
	GetPreparePool() *Blockchain.PreparePool
	GetWallet() *Blockchain.Wallet
	GetCommitPool() *Blockchain.CommitPool
	GetMessagePool() *Blockchain.MessagePool
	setWallet(newWallet Blockchain.Wallet)
}

func (consensus *CoreConsensus) Close() {
	consensus.Metrics.Close()
}

func (consensus CoreConsensus) IsOlderThan(timestamp time.Duration) bool {
	return consensus.Validators.IsOlderThan(timestamp)
}

func (consensus CoreConsensus) GetControl() Blockchain.ControlType {
	return consensus.controlType
}

func (consensus *CoreConsensus) SetSocketHandler(sockets Blockchain.Sockets) {
	consensus.SocketHandler = sockets
	consensus.BlockChain.SocketDelay = sockets
}
