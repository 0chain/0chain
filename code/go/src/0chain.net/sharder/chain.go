package sharder

import (
	"context"
	"sync"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/round"
)

var sharderChain = &Chain{}

/*SetupSharderChain - setup the sharder's chain */
func SetupSharderChain(c *chain.Chain) {
	sharderChain.Chain = *c
	sharderChain.Initialize()
	sharderChain.roundsMutex = &sync.Mutex{}
	sharderChain.BlockChannel = make(chan *block.Block, 128)
}

/*Initialize - intializes internal datastructures to start again */
func (sc *Chain) Initialize() {
	sc.Chain.Initialize()
	sc.rounds = make(map[int64]*round.Round)
}

/*GetSharderChain - get the sharder's chain */
func GetSharderChain() *Chain {
	return sharderChain
}

/*Chain - A chain structure to manage the sharder activities */
type Chain struct {
	chain.Chain
	BlockChannel chan *block.Block
	roundsMutex  *sync.Mutex
	rounds       map[int64]*round.Round
}

/*GetBlockChannel - get the block channel where the incoming blocks from the network are put into for further processing */
func (sc *Chain) GetBlockChannel() chan *block.Block {
	return sc.BlockChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
func (sc *Chain) SetupGenesisBlock(hash string) *block.Block {
	gr, gb := sc.GenerateGenesisBlock(hash)
	if gr == nil || gb == nil {
		panic("Genesis round/block can not be null")
	}
	//sc.AddRound(gr)
	sc.AddGenesisBlock(gb)
	return gb
}

/*GetBlockFromStore - get the block from the store */
func (sc *Chain) GetBlockFromStore(blockHash string, round int64) (*block.Block, error) {
	return blockstore.GetStore().Read(blockHash, round)
}

/*AddRound - Add Round to the block */
func (sc *Chain) AddRound(r *round.Round) bool {
	sc.roundsMutex.Lock()
	defer sc.roundsMutex.Unlock()
	_, ok := sc.rounds[r.Number]
	if ok {
		return false
	}
	r.ComputeRanks(sc.Miners.Size())
	sc.rounds[r.Number] = r
	if r.Number > sc.CurrentRound {
		sc.CurrentRound = r.Number
	}
	return true
}

/*GetRound - get a round */
func (sc *Chain) GetRound(roundNumber int64) *round.Round {
	sc.roundsMutex.Lock()
	defer sc.roundsMutex.Unlock()
	round, ok := sc.rounds[roundNumber]
	if !ok {
		return nil
	}
	return round
}

/*DeleteRound - delete a round and associated block data */
func (sc *Chain) DeleteRound(ctx context.Context, r *round.Round) {
	sc.roundsMutex.Lock()
	defer sc.roundsMutex.Unlock()
	delete(sc.rounds, r.Number)
}

/*DeleteRoundsBelow - delete rounds below */
func (sc *Chain) DeleteRoundsBelow(ctx context.Context, roundNumber int64) {
	sc.roundsMutex.Lock()
	defer sc.roundsMutex.Unlock()
	rounds := make([]*round.Round, 0, 1)
	for _, r := range sc.rounds {
		if r.Number < roundNumber {
			rounds = append(rounds, r)
		}
	}
	for _, r := range rounds {
		delete(sc.rounds, r.Number)
	}
}
