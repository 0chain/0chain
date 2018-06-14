package miner

import (
	"context"
	"fmt"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

var ErrRoundMismatch = common.NewError("round_mismatch", "Current round number of the chain doesn't match the block generation round")

var minerChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = *c
	minerChain.roundsMutex = &sync.Mutex{}
	minerChain.BlockMessageChannel = make(chan *BlockMessage, 25)
	minerChain.rounds = make(map[int64]*Round)
	minerChain.Blocks = make(map[string]*block.Block)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

/*Chain - A miner chain to manage the miner activities */
type Chain struct {
	chain.Chain
	/* This is a cache of blocks that include speculative blocks */
	Blocks              map[datastore.Key]*block.Block
	BlockMessageChannel chan *BlockMessage
	roundsMutex         *sync.Mutex
	rounds              map[int64]*Round
	CurrentRound        int64
}

/*GetBlockMessageChannel - get the block messages channel */
func (mc *Chain) GetBlockMessageChannel() chan *BlockMessage {
	return mc.BlockMessageChannel
}

/*CreateRound - create a round */
func (mc *Chain) CreateRound(r *round.Round) *Round {
	var mr Round
	mr.Round = *r
	mr.blocksToVerifyChannel = make(chan *block.Block, 200)
	return &mr
}

/*AddRound - Add Round to the block */
func (mc *Chain) AddRound(r *Round) bool {
	_, ok := mc.rounds[r.Number]
	if ok {
		return false
	}
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	_, ok = mc.rounds[r.Number]
	if ok {
		return false
	}
	r.ComputeRanks(mc.Miners.Size())
	mc.rounds[r.Number] = r
	if r.Number > mc.CurrentRound {
		mc.CurrentRound = r.Number
	}
	return true
}

/*GetRound - get a round */
func (mc *Chain) GetRound(roundNumber int64) *Round {
	round, ok := mc.rounds[roundNumber]
	if !ok {
		return nil
	}
	return round
}

/*DeleteRound - delete a round and associated block data */
func (mc *Chain) DeleteRound(ctx context.Context, r *round.Round) {
	for bid, blk := range mc.Blocks {
		if blk.Round == r.Number {
			delete(mc.Blocks, bid)
		}
	}
	delete(mc.rounds, r.Number)
}

/*AddBlock - adds a block to the cache */
func (mc *Chain) AddBlock(b *block.Block) {
	mc.Blocks[b.Hash] = b
	if b.Round == 0 {
		mc.LatestFinalizedBlock = b // Genesis block is always finalized
	} else if b.PrevBlock == nil {
		pb, ok := mc.Blocks[b.PrevHash]
		if ok {
			b.PrevBlock = pb
		} else {
			Logger.Error("Prev block not present", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("prev_block", b.PrevHash))
		}
	}
}

/*GetBlock - returns a known block for a given hash from the cache */
func (mc *Chain) GetBlock(ctx context.Context, hash string) (*block.Block, error) {
	b, ok := mc.Blocks[datastore.ToKey(hash)]
	if ok {
		return b, nil
	}
	/*
		b = block.Provider().(*block.Block)
		err := b.Read(ctx, datastore.ToKey(hash))
		if err != nil {
			return b, nil
		}*/
	return nil, common.NewError(datastore.EntityNotFound, fmt.Sprintf("Block with hash (%v) not found", hash))
}

/*DeleteBlock - delete a block from the cache */
func (mc *Chain) DeleteBlock(ctx context.Context, b *block.Block) {
	delete(mc.Blocks, b.Hash)
}

/*GetRoundBlocks - get the blocks for a given round */
func (mc *Chain) GetRoundBlocks(round int64) []*block.Block {
	blocks := make([]*block.Block, 0, 1)
	for _, blk := range mc.Blocks {
		if blk.Round == round {
			blocks = append(blocks, blk)
		}
	}
	return blocks
}
