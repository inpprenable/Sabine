package Blockchain

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"time"
)

type Input interface {
	inputToByte() ([]byte, error)
}

type transactCore struct {
	Ids       uuid.UUID         `json:"ids"`
	From      ed25519.PublicKey `json:"from"`
	Input     Input             `json:"Data"`
	Timestamp int64             `json:"timestamp"`
}

type BrutData struct {
	Data []byte `json:"data"`
}

func (in BrutData) inputToByte() ([]byte, error) {
	return json.Marshal(in)
}

type Transaction struct {
	Hash       []byte       `json:"hash"`
	Signature  []byte       `json:"signature"`
	TransaCore transactCore `json:"transaCore"`
}

func NewTransaction(in Input, wallet Wallet) (transac *Transaction) {
	transacCore := transactCore{
		Ids:       Ids(),
		From:      wallet.PublicKey(),
		Input:     in,
		Timestamp: time.Now().UnixNano(),
	}
	inputByte := transacCore.ToByte()
	hash := Hash(inputByte)
	transac = &Transaction{
		Hash:       hash,
		Signature:  wallet.Sign(hash),
		TransaCore: transacCore,
	}
	return
}

func NewBruteTransaction(data []byte, wallet Wallet) *Transaction {
	return NewTransaction(BrutData{Data: data}, wallet)
}

func (transaction Transaction) VerifyTransaction() bool {
	hashedData := transaction.TransaCore.ToByte()
	return VerifySignature(transaction.TransaCore.From, Hash(hashedData), transaction.Signature)
}

func (transacCore transactCore) ToByte() []byte {
	byted, err := json.Marshal(transacCore)
	if err != nil {
		log.Fatal().Msgf("Json Parsing error : %s", err)
	}
	return byted
}

func (transaction Transaction) ToByte() []byte {
	byted, err := json.Marshal(transaction)
	if err != nil {
		log.Fatal().Msgf("Json Parsing error : %s", err)
	}
	return byted
}

func ByteToTx(data []byte) (transaction Transaction) {
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		log.Fatal().Msgf("Json Parsing error : %s", err)
	}
	return
}

type CommandeType int8

const (
	VarieValid CommandeType = iota
	ChangeDelay
)

type Commande struct {
	Order     CommandeType `json:"order"`
	Variation int          `json:"variation"`
	// Possibility to create other fields
}

func (c Commande) inputToByte() ([]byte, error) {
	return json.Marshal(c)
}

func (transaction Transaction) IsCommand() bool {
	_, ok := transaction.TransaCore.Input.(Commande)
	return ok
}

// Verify the validity of a Commande and return if it's valid and its error
func (transac Transaction) verifyAsCommand(validators ValidatorInterf) (bool, error) {
	commande, ok := transac.TransaCore.Input.(Commande)
	if !ok {
		return false, errors.New("NOT A COMMAND")
	}
	if !validators.IsActiveValidator(transac.TransaCore.From) {
		return false, errors.New("NOT EMITTED BY A VALIDATOR")
	}
	switch commande.Order {
	case VarieValid:
		newSize := validators.GetNumberOfValidator() + commande.Variation
		if !validators.IsSizeValid(newSize) {
			return false, fmt.Errorf("NEW SIZE OUT OF BAND: %d", newSize)
		}
	case ChangeDelay:
		if commande.Variation < 0 {
			return false, errors.New("NO NEGATIVE DELAY")
		}
	default:
		return false, errors.New("UNKNOWN COMMAND ID")
	}
	return true, nil
}

//VerifyAsCommandShort Remove the error From the Check of verifyAsCommand
func (transac Transaction) VerifyAsCommandShort(validators ValidatorInterf) bool {
	ok, err := transac.verifyAsCommand(validators)
	if err != nil {
		log.Error().Msg(err.Error())
	}
	return ok
}

func (transaction Transaction) GetHashPayload() string {
	return base64.StdEncoding.EncodeToString(transaction.Hash)
}

func (transac Transaction) GetProposer() ed25519.PublicKey {
	return transac.TransaCore.From
}
