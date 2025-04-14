package Blockchain

import (
	"crypto/ed25519"
)

type MessageType int8

const NbTypeMess = 6
const (
	TransactionMess MessageType = iota + 1
	PrePrepare
	PrepareMess
	CommitMess
	RoundChangeMess
	BlocMsg
)

type ConsensusParam struct {
	Broadcast           bool
	PoANV               bool
	RamOpt              bool
	MetricSaveFile      string
	TickerSave          int
	ControlType         ControlType
	ControlPeriod       int
	RefreshingPeriod    int
	Behavior            OverloadBehavior
	ModelFile           string
	SelectionType       SelectionValidatorType
	SelectorArgs        ArgsSelector
	AcceptTxFromUnknown bool
}

// String Return the string associate to MessageType
func (id MessageType) String() string {
	switch id {
	case TransactionMess:
		return "Transaction"
	case PrePrepare:
		return "PrePrepare"
	case PrepareMess:
		return "Prepare"
	case CommitMess:
		return "Commit"
	case RoundChangeMess:
		return "RoundChange"
	case BlocMsg:
		return "Block"
	default:
		return "unknown"
	}
}

type Consensus interface {
	ValidatorGetterInterf
	MakeTransaction(Commande) *Transaction
	IsPoANV() bool
	MessageHandler(message Message)
	// GetId returns the id of the current node
	GetId() int
	GetProposer() ed25519.PublicKey
	GetSeqNb() int
	MinApprovals() int
	Close()
	SetHTTPViewer(port string)
	IsProposer() bool
	ReceiveTrustedMess(message Message)
	SetControlInstruction(instruct bool)
	GetControl() ControlType
	GenerateNewValidatorListProposition(newSize int) []ed25519.PublicKey
	IsActiveValidator(key ed25519.PublicKey) bool
	GetPubKeyofId(int) ed25519.PublicKey
}

type writeChainInterf interface {
	GetBlockchain() *Blockchain
}

type TestConsensus interface {
	Consensus
	GetBlockchain() *Blockchain
	GetValidator() ValidatorInterf
	GetTransactionPool() TransactionPoolInterf
	GetBlockPool() *BlockPool
}

type Message struct {
	Priority    bool
	Flag        MessageType `json:"flag"`
	Data        Payload     `json:"data"`
	ToBroadcast BroadcastType
}

type Payload interface {
	ToByte() []byte
	GetHashPayload() string
	GetProposer() ed25519.PublicKey
}

type BroadcastType uint8

const (
	// DefaultBehaviour the transaction is transmitted to the proposer
	DefaultBehaviour BroadcastType = iota
	// AskToBroadcast the transaction is transmitted to all nodes
	AskToBroadcast
	// DontBroadcast the transaction is not transmitted to other nodes
	DontBroadcast
)
