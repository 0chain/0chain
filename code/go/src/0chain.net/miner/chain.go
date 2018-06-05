package miner

import (
	"context"
	"fmt"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
)

/*
NOTE: This is all work in progress. All of it might change.
*/

var minerChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = *c
	minerChain.roundsMutex = &sync.Mutex{}
	minerChain.BlockMessageChannel = make(chan *BlockMessage, 25)
	minerChain.rounds = make(map[int64]*round.Round)
	minerChain.Blocks = make(map[string]*block.Block)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

/*Chain - A miner chain is a chain that also tracks all the speculative SpeculativeChains and Blocks */
type Chain struct {
	chain.Chain
	SpeculativeChains   []*block.Block
	Blocks              map[datastore.Key]*block.Block
	BlockMessageChannel chan *BlockMessage
	roundsMutex         *sync.Mutex
	rounds              map[int64]*round.Round
}

/*GetBlockMessageChannel - get the block messages channel */
func (mc *Chain) GetBlockMessageChannel() chan *BlockMessage {
	return mc.BlockMessageChannel
}

/*IsBlockPresent - do we already have this block? */
func (mc *Chain) IsBlockPresent(hash string) bool {
	_, ok := mc.Blocks[hash]
	return ok
}

/*AddBlock - adds a block to the cache */
func (mc *Chain) AddBlock(b *block.Block) {
	mc.Blocks[b.Hash] = b
}

/*GetBlock - returns a known block for a given hash from the speculative list of blocks */
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

/*GetRound - get a round */
func (mc *Chain) GetRound(roundNumber int64) *round.Round {
	round, ok := mc.rounds[roundNumber]
	if !ok {
		return nil
	}
	return round
}

/*AddRound - Add Round to the block */
func (mc *Chain) AddRound(r *round.Round) bool {
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
	return true
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

/*GenerateRoundBlock - given a round number generates a block*/
func (mc *Chain) GenerateRoundBlock(ctx context.Context, r *round.Round) (*block.Block, error) {
	pround := mc.GetRound(r.Number - 1)
	if pround == nil {
		return nil, common.NewError("invalid_round,", "Round not available")
	}
	b := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	b.ChainID = mc.ID
	b.SetPreviousBlock(pround.Block)
	err := mc.GenerateBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	r.Block = b
	r.AddBlock(b) // We need to add our own generated block to the list of blocks
	mc.AddBlock(b)
	mc.SendBlock(ctx, b)
	return b, nil
}

/*VerifyRoundBlock - given a block is verified for a round*/
func (mc *Chain) VerifyRoundBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	if b == nil {
		return nil, common.NewError("invalid_block", "Block not available")
	}
	pround := mc.GetRound(b.Round - 1)
	if pround == nil {
		//TODO: create previous round AND request previous block from miner who sent current block for verification
		return nil, common.NewError("invalid_round,", "Round not available")
	}
	prevBlock, _ := pround.GetBlock(b.PrevHash)
	if prevBlock == nil {
		//TODO: create previous round AND request previous block from miner who sent current block for verification
		return nil, common.NewError("invalid_block", "Prevous block doesn't exist")
	}

	for _, ticket := range prevBlock.VerificationTickets {
		bvt := ticket.GetBlockVerificationTicket(prevBlock)
		if err := mc.VerifyTicket(ctx, prevBlock, bvt); err != nil {
			return nil, err
		}
	}

	bvt, err := mc.VerifyBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	r := mc.GetRound(b.Round)
	if r == nil {
		r = &round.Round{Number: b.Round}
		mc.AddRound(r)
	}
	self := node.GetSelfNode(ctx)
	if self == nil {
		panic("Invalid setup, could not find the self node")
	}
	r.AddBlock(b)
	return bvt, nil
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(lfb *block.Block) {
	if lfb.Hash == mc.LatestFinalizedBlock.Hash {
		return
	}
	ctx := memorystore.WithConnection(context.Background())
	for b := lfb; b != nil && b != mc.LatestFinalizedBlock; b = b.GetPreviousBlock() {
		mc.Finalize(ctx, b)
	}
}
