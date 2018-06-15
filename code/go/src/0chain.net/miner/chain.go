package miner

import (
	"context"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/round"
)

var ErrRoundMismatch = common.NewError("round_mismatch", "Current round number of the chain doesn't match the block generation round")

var minerChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = *c
	minerChain.roundsMutex = &sync.Mutex{}
	minerChain.BlockMessageChannel = make(chan *BlockMessage, 25)
	minerChain.rounds = make(map[int64]*Round)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

/*Chain - A miner chain to manage the miner activities */
type Chain struct {
	chain.Chain
	BlockMessageChannel chan *BlockMessage
	roundsMutex         *sync.Mutex
	rounds              map[int64]*Round
}

/*GetBlockMessageChannel - get the block messages channel */
func (mc *Chain) GetBlockMessageChannel() chan *BlockMessage {
	return mc.BlockMessageChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
func (mc *Chain) SetupGenesisBlock() *block.Block {
	gr, gb := mc.GenerateGenesisBlock()
	if gr == nil || gb == nil {
		panic("Genesis round/block canot be null")
	}
	mgr := mc.CreateRound(gr)
	mgr.AddBlockToVerify(gb)
	mc.AddRound(mgr)
	mc.AddBlock(gb)
	return gb
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

/*ValidateMagicBlock - validate the block for a given round has the right magic block */
func (mc *Chain) ValidateMagicBlock(ctx context.Context, b *block.Block) bool {
	//TODO: This needs to take the round number into account and go backwards as needed to validate
	return b.MagicBlockHash == mc.CurrentMagicBlock.Hash
}
