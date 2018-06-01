package miner

import (
	"context"
	"math/rand"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
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
	minerChain.BlocksChannel = make(chan *block.Block)
	minerChain.rounds = make(map[int64]*round.Round)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

/*Chain - A miner chain is a chain that also tracks all the speculative SpeculativeChains and Blocks */
type Chain struct {
	chain.Chain
	SpeculativeChains []*block.Block
	Blocks            map[datastore.Key]*block.Block
	BlocksChannel     chan *block.Block
	roundsMutex       *sync.Mutex
	rounds            map[int64]*round.Round
}

/*IsBlockPresent - do we already have this block? */
func (mc *Chain) IsBlockPresent(hash string) bool {
	_, ok := mc.Blocks[hash]
	return ok
}

/*GetBlock - returns a known block for a given hash from the speculative list of blocks */
func (mc *Chain) GetBlock(ctx context.Context, hash string) (*block.Block, error) {
	b, ok := mc.Blocks[datastore.Key(hash)]
	if ok {
		return b, nil
	}
	b = block.Provider().(*block.Block)
	err := b.Read(ctx, datastore.ToKey(hash))
	if err != nil {
		return b, nil
	}
	return nil, err
}

/*BestBlockToExtend - what's the best speculative block to extend from among the set of Blocks being tracked */
func (mc *Chain) BestBlockToExtend() *block.Block {
	//TODO: For now just do it based on random so it covers many test cases
	if len(mc.SpeculativeChains) == 0 {
		return mc.LatestFinalizedBlock
	}
	ind := rand.Int63n(int64(len(mc.SpeculativeChains)))
	return mc.SpeculativeChains[ind]
}

/*AddBlock - add a sepculative block
* Assumes Genesis block is already setup
 */
func (mc *Chain) AddBlock(b *block.Block) bool {
	if !datastore.IsEqual(b.ChainID, mc.GetKey()) {
		return false
	}

	if b.Hash == "" || b.PrevHash == "" {
		return false
	}

	//Block shouldn't already exist
	_, ok := mc.Blocks[b.Hash]
	if ok {
		return false
	}

	//PrevBlock should be present
	prevBlock, ok := mc.Blocks[b.PrevHash]
	if !ok {
		//TODO: May be we missed getting the previous block on the network
		//If so, may be we should get it from someone on the net
		return false
	}
	if prevBlock.Round >= b.Round {
		// This is not only functional logic, but also prevents circular SpeculativeChains and infinite loops
		return false
	}
	b.PrevBlock = prevBlock

	// No need to process Blocks that are older than the finalized block!
	if mc.LatestFinalizedBlock.Round > b.Round {
		return false
	}

	if mc.MaxRound < b.Round {
		mc.MaxRound = b.Round
	}

	// We want to add the block now, but only after compacting
	b.CompactBlock()

	var found = false
	for idx, lb := range mc.SpeculativeChains {
		if lb == b.PrevBlock {
			// as chain extends, we just work with the latest block
			mc.SpeculativeChains[idx] = b
			found = true
			break
		}
	}
	//Perhaps we already replaced the prevblock with some other speculative block
	//So, we still need to add this block
	if !found {
		mc.SpeculativeChains = append(mc.SpeculativeChains, b)
	}
	return true
}

/*AddLatestFinalizedBlock - Adds the latest finalized block. This is local to this node and each node
* may have it's own version of finalized block. But they all have to eventualy agree.
 */
func (mc *Chain) AddLatestFinalizedBlock(ctx context.Context, b *block.Block) bool {
	if !datastore.IsEqual(b.ChainID, mc.GetKey()) {
		return false
	}
	if b.PrevBlock == nil {
		pb, err := mc.GetBlock(ctx, b.PrevHash)
		if err != nil {
			return false
		}
		b.PrevBlock = pb
	}
	return true
}

/*ComputeSpeculativeChain - Given a block, compute the speculative chain up to the latest finalized block
* Ordering : 0 (latest block) -> n (given block)
 */
func ComputeSpeculativeChain(c *chain.Chain, b *block.Block) []*block.Block {
	sc := make([]*block.Block, 0, 10)
	lb := c.LatestFinalizedBlock
	for cb := b; cb != nil; cb = cb.PrevBlock {
		sc = append(sc, cb)
		if cb == lb {
			break
		}
	}
	for i, j := len(sc)-1, 0; i >= 0 && j < i; i, j = i-1, j+1 {
		sc[i], sc[j] = sc[j], sc[i]
	}
	return sc
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
	mc.rounds[r.Number] = r
	return true
}

/*GenerateBlock - given a round number generates a block*/
func (mc *Chain) GenerateRoundBlock(ctx context.Context, roundNumber int64) (*block.Block, error) {
	pround := mc.GetRound(roundNumber - 1)
	if pround == nil {
		return nil, common.NewError("invalid_round,", "Round not available")
	}
	r := mc.GetRound(roundNumber)
	if r == nil {
		r = &round.Round{Number: roundNumber}
		mc.AddRound(r)
	}
	b := &block.Block{}
	b.ChainID = mc.ID
	b.Round = roundNumber
	b.SetPreviousBlock(pround.Block)
	err := mc.GenerateBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	r.Block = b
	r.AddBlock(b) // We need to add our own generated block to the list of blocks
	mc.Miners.SendAll(VBSender(b))
	return b, nil
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
