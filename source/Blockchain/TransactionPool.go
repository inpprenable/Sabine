package Blockchain

import (
	"github.com/rs/zerolog/log"
	"pbftnode/source/config"
	"sync"
	"time"
)

type OverloadBehavior uint8

const (
	Nothing OverloadBehavior = iota
	Ignore
	Drop
)

func StrToBehavior(behavior string) OverloadBehavior {
	switch behavior {
	case "Nothing":
		return Nothing
	case "Ignore":
		return Ignore
	case "Drop":
		return Drop
	default:
		log.Error().Msgf("The string %s is not a behavior", behavior)
		return Nothing
	}
}

func (behavior OverloadBehavior) afterTxAdd(pool *TransactionPool) {
	switch behavior {
	case Ignore:
		if pool.txPoolSize() == pool.transactionThresold {
			pool.stopChanHandler.tellIfContinu(false)
		}
	case Drop:
		if pool.txPoolSize() == pool.transactionThresold {
			pool.almostClear()
		}
	}
}

func (behavior OverloadBehavior) afterTxRemove(pool *TransactionPool) {
	switch behavior {
	case Ignore:
		if !pool.stopChanHandler.getStatus() && pool.txPoolSize() <= pool.transactionThresold*3/4 {
			pool.stopChanHandler.tellIfContinu(true)
		}
	}
}

type TransactionPool struct {
	validators          ValidatorInterf
	transactions        map[string]Transaction
	commands            map[string]Transaction
	transactionThresold int
	sortedList          sortedStruct
	metrics             MetricInterf
	chanExistingTx      chan struct {
		tx   Transaction
		resp chan bool
	}
	chanPoolSize chan chan int
	chanClear    chan chan struct{}
	chanAddTx    chan struct {
		tx   Transaction
		resp chan bool
	}
	chanRemoveTx chan struct {
		tx   Transaction
		resp chan struct{}
	}
	overloadBehavior OverloadBehavior
	chanGetBlockTx   chan chan []Transaction
	stopChanHandler  stopingChanHandler
}

type stopingChanHandler struct {
	listChan []chan bool
	status   bool
}

func newStopingChanHandler() stopingChanHandler {
	return stopingChanHandler{status: true}
}

func (handler *stopingChanHandler) createAChan() <-chan bool {
	channel := make(chan bool, 8)
	handler.listChan = append(handler.listChan, channel)
	return channel
}

func (handler *stopingChanHandler) tellIfContinu(newStatus bool) {
	if handler.status != newStatus {
		handler.status = newStatus
		for _, channel := range handler.listChan {
			channel <- newStatus
		}
	}
}

func (handler stopingChanHandler) getStatus() bool {
	return handler.status
}

func (pool *TransactionPool) loopTransactionPool() {
	var wait sync.WaitGroup
	for {
		select {
		case query := <-pool.chanExistingTx:
			wait.Add(1)
			go func() {
				query.resp <- pool.existingTransaction(query.tx)
				wait.Done()
			}()
		case query := <-pool.chanPoolSize:
			wait.Add(1)
			go func() {
				query <- pool.poolSize()
				wait.Done()
			}()
		case query := <-pool.chanClear:
			wait.Wait()
			pool.clear()
			query <- struct{}{}
		case query := <-pool.chanAddTx:
			wait.Wait()
			query.resp <- pool.addTransaction(query.tx)
		case query := <-pool.chanRemoveTx:
			query.resp <- struct{}{}
			wait.Wait()
			pool.removeTransaction(query.tx)
		case query := <-pool.chanGetBlockTx:
			wait.Add(1)
			go func() {
				query <- pool.getTxForBloc()
				wait.Done()
			}()
		}
	}
}

// ExistingTransaction check if the transaction exists or not
func (pool TransactionPool) ExistingTransaction(transaction Transaction) bool {
	respChan := make(chan bool)
	pool.chanExistingTx <- struct {
		tx   Transaction
		resp chan bool
	}{tx: transaction, resp: respChan}
	resp := <-respChan
	close(respChan)
	return resp
}

func (pool TransactionPool) existingTransaction(transaction Transaction) bool {
	_, okc := pool.commands[string(transaction.Hash)]
	_, okt := pool.transactions[string(transaction.Hash)]
	return okc || okt
}

// PoolSize return the pool size
func (pool TransactionPool) PoolSize() int {
	query := make(chan int)
	pool.chanPoolSize <- query
	resp := <-query
	close(query)
	return resp
}

func (pool TransactionPool) poolSize() int {
	return pool.txPoolSize() + pool.commandPoolSize()
}

func (pool TransactionPool) txPoolSize() int {
	return len(pool.transactions)
}

func (pool TransactionPool) commandPoolSize() int {
	return len(pool.commands)
}

func (pool TransactionPool) IsEmpty() bool {
	return pool.PoolSize() == 0
}

func (pool TransactionPool) Clear() {
	query := make(chan struct{})
	pool.chanClear <- query
	<-query
	close(query)
}

func (pool *TransactionPool) clear() {
	log.Info().Msg("TRANSACTION POOL CLEARED")
	pool.transactions = make(map[string]Transaction, pool.transactionThresold)
	pool.sortedList = newSortedLinkedList(&pool.transactions)

}

func (pool TransactionPool) AddTransaction(transaction Transaction) bool {
	respChan := make(chan bool)
	pool.chanAddTx <- struct {
		tx   Transaction
		resp chan bool
	}{tx: transaction, resp: respChan}
	resp := <-respChan
	close(respChan)
	return resp
}

func (pool *TransactionPool) addTransaction(transaction Transaction) bool {
	if transaction.IsCommand() {
		if pool.commandPoolSize() < pool.transactionThresold {
			pool.commands[string(transaction.Hash)] = transaction
			log.Debug().Msgf("added command to pool")
			pool.metrics.AddOneReceivedTx()
			return true
		}
	} else {
		if pool.txPoolSize() < pool.transactionThresold {
			pool.transactions[string(transaction.Hash)] = transaction
			pool.sortedList.insert(string(transaction.Hash))
			log.Debug().Msgf("added transaction to pool")
			pool.metrics.AddOneReceivedTx()
			pool.overloadBehavior.afterTxAdd(pool)
			return true
		}
	}
	return false
}

func (pool TransactionPool) RemoveTransaction(transaction Transaction) {
	respChan := make(chan struct{})
	pool.chanRemoveTx <- struct {
		tx   Transaction
		resp chan struct{}
	}{tx: transaction, resp: respChan}
	<-respChan
	close(respChan)
}

func (pool *TransactionPool) removeTransaction(transaction Transaction) {
	log.Debug().Msgf("remove the transaction : %s", transaction.GetHashPayload())
	if transaction.IsCommand() {
		delete(pool.commands, string(transaction.Hash))
	} else {
		hash := string(transaction.Hash)
		_, ok := pool.transactions[hash]
		if ok {
			delete(pool.transactions, hash)
			pool.sortedList.remove(hash)
		}
		pool.overloadBehavior.afterTxRemove(pool)
	}
}

func (pool *TransactionPool) almostClear() {
	almostSize := pool.transactionThresold / 64
	log.Warn().Int64("at", time.Now().UnixNano()).
		Msgf("%d messages are removed", pool.txPoolSize()-almostSize)
	newTxMap := make(map[string]Transaction, pool.transactionThresold)
	newSortList := newSortedLinkedList(&newTxMap)
	var cmpt int
	for key := range pool.transactions {
		newTxMap[key] = pool.transactions[key]
		newSortList.insert(key)
		cmpt++
		if cmpt == almostSize {
			break
		}
	}
	pool.transactions = newTxMap
	pool.sortedList = newSortList
	pool.sortedList.setMap(&(pool.transactions))
}

func (pool TransactionPool) GetTxForBloc() []Transaction {
	query := make(chan []Transaction)
	pool.chanGetBlockTx <- query
	resp := <-query
	close(query)
	return resp
}

func (pool *TransactionPool) getTxForBloc() []Transaction {
	var listTx []Transaction
	for key, tx := range pool.commands {
		if tx.VerifyTransaction() && tx.VerifyAsCommandShort(pool.validators) {
			listTx = append(listTx, tx)
			return listTx
		} else {
			log.Warn().Msgf("The command %s is invalid and is deleted", key)
			go func() { pool.RemoveTransaction(pool.commands[key]) }()
		}
	}
	potentiel := pool.sortedList.getFirstsElem(config.BlockSize)
	for _, hash := range potentiel {
		tx := pool.transactions[hash]
		if tx.VerifyTransaction() {
			listTx = append(listTx, tx)
		} else {
			log.Warn().Msgf("The transaction %s is invalid and is deleted", hash)
			go func() { pool.RemoveTransaction(pool.transactions[hash]) }()
		}
	}

	var i int
	for len(listTx) < config.BlockSize && len(pool.transactions) > len(listTx)+i {
		hash := pool.sortedList.getElemNumber(i)
		tx := pool.transactions[hash]
		if tx.VerifyTransaction() {
			listTx = append(listTx, tx)
		} else {
			log.Warn().Msgf("The transaction %s is invalid and delete", hash)
			go func() { pool.RemoveTransaction(pool.transactions[hash]) }()
			i++
		}
	}
	return listTx
}

func (pool *TransactionPool) SetMetricHandler(handler MetricInterf) {
	pool.metrics = handler
}

func NewTransactionPool(validator ValidatorInterf, behavior OverloadBehavior) *TransactionPool {
	pool := &TransactionPool{
		validators:          validator,
		transactions:        make(map[string]Transaction, config.TransactionThreshold),
		commands:            make(map[string]Transaction, config.TransactionThreshold),
		transactionThresold: config.TransactionThreshold,
		chanExistingTx: make(chan struct {
			tx   Transaction
			resp chan bool
		}),
		chanPoolSize: make(chan chan int),
		chanClear:    make(chan chan struct{}),
		chanAddTx: make(chan struct {
			tx   Transaction
			resp chan bool
		}),
		chanRemoveTx: make(chan struct {
			tx   Transaction
			resp chan struct{}
		}),
		chanGetBlockTx:   make(chan chan []Transaction),
		stopChanHandler:  newStopingChanHandler(),
		overloadBehavior: behavior,
	}
	pool.sortedList = newSortedLinkedList(&pool.transactions)
	go pool.loopTransactionPool()
	return pool
}

func (pool *TransactionPool) GetStopingChan() <-chan bool {
	return pool.stopChanHandler.createAChan()
}

func (pool *TransactionPool) changeOverloadBehavior(newBehavior OverloadBehavior) {
	pool.overloadBehavior = newBehavior
}
