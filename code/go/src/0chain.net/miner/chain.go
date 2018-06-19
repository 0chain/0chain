package miner

import (
	"context"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/node"
	"0chain.net/round"
)

var ErrRoundMismatch = common.NewError("round_mismatch", "Current round number of the chain doesn't match the block generation round")

var minerChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = *c
	minerChain.Initialize()
	minerChain.roundsMutex = &sync.Mutex{}
	minerChain.BlockMessageChannel = make(chan *BlockMessage, 25)
}

/*Initialize - intializes internal datastructures to start again */
func (mc *Chain) Initialize() {
	minerChain.Chain.Initialize()
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
func (mc *Chain) SetupGenesisBlock(hash string) *block.Block {
	gr, gb := mc.GenerateGenesisBlock(hash)
	if gr == nil || gb == nil {
		panic("Genesis round/block canot be null")
	}
	mgr := mc.CreateRound(gr)
	mc.AddRound(mgr)
	mc.AddGenesisBlock(gb)
	return gb
}

/*CreateRound - create a round */
func (mc *Chain) CreateRound(r *round.Round) *Round {
	var mr Round
	mr.Round = *r
	mr.blocksToVerifyChannel = make(chan *block.Block, 200)
	r.ComputeRanks(mc.Miners.Size())
	return &mr
}

/*AddRound - Add Round to the block */
func (mc *Chain) AddRound(r *Round) bool {
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	_, ok := mc.rounds[r.Number]
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
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	round, ok := mc.rounds[roundNumber]
	if !ok {
		return nil
	}
	return round
}

/*DeleteRound - delete a round and associated block data */
func (mc *Chain) DeleteRound(ctx context.Context, r *round.Round) {
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	delete(mc.rounds, r.Number)
}

/*CanGenerateRound - checks if the miner can generate a block in the given round */
func (mc *Chain) CanGenerateRound(r *round.Round, miner *node.Node) bool {
	return 2*r.GetRank(miner.SetIndex) <= mc.Miners.Size()
}

/*ValidGenerator - check whether this block is from a valid generator */
func (mc *Chain) ValidGenerator(r *round.Round, b *block.Block) bool {
	miner := mc.Miners.GetNode(b.MinerID)
	if miner == nil {
		return false
	}
	return mc.CanGenerateRound(r, miner)
}
