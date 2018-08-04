package miner

import (
	"context"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/round"
)

//RoundMismatch - to indicate an error where the current round and the given round don't match
const RoundMismatch = "round_mismatch"

//ErrRoundMismatch - an error object for mismatched round error
var ErrRoundMismatch = common.NewError(RoundMismatch, "Current round number of the chain doesn't match the block generation round")

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
	DiscoverClients     bool
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
	r.ComputeRanks(mc.Miners.Size())
	mr.Round = *r
	mr.blocksToVerifyChannel = make(chan *block.Block, 200)
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

/*DeleteRoundsBelow - delete rounds below */
func (mc *Chain) DeleteRoundsBelow(ctx context.Context, round int64) {
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	rounds := make([]*Round, 0, 1)
	for _, r := range mc.rounds {
		if r.Number < round {
			rounds = append(rounds, r)
		}
	}
	for _, r := range rounds {
		r.Clear()
		delete(mc.rounds, r.Number)
	}
}

/*CancelRoundsBelow - delete rounds below */
func (mc *Chain) CancelRoundsBelow(ctx context.Context, round int64) {
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	for _, r := range mc.rounds {
		if r.Number < round {
			r.CancelVerification()
		}
	}
}

func (mc *Chain) deleteTxns(txns []datastore.Entity) error {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), transactionMetadataProvider)
	defer memorystore.Close(ctx)
	return transactionMetadataProvider.GetStore().MultiDelete(ctx, transactionMetadataProvider, txns)
}
