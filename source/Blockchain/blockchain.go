package Blockchain

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"sync"
)

type Blockchain struct {
	validatorList ValidatorInterf
	//chain         []Block
	firstBlock     *linkedBlock
	sizeChain      int
	lastBlock      *linkedBlock
	secLastBlock   *linkedBlock
	existingBlock  map[string]struct{}
	executedTx     executedTxSec
	Save           func()
	executedTxLock sync.RWMutex
	metrics        *MetricHandler
	DelayUpdater   UpdateDelay
	selector       ValidatorSelectorUpdateInter
	proposer       ed25519.PublicKey
	proposerID     int
}

type linkedBlock struct {
	block    Block
	nextLink *linkedBlock
}

func newLinkedBlock(block Block) *linkedBlock {
	return &linkedBlock{block: block}
}

func (link *linkedBlock) append(block Block) *linkedBlock {
	link.nextLink = newLinkedBlock(block)
	return link.nextLink
}

type ChainInterface interface {
	GetLastBLoc() Block
	GetSecLastBLoc() Block
	GetBlock(id int) Block
}

// NewBlockchain the constructor takes an argument validators class object
// this is used to create a list of validators
func NewBlockchain(validators ValidatorInterf, selector ValidatorSelectorUpdateInter, socketDelay UpdateDelay) (blockchain *Blockchain) {
	firstBlock := newLinkedBlock(Genesis())
	blockchain = &Blockchain{
		//chain:         []Block{*Genesis()},
		firstBlock:    firstBlock,
		lastBlock:     firstBlock,
		sizeChain:     1,
		existingBlock: make(map[string]struct{}),
		validatorList: validators,
		executedTx:    newMapExecTxSec(),
		Save:          func() {},
		selector:      selector,
		DelayUpdater:  socketDelay,
	}
	blockchain.existingBlock[string(firstBlock.block.Hash)] = struct{}{}
	blockchain.updateProposer()
	return
}

// pushes confirmed blocks into the chain
func (blockchain *Blockchain) addBlock(block Block) {
	blockchain.secLastBlock = blockchain.lastBlock
	blockchain.lastBlock = blockchain.lastBlock.append(block)
	blockchain.sizeChain++
	blockchain.existingBlock[string(block.Hash)] = struct{}{}
	//blockchain.chain = append(blockchain.chain, block)
	blockchain.metrics.AddOneCommittedBlock()
	log.Debug().Msgf("NEW BLOCK ADDED TO CHAIN : n°%d", block.SequenceNb)
}

// CreateBlock wrapper function to create blocks
func (blockchain *Blockchain) CreateBlock(transactions []Transaction, wallet Wallet) *Block {
	return blockchain.GetLastBLoc().CreateBlock(transactions, wallet)
}

// GetProposer calculates the next propsers by calculating a random index of the validators list
// index is calculated using the hash of the latest block
func (blockchain *Blockchain) GetProposer() ed25519.PublicKey {
	return blockchain.proposer
}

func (blockchain *Blockchain) GetProposerId() int {
	return blockchain.proposerID
}

func (blockchain *Blockchain) IsValidNewBlock(block Block) bool {
	lastBlock := blockchain.GetLastBLoc()
	if lastBlock.SequenceNb+1 == block.SequenceNb && bytes.Equal(lastBlock.Hash, block.LastHast) && block.VerifyBlock() && block.VerifyProposer(blockchain.GetProposer()) {
		log.Debug().Msg("BLOCK VALID")
		return true
	} else {
		log.Debug().Msg("BLOCK INVALID")
		return false
	}
}

// AddUpdatedBlockCommited Add a block from a commit
func (blockchain *Blockchain) AddUpdatedBlockCommited(hash []byte, blockpool *BlockPool, transacPool TransactionPoolInterf, preparePool PreparePool, commitPool CommitPool) {
	block := blockpool.GetBlock(hash)
	/*Manque des champs*/
	blockchain.AddUpdatedBlock(block, transacPool)
}

// AddUpdatedBlock Add a block from a block
func (blockchain *Blockchain) AddUpdatedBlock(block Block, transacPool TransactionPoolInterf) {
	/*Manque des champs*/
	blockchain.addBlock(block)
	blockchain.flushTransaction(&block, transacPool)
	blockchain.updateProposer()
}

func (blockchain *Blockchain) flushTransaction(block *Block, transacPool TransactionPoolInterf) {
	blockchain.executedTxLock.Lock()
	for _, transaction := range block.Transactions {
		commande, ok := transaction.TransaCore.Input.(Commande)
		if ok {
			log.Debug().Msg("The command will be applied")
			blockchain.apply(commande)
			log.Debug().Msg("The command was applied")
		}
		blockchain.executedTx.Add(transaction.Hash)
		transacPool.RemoveTransaction(transaction)
	}
	blockchain.executedTxLock.Unlock()
	blockchain.metrics.AddNCommittedTx(len(block.Transactions))
}

func (blockchain *Blockchain) GetLastBLoc() Block {
	//return blockchain.chain[len(blockchain.chain)-1]
	return blockchain.lastBlock.block
}

func (blockchain *Blockchain) GetSecLastBLoc() Block {
	return blockchain.secLastBlock.block
}

func (blockchain *Blockchain) GetCurrentSeqNb() int {
	return blockchain.GetLastBLoc().SequenceNb
}

func (blockchain *Blockchain) GetBlock(id int) Block {
	start := blockchain.firstBlock
	for i := 0; i < id; i++ {
		start = start.nextLink
	}
	//return blockchain.chain[id]
	return start.block
}

func (blockchain *Blockchain) GetLenght() int {
	//return len(blockchain.chain)
	return blockchain.sizeChain
}

func (blockchain *Blockchain) getIDOf(wallet Wallet) int {
	return blockchain.validatorList.GetIndexOfValidator(wallet.publicKey)
}

func (blockchain *Blockchain) apply(commande Commande) {
	switch commande.Order {
	case VarieValid:
		//newSize := blockchain.validatorList.GetNumberOfValidator() + commande.Variation
		newSize := len(commande.NewValidatorSet)
		if blockchain.validatorList.IsSizeValid(newSize) {
			//blockchain.validatorList.SetNumberOfNode(newSize)
			updated := blockchain.validatorList.SetNewListOfValidator(commande.NewValidatorSet)
			if updated {
				listValIndex := blockchain.validatorList.getIndexOfValidators()
				blockchain.selector.Update(listValIndex)
			}
			log.Debug().Msgf("Number of Active Node %d", blockchain.validatorList.GetNumberOfValidator())
			blockchain.validatorList.logAllValidator()
		} else {
			log.Error().Msgf("The new size of block is invalid, new size :%d", newSize)
		}
	case ChangeDelay:
		if blockchain.DelayUpdater != nil {
			blockchain.DelayUpdater.UpdateDelay(commande.Variation)
		}
	default:
		log.Error().Msgf("Unknown message type %d", commande.Order)
	}
}

func (blockchain *Blockchain) ExistTx(transactionHash []byte) bool {
	blockchain.executedTxLock.RLock()
	ok := blockchain.executedTx.Exist(transactionHash)
	blockchain.executedTxLock.RUnlock()
	return ok
}

// GetChainJson Return the blockchain in the Json format
func (blockchain *Blockchain) GetChainJson() []byte {
	byted, err := json.MarshalIndent(*blockchain.GetBlocs(), "", "  ")
	if err != nil {
		log.Warn().Msg("Error marshal blockchain")
	}
	return byted
}
func (blockchain *Blockchain) GetBlocs() *[]Block {
	size := blockchain.sizeChain
	copie := make([]Block, size)
	pointer := blockchain.firstBlock
	for i := 0; i < size; i++ {
		copie[i] = pointer.block
		pointer = pointer.nextLink
	}
	return &copie
}

func (blockchain *Blockchain) GetFragmentBlocks(point chan<- *[]*Block) {
	size := blockchain.sizeChain
	pointer := blockchain.firstBlock
	var done int
	for done != size {
		blockFragment := min(blocFragmentMax, size-done)
		done += blockFragment
		copie := make([]*Block, blockFragment)
		for j := 0; j < blockFragment; j++ {
			copie[j] = &(pointer.block)
			pointer = pointer.nextLink
		}
		point <- &copie
	}
	point <- nil
}

func (blockchain *Blockchain) ExistBlockOfHash(hash []byte) bool {
	_, ok := blockchain.existingBlock[string(hash)]
	return ok
}

func (blockchain *Blockchain) SetMetricHandler(handler *MetricHandler) {
	blockchain.metrics = handler
}

// updateProposer change the current proposer for the next
func (blockchain *Blockchain) updateProposer() {
	lastBlock := blockchain.lastBlock.block
	nbVal := blockchain.validatorList.GetNumberOfValidator()
	index := int(lastBlock.Hash[0]) % nbVal
	blockchain.proposerID, blockchain.proposer = blockchain.validatorList.GetActiveValidatorIndexOfValue(index)
	log.Debug().Msgf("The next proposer is of index %d", blockchain.proposerID)
}
