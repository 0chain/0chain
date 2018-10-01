package miner

import (
	"context"
	"sync"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
)

//RoundMismatch - to indicate an error where the current round and the given round don't match
const RoundMismatch = "round_mismatch"

//ErrRoundMismatch - an error object for mismatched round error
var ErrRoundMismatch = common.NewError(RoundMismatch, "Current round number of the chain doesn't match the block generation round")

var minerChain = &Chain{}

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = c
	minerChain.BlockMessageChannel = make(chan *BlockMessage, 25)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

/*Chain - A miner chain to manage the miner activities */
type Chain struct {
	*chain.Chain
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
	r.ComputeMinerRanks(mc.Miners.Size())
	mr.Round = r
	mr.blocksToVerifyChannel = make(chan *block.Block, mc.NumGenerators)
	return &mr
}

/*CancelRoundsBelow - delete rounds below */
func (mc *Chain) CancelRoundsBelow(ctx context.Context, round int64) {
	mc.roundsMutex.Lock()
	defer mc.roundsMutex.Unlock()
	for _, r := range mc.rounds {
		if r.GetRoundNumber() < round {
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

/*SetPreviousBlock - set the previous block */
func (mc *Chain) SetPreviousBlock(ctx context.Context, r *round.Round, b *block.Block, pb *block.Block) {
	if r == nil {
		mr := mc.GetMinerRound(b.Round)
		if mr != nil {
			r = mr.Round
		} else {
			r = datastore.GetEntityMetadata("round").Instance().(*round.Round)
			r.Number = b.Round
			r.RandomSeed = b.RoundRandomSeed
		}
	}
	b.SetPreviousBlock(pb)
	b.RoundRandomSeed = r.GetRandomSeed()
	bNode := node.GetNode(b.MinerID)
	b.RoundRank = r.GetMinerRank(bNode.SetIndex)
	b.ComputeChainWeight()
}

//GetMinerRound - get the miner's version of the round
func (mc *Chain) GetMinerRound(roundNumber int64) *Round {
	r := mc.GetRound(roundNumber)
	if r == nil {
		return nil
	}
	mr, ok := r.(*Round)
	if !ok {
		return nil
	}
	return mr
}
