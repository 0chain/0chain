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

func (sc *Chain) setupLatestBlocks(ctx context.Context, round *round.Round,
	lfb, lfmb *block.Block) {

	// using ClientState of genesis block

	sc.InitBlockState(lfb)
	lfb.SetStateStatus(block.StateSuccessful)
	lfb.SetBlockState(block.StateNotarized)

	sc.UpdateMagicBlock(lfmb.MagicBlock)

	sc.SetRandomSeed(round, round.GetRandomSeed())
	round.ComputeMinerRanks(lfmb.MagicBlock.Miners)
	round.Block = lfb
	round.AddNotarizedBlock(lfb)

	// set LFB and LFMB of the Chain, add the block to internal Chain's map
	sc.AddLoadedFinalizedBlocks(lfb, lfmb)
}

// iterate over rounds from latest to zero looking for LFB and ignoring
// missing blocks in blockstore
func (sc *Chain) iterateRoundsLookingForLFB(ctx context.Context) (
	lfb *block.Block, r *round.Round, err error) {

	var (
		remd = datastore.GetEntityMetadata("round")
		rctx = ememorystore.WithEntityConnection(ctx, remd)
	)
	defer ememorystore.Close(rctx)

	var (
		conn = ememorystore.GetEntityCon(rctx, remd)
		iter = conn.Conn.NewIterator(conn.ReadOptions)
	)
	defer iter.Close()

	r = remd.Instance().(*round.Round) //

	iter.SeekToLast() // from last

	if !iter.Valid() {
		return nil, r, nil // round 0 (genesis, initial)
	}

	// loop

	for ; iter.Valid(); iter.Prev() {
		if err = datastore.FromJSON(iter.Value().Data(), r); err != nil {
			return // critical
		}

		Logger.Debug("load lfb, got round", zap.Int64("round", r.Number),
			zap.String("block_hash", r.BlockHash))

		lfb, err = sc.GetBlockFromStore(r.BlockHash, r.Number)
		if err != nil {
			continue // TODO: can we use os.IsNotExist(err) or should not
		}

		return // got them
	}

	// not found

	return nil, nil, common.NewError("load_lfb", "lfb not found")
}

func (sc *Chain) loadLatestFinalizedMagicBlockFromStore(ctx context.Context,
	lfb *block.Block) (lfmb *block.Block, err error) {

	if lfb.LatestFinalizedMagicBlockHash == lfb.Hash {

		if lfb.MagicBlock == nil {
			// fatal
			return nil, common.NewError("load_lfb", "missing MagicBlock field")
		}

		return lfb, nil // the same
	}

	// load from store

	Logger.Debug("load lfmb from store",
		zap.String("block_with_magic_block_hash",
			lfb.LatestFinalizedMagicBlockHash))

	lfmb, err = blockstore.GetStore().Read(lfb.LatestFinalizedMagicBlockHash)
	if err != nil {
		// fatality, can't find related LFMB
		return nil, common.NewError("load_lfb",
			"related magic block not found: "+err.Error())
	}

	Logger.Debug("load lfmb from store", zap.Int64("round", lfmb.Round),
		zap.String("hash", lfmb.Hash))

	if lfmb.MagicBlock == nil {
		// fatal
		return nil, common.NewError("load_lfb", "missing MagicBlock field")
	}

	return
}

// LoadLatestBlocksFromStore loads LFB and LFMB from store and sets them
// to corresponding fields of the sharder's Chain.
func (sc *Chain) LoadLatestBlocksFromStore(ctx context.Context) (err error) {

	var (
		round     *round.Round
		lfb, lfmb *block.Block
	)
	if lfb, round, err = sc.iterateRoundsLookingForLFB(ctx); err != nil {
		return // can't find a finalized block or has malformed rounds DB
	}

	if round.Number == 0 {
		return // ok, genesis block
	}

	Logger.Debug("load lfb from store", zap.Int64("round", lfb.Round),
		zap.String("hash", lfb.Hash))

	// check out previous magic block hash

	if lfb.LatestFinalizedMagicBlockHash == "" {
		return common.NewError("load_lfb",
			"empty LatestFinalizedMagicBlockHash field") // fatal
	}

	// related magic block
	lfmb, err = sc.loadLatestFinalizedMagicBlockFromStore(ctx, lfb)
	if err != nil {
		return
	}

	// setup all related for a non-genesis case
	sc.setupLatestBlocks(ctx, round, lfb, lfmb)
	return // ok
}
