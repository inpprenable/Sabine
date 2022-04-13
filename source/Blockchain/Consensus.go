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
	Broadcast        bool
	PoANV            bool
	RamOpt           bool
	MetricSaveFile   string
	TickerSave       int
	ControlType      ControlType
	ControlPeriod    int
	RefreshingPeriod int
	Behavior         OverloadBehavior
	ModelFile        string
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
	GetId() int
	GetProposerId() int
	GetSeqNb() int
	MinApprovals() int
	Close()
	SetHTTPViewer(port string)
	IsProposer() bool
	ReceiveTrustedMess(message Message)
	SetControlInstruction(instruct bool)
	GetControl() ControlType
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
	DefaultBehavour BroadcastType = iota
	AskToBroadcast
	DontBroadcast
)
