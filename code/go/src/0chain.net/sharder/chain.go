package sharder

import (
	"context"

	"0chain.net/cache"
	"0chain.net/ememorystore"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/datastore"
	"0chain.net/round"
)

var sharderChain = &Chain{}

/*SetupSharderChain - setup the sharder's chain */
func SetupSharderChain(c *chain.Chain) {
	sharderChain.Chain = c
	sharderChain.BlockChannel = make(chan *block.Block, 128)
	sharderChain.RoundChannel = make(chan *round.Round, 128)
	blockCacheSize := 100
	sharderChain.BlockCache = cache.NewLRUCache(blockCacheSize)
	transactionCacheSize := int(c.BlockSize) * blockCacheSize
	sharderChain.BlockTxnCache = cache.NewLRUCache(transactionCacheSize)
}

/*GetSharderChain - get the sharder's chain */
func GetSharderChain() *Chain {
	return sharderChain
}

/*Chain - A chain structure to manage the sharder activities */
type Chain struct {
	*chain.Chain
	BlockChannel  chan *block.Block
	RoundChannel  chan *round.Round
	BlockCache    cache.Cache
	BlockTxnCache cache.Cache
	SharderStats  Stats
}

/*GetBlockChannel - get the block channel where the incoming blocks from the network are put into for further processing */
func (sc *Chain) GetBlockChannel() chan *block.Block {
	return sc.BlockChannel
}

/*GetRoundChannel - get the round channel where the finalized rounds are put into for further processing */
func (sc *Chain) GetRoundChannel() chan *round.Round {
	return sc.RoundChannel
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
	bs := block.BlockSummary{Hash: blockHash, Round: round}
	return sc.GetBlockFromStoreBySummary(&bs)
}

/*GetBlockFromStoreBySummary - get the block from the store */
func (sc *Chain) GetBlockFromStoreBySummary(bs *block.BlockSummary) (*block.Block, error) {
	return blockstore.GetStore().ReadWithBlockSummary(bs)
}

/*GetRoundFromStore - get the round from a store*/
func (sc *Chain) GetRoundFromStore(ctx context.Context, roundNum int64) (*round.Round, error) {
	r := datastore.GetEntity("round").(*round.Round)
	r.Number = roundNum
	roundEntityMetadata := r.GetEntityMetadata()
	rctx := ememorystore.WithEntityConnection(ctx, roundEntityMetadata)
	defer ememorystore.Close(rctx)
	err := r.Read(rctx, r.GetKey())
	return r, err
}

//GetSharderRound - get the sharder's version of the round
func (sc *Chain) GetSharderRound(roundNumber int64) *round.Round {
	r := sc.GetRound(roundNumber)
	if r == nil {
		return nil
	}
	sr, ok := r.(*round.Round)
	if !ok {
		return nil
	}
	return sr
}
