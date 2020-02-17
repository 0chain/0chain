package sharder

import (
	"context"
	"time"

	"0chain.net/core/cache"
	"0chain.net/core/ememorystore"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/sharder/blockstore"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
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
	c.SetFetchedNotarizedBlockHandler(sharderChain)
	sharderChain.BlockSyncStats = &SyncStats{}
	c.RoundF = SharderRoundFactory{}
}

/*GetSharderChain - get the sharder's chain */
func GetSharderChain() *Chain {
	return sharderChain
}

/*Chain - A chain structure to manage the sharder activities */
type Chain struct {
	*chain.Chain
	BlockChannel   chan *block.Block
	RoundChannel   chan *round.Round
	BlockCache     cache.Cache
	BlockTxnCache  cache.Cache
	SharderStats   Stats
	BlockSyncStats *SyncStats
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
func (sc *Chain) SetupGenesisBlock(hash string, magicBlock *block.MagicBlock) *block.Block {
	gr, gb := sc.GenerateGenesisBlock(hash, magicBlock)
	if gr == nil || gb == nil {
		panic("Genesis round/block can not be null")
	}
	//sc.AddRound(gr)
	sc.AddGenesisBlock(gb)
	// Save the block
	err := sc.storeBlock(common.GetRootContext(), gb)
	if err != nil {
		Logger.Error("Failed to save genesis block",
			zap.Error(err))
	}
	if gb.MagicBlock != nil {
		var tries int64
		bs := gb.GetSummary()
		err = sc.StoreMagicBlockMapFromBlock(common.GetRootContext(), bs.GetMagicBlockMap())
		for err != nil {
			tries++
			Logger.Error("setup genesis block -- failed to store magic block map", zap.Any("error", err), zap.Any("tries", tries))
			time.Sleep(time.Millisecond * 100)
			err = sc.StoreMagicBlockMapFromBlock(common.GetRootContext(), bs.GetMagicBlockMap())
		}
	}
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

/*GetBlockHash - get the block hash for a given round */
func (sc *Chain) GetBlockHash(ctx context.Context, roundNumber int64) (string, error) {
	r := sc.GetSharderRound(roundNumber)
	if r == nil {
		sr, err := sc.GetRoundFromStore(ctx, roundNumber)
		if err != nil {
			return "", err
		}
		r = sr
	}
	return r.BlockHash, nil
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

// load and set FLB from store
func (sc *Chain) loadLatestFinalizedBlockFromStore(round *round.Round) (
	lfb *block.Block) {

	var err error
	lfb, err = sc.GetBlockFromStore(round.BlockHash, round.Number)
	if err != nil {
		Logger.DPanic("obtaining LFB from DB", zap.Error(err),
			zap.Int64("round", round.Number),
			zap.String("hash", round.BlockHash))
	}
	sc.SetLatestFinalizedBlock(lfb)

	Logger.Info("lfb from store", zap.Int64("round", lfb.Round),
		zap.String("hash", lfb.Hash))
	return
}

// iterate from given round down to find block contains magic block
func (sc *Chain) iterateDownToMagicBlock(b *block.Block, wantHash string) (
	lfmb *block.Block) {

	var err error

	for b.Round >= 0 {
		if b, err = sc.GetBlockFromStore(b.PrevHash, b.Round-1); err != nil {
			Logger.DPanic("looking for LFMB", zap.Error(err),
				zap.Int64("round", b.Round),
				zap.String("hash", b.Hash))
		}
		if b.Hash == wantHash {
			return b
		}
	}

	Logger.DPanic("looking for LFMB: block not found",
		zap.String("hash", wantHash))
	return
}

// LoadLatestBlocksFromStore loads LFB and LFMB from store and sets them
// to corresponding fields of the sharder's Chain.
func (sc *Chain) LoadLatestBlocksFromStore(ctx context.Context) {

	var round, err = sc.GetMostRecentRoundFromDB(ctx)

	if err != nil {
		Logger.DPanic("no round from DB given")
		return
	}

	if round.Number == 0 {
		Logger.Debug("genesis lfb and lfmb used")
		return
	}

	var lfb = sc.loadLatestFinalizedBlockFromStore(round)

	// genesis case

	if lfb.LatestFinalizedMagicBlockHash == "" {
		Logger.DPanic("no magic block hash", zap.Int64("lfb_round", lfb.Round),
			zap.String("lfb_hash", lfb.Hash))
		return
	}

	// TODO: more effective way instead of the iterating down (may be)

	// non-genesis case
	var lfmb = sc.iterateDownToMagicBlock(lfb, lfb.LatestFinalizedMagicBlockHash)

	if lfmb.MagicBlock == nil {
		Logger.DPanic("obtaining LFMB from DB: missing magic block",
			zap.Int64("round", lfmb.Round),
			zap.String("hash", lfmb.Hash))
	}

	sc.SetLatestFinalizedMagicBlock(lfmb)

	Logger.Info("lfmb from store", zap.Int64("round", lfmb.Round),
		zap.String("hash", lfmb.Hash))
}
