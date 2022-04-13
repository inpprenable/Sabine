package Consensus

import (
	"bytes"
	"github.com/rs/zerolog/log"
	"pbftnode/source/Blockchain"
)

type NewRound struct {
	*PBFTStateConsensus
}

func (consensus NewRound) testState(message Blockchain.Message) bool {
	preprepared, ok := message.Data.(Blockchain.Block)
	return ok && message.Flag == Blockchain.PrePrepare && consensus.BlockChain.IsValidNewBlock(preprepared)
}

func (consensus NewRound) giveMeNext(message *Blockchain.Message) Blockchain.Payload {
	if message != nil && consensus.testState(*message) {
		return message.Data
	}
	for _, block := range consensus.BlockPool.GetBlocksOfSeqNb(consensus.BlockChain.GetCurrentSeqNb() + 1) {
		if consensus.BlockChain.IsValidNewBlock(block) {
			return block
		}
	}
	return nil
}

func (consensus *NewRound) update(message Blockchain.Payload) *Blockchain.Message {
	block, _ := message.(Blockchain.Block)
	consensus.currentHash = block.Hash
	consensus.updateState(PrePreparedSt)
	var prepare = Blockchain.CreatePrepare(&block, &(consensus.Wallet))
	emitMsg := Blockchain.Message{
		Flag: Blockchain.PrepareMess,
		Data: prepare,
	}
	consensus.PreparePool.AddPrepare(prepare)
	consensus.SocketHandler.BroadcastMessage(emitMsg)
	consensus.updateState(PreparedSt)
	return &emitMsg
}

type NewRoundProposer struct {
	*PBFTStateConsensus
}

func (consensus NewRoundProposer) testState(message Blockchain.Message) bool {
	return !consensus.TransactionPool.IsEmpty()
}

func (consensus NewRoundProposer) giveMeNext(message *Blockchain.Message) Blockchain.Payload {
	consensus.Control.CheckControl()
	if consensus.testState(Blockchain.Message{}) {
		return *consensus.BlockChain.CreateBlock(consensus.TransactionPool.GetTxForBloc(), consensus.Wallet)
		//return *consensus.BlockChain.CreateBlock(consensus.TransactionPoolMutex.GetListTransact(), consensus.Wallet)
	}
	return nil
}

func (consensus *NewRoundProposer) update(payload Blockchain.Payload) *Blockchain.Message {
	block := payload.(Blockchain.Block)
	consensus.currentHash = block.Hash
	consensus.BlockPool.AddBlock(block)
	log.Debug().Msg("CREATED BLOCK, broadcast it")
	message := Blockchain.Message{
		Flag: Blockchain.PrePrepare,
		Data: block,
	}
	consensus.SocketHandler.BroadcastMessage(message)
	consensus.updateState(NewRoundSt)
	return &message
}

type Prepared struct {
	*PBFTStateConsensus
}

func (consensus Prepared) testState(message Blockchain.Message) bool {
	prepare, ok := message.Data.(Blockchain.Prepare)
	return ok && bytes.Equal(prepare.BlockHash, consensus.currentHash) && consensus.Validators.IsActiveValidator(prepare.GetProposer())
}

func (consensus Prepared) giveMeNext(message *Blockchain.Message) Blockchain.Payload {
	if consensus.PreparePool.GetNbPrepareOfHashOfActive(string(consensus.currentHash), consensus.Validators) >= consensus.MinApprovals() {
		return Blockchain.CreateCommit(consensus.currentHash, &(consensus.Wallet))
	}
	return nil
}

func (consensus *Prepared) update(payload Blockchain.Payload) *Blockchain.Message {
	var commit = payload.(Blockchain.Commit)
	consensus.PreparePool.Validate(string(commit.BlockHash), consensus.Validators)
	log.Debug().Msg("Broadcast commit")
	consensus.CommitPool.AddCommit(commit)
	message := Blockchain.Message{
		Flag: Blockchain.CommitMess,
		Data: commit,
	}
	consensus.SocketHandler.BroadcastMessage(message)
	consensus.updateState(CommittedSt)
	return &message
}

type Committed struct {
	*PBFTStateConsensus
}

func (consensus Committed) testState(message Blockchain.Message) bool {
	commit, ok := message.Data.(Blockchain.Commit)
	return ok && bytes.Equal(commit.BlockHash, consensus.currentHash) && consensus.Validators.IsActiveValidator(commit.GetProposer())
}

func (consensus Committed) giveMeNext(message *Blockchain.Message) Blockchain.Payload {
	if consensus.CommitPool.GetNbPrepareOfHashOfActive(string(consensus.currentHash), consensus.Validators) >= consensus.MinApprovals() {
		return consensus.BlockPool.GetBlock(consensus.currentHash)
	}
	return nil
}

func (consensus *Committed) update(payload Blockchain.Payload) *Blockchain.Message {
	block := payload.(Blockchain.Block)
	consensus.updateState(FinalCommittedSt)
	consensus.CommitPool.Validate(string(block.Hash), consensus.Validators)
	if consensus.IsProposer() && consensus.PoANV {
		consensus.SocketHandler.BroadcastMessageNV(Blockchain.Message{
			Flag: Blockchain.BlocMsg,
			Data: Blockchain.BlockMsg{Block: block},
		})
	}
	consensus.BlockChain.AddUpdatedBlockCommited(block.Hash, consensus.BlockPool, consensus.TransactionPool, consensus.PreparePool, consensus.CommitPool)
	if consensus.RamOpt {
		consensus.BlockPool.RemoveBlock(block.Hash)
	}
	consensus.updateStateAfterCommit()
	return nil
}

type NewRoundNV struct {
	*PBFTStateConsensus
}

func (consensus NewRoundNV) testState(message Blockchain.Message) bool {
	blockMsg, ok := message.Data.(Blockchain.BlockMsg)
	return ok && consensus.BlockChain.IsValidNewBlock(blockMsg.Block)
}

func (consensus NewRoundNV) giveMeNext(message *Blockchain.Message) Blockchain.Payload {
	if message != nil && consensus.testState(*message) {
		return message.Data
	}
	for _, block := range consensus.BlockPoolNV.GetBlocksOfSeqNb(consensus.BlockChain.GetCurrentSeqNb() + 1) {
		if consensus.BlockChain.IsValidNewBlock(block) {
			return Blockchain.BlockMsg{Block: block}
		}
	}
	return nil
}

func (consensus *NewRoundNV) update(payload Blockchain.Payload) *Blockchain.Message {
	block := payload.(Blockchain.BlockMsg)
	consensus.currentHash = block.Block.Hash
	consensus.BlockChain.AddUpdatedBlock(block.Block, consensus.TransactionPool)
	if consensus.RamOpt {
		consensus.BlockPoolNV.RemoveBlock(block.Block.Hash)
	}
	consensus.updateState(FinalCommittedSt)
	consensus.updateStateAfterCommit()
	return nil
}
