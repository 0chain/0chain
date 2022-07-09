package sharder

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"0chain.net/core/cache"
	"0chain.net/core/ememorystore"
	"0chain.net/core/logging"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/sharder/blockstore"

	"github.com/0chain/gorocksdb"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var sharderChain = &Chain{}

/*SetupSharderChain - setup the sharder's chain */
func SetupSharderChain(c *chain.Chain) {
	sharderChain.Chain = c
	sharderChain.blockChannel = make(chan *block.Block, 1)
	sharderChain.RoundChannel = make(chan *round.Round, 1)
	blockCacheSize := 100
	sharderChain.BlockCache = cache.NewLRUCache(blockCacheSize)
	transactionCacheSize := int(c.BlockSize()) * blockCacheSize
	if transactionCacheSize > 5000 {
		transactionCacheSize = 5000
	}
	sharderChain.BlockTxnCache = cache.NewLRUCache(transactionCacheSize)
	c.SetFetchedNotarizedBlockHandler(sharderChain)
	c.SetViewChanger(sharderChain)
	c.SetAfterFetcher(sharderChain)
	c.SetMagicBlockSaver(sharderChain)
	sharderChain.BlockSyncStats = &SyncStats{}
	sharderChain.TieringStats = &MinioStats{}
	sharderChain.processingBlocks = cache.NewLRUCache(1000)
	c.RoundF = SharderRoundFactory{}
}

/*GetSharderChain - get the sharder's chain */
func GetSharderChain() *Chain {
	return sharderChain
}

type MinioStats struct {
	TotalBlocksUploaded int64
	LastRoundUploaded   int64
	LastUploadTime      time.Time
}

/*Chain - A chain structure to manage the sharder activities */
type Chain struct {
	*chain.Chain
	blockChannel   chan *block.Block
	RoundChannel   chan *round.Round
	BlockCache     cache.Cache
	BlockTxnCache  cache.Cache
	SharderStats   Stats
	BlockSyncStats *SyncStats
	TieringStats   *MinioStats

	processingBlocks *cache.LRU
	pbMutex          sync.RWMutex
}

// PushToBlockProcessor pushs the block to processor,
func (sc *Chain) PushToBlockProcessor(b *block.Block) error {
	select {
	case sc.blockChannel <- b:
		return nil
	case <-time.After(3 * time.Second):
		return errors.New("push to block processor timeout")
	}
}

/*GetRoundChannel - get the round channel where the finalized rounds are put into for further processing */
func (sc *Chain) GetRoundChannel() chan *round.Round {
	return sc.RoundChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
func (sc *Chain) SetupGenesisBlock(hash string, magicBlock *block.MagicBlock, initStates *state.InitStates) *block.Block {
	gr, gb := sc.GenerateGenesisBlock(hash, magicBlock, initStates)
	sc.AddRound(gr)
	sc.AddGenesisBlock(gb)

	// Save the round
	if err := sc.StoreRound(gr.(*round.Round)); err != nil {
		Logger.Panic("setup genesis block, save genesis round failed", zap.Error(err))
	}

	// Save the block
	err := sc.storeBlock(gb)
	if err != nil {
		Logger.Panic("setup genesis block, save genesis block failed", zap.Error(err))
	}

	if gb.MagicBlock != nil {
		var tries int64
		bs := gb.GetSummary()
		err = sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
		for err != nil {
			tries++
			Logger.Error("setup genesis block -- failed to store magic block map", zap.Any("error", err), zap.Any("tries", tries))
			time.Sleep(time.Millisecond * 100)
			err = sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
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
	b, err := blockstore.GetStore().ReadWithBlockSummary(bs)
	if err != nil {
		logging.Logger.Error("get block from store by summary failed", zap.Error(err))
		return nil, err
	}
	return b, nil
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
	var err error
	r := sc.GetSharderRound(roundNumber)
	if r == nil {
		r, err = sc.GetRoundFromStore(ctx, roundNumber)
		if err != nil {
			return "", err
		}
	}
	if r.BlockHash == "" {
		return "", fmt.Errorf("round %d has empty block hash", roundNumber)
	}
	return r.BlockHash, nil
}

// GetSharderRound - get the sharder's version of the round.
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

type blocksLoaded struct {
	lfb   *block.Block // latest finalized block with stored client state
	lfmb  *block.Block // magic block related to the lfb
	r     *round.Round // round related to the lfb
	nlfmb *block.Block // magic block equal to the lfmb or newer
}

func (sc *Chain) setupLatestBlocks(ctx context.Context, bl *blocksLoaded) (
	err error) {

	// using ClientState of genesis block

	bl.lfb.SetStateStatus(block.StateSuccessful)
	if err = sc.InitBlockState(bl.lfb); err != nil {
		bl.lfb.SetStateStatus(0)
		Logger.Info("load_lfb -- can't initialize stored block state",
			zap.Error(err))
		// return common.NewErrorf("load_lfb",
		//	"can't init block state: %v", err) // fatal
	}

	// setup lfmb first
	if err = sc.UpdateMagicBlock(bl.lfmb.MagicBlock); err != nil {
		return common.NewErrorf("load_lfb",
			"can't update magic block: %v", err) // fatal
	}

	sc.SetRandomSeed(bl.r, bl.r.GetRandomSeed())
	bl.r.Block = bl.lfb
	bl.r.BlockHash = bl.lfb.Hash

	// set LFB and LFMB of the Chain, add the block to internal Chain's map
	sc.AddLoadedFinalizedBlocks(bl.lfb, bl.lfmb, bl.r)

	// check is it notarized
	err = sc.VerifyBlockNotarization(ctx, bl.lfb)
	if err != nil {
		Logger.Error("load_lfb - verify notarization failed",
			zap.Error(err),
			zap.Int64("round", bl.lfb.Round),
			zap.String("block", bl.lfb.Hash))
		err = nil // not a real error
		return    // do nothing, if not notarized
	}
	bl.lfb.SetBlockNotarized()

	// add as notarized
	bl.lfb.SetBlockState(block.StateNotarized)
	bl.lfb, _ = bl.r.AddNotarizedBlock(bl.lfb)

	// setup nlfmb
	if bl.nlfmb != nil && bl.nlfmb.Round > bl.lfmb.Round {
		if err = sc.UpdateMagicBlock(bl.nlfmb.MagicBlock); err != nil {
			return common.NewErrorf("load_lfb",
				"can't update newer magic block: %v", err) // fatal
		}
		sc.SetLatestFinalizedMagicBlock(bl.nlfmb) // the real latest
	}

	return // everything is ok
}

func (sc *Chain) loadLatestFinalizedMagicBlockFromStore(ctx context.Context,
	lfb *block.Block) (lfmb *block.Block, err error) {

	// check out lfmb magic block hash

	if lfb.LatestFinalizedMagicBlockHash == "" {
		return nil, common.NewError("load_lfb",
			"empty LatestFinalizedMagicBlockHash field") // fatal or genesis
	}

	if lfb.LatestFinalizedMagicBlockHash == lfb.Hash {
		if lfb.MagicBlock == nil {
			// fatal
			return nil, common.NewError("load_lfb", "missing MagicBlock field")
		}
		return lfb, nil // the same
	}

	// load from store

	Logger.Debug("load_lfb (lfmb) from store",
		zap.String("block_with_magic_block_hash",
			lfb.LatestFinalizedMagicBlockHash),
		zap.Int64("block_with_magic_block_round",
			lfb.LatestFinalizedMagicBlockRound))

	lfmb, err = blockstore.GetStore().Read(lfb.LatestFinalizedMagicBlockHash,
		lfb.LatestFinalizedMagicBlockRound)
	if err != nil {
		// fatality, can't find related LFMB
		return nil, common.NewErrorf("load_lfb",
			"related magic block not found: hash: %v, err: %v", lfb.LatestFinalizedMagicBlockHash, err)
	}

	// with current implementation it's a case
	if lfmb == nil {
		// fatality, can't find related LFMB
		return nil, common.NewError("load_lfb",
			"related magic block not found (no error)")
	}

	Logger.Debug("load_lfb (lfmb) from store", zap.Int64("round", lfmb.Round),
		zap.String("hash", lfmb.Hash))

	if lfmb.MagicBlock == nil {
		// fatal
		return nil, common.NewError("load_lfb", "missing MagicBlock field")
	}

	return
}

// just get highest known MB
func (sc *Chain) loadHighestMagicBlock(ctx context.Context,
	lfb *block.Block) (lfmb *block.Block, err error) {

	if lfb.MagicBlock != nil {
		return lfb, nil
	}

	var hmbm *block.MagicBlockMap
	if hmbm, err = sc.GetHighestMagicBlockMap(ctx); err != nil {
		return nil, common.NewErrorf("load_lfb",
			"getting highest MB map: %v", err) // critical
	}

	Logger.Debug("load_lfb (lfmb), got round",
		zap.Int64("round", hmbm.BlockRound),
		zap.String("block_hash", hmbm.Hash))

	var bl *block.Block
	bl, err = sc.GetBlockFromStore(hmbm.Hash, hmbm.BlockRound)
	if err != nil {
		return nil, common.NewErrorf("load_lfb",
			"getting block with highest MB: %v", err) // critical
	}

	if bl.MagicBlock != nil {
		return bl, nil // got it
	}

	return // not found
}

func (sc *Chain) walkDownLookingForLFB(iter *gorocksdb.Iterator,
	r *round.Round) (lfb *block.Block, err error) {

	for ; iter.Valid(); iter.Prev() {
		if err = datastore.FromJSON(iter.Value().Data(), r); err != nil {
			return nil, common.NewErrorf("load_lfb",
				"decoding round info: %v", err) // critical
		}

		Logger.Debug("load_lfb, got round", zap.Int64("round", r.Number),
			zap.String("block_hash", r.BlockHash))

		lfb, err = sc.GetBlockFromStore(r.BlockHash, r.Number)
		if err != nil {
			Logger.Error("load_lfb, could not get block from store", zap.Error(err))
			continue // TODO: can we use os.IsNotExist(err) or should not
		}

		// check out required corresponding state

		// Don't check the state. It can be missing if the state had synced.
		// But it works fine anyway.

		if !sc.HasClientStateStored(lfb.ClientStateHash) {
			Logger.Warn("load_lfb, missing corresponding state",
				zap.Int64("round", r.Number),
				zap.String("block_hash", r.BlockHash))
			// we can't use this block, because of missing or malformed state
			continue
		}

		// check if lfb has full state
		if !sc.ValidateState(lfb) {
			Logger.Warn("load_lfb, lfb state missing nodes",
				zap.Int64("round", r.Number),
				zap.String("block_hash", r.BlockHash))
			// go back 50 rounds if
			if lfb.Round > 50 {
				for i := 0; i < 50; i++ {
					iter.Prev()
				}
			}
			continue
		}

		lfnb, err := sc.GetNotarizedBlockFromSharders(context.TODO(), "", lfb.Round)
		if err != nil || lfnb.Hash != lfb.Hash {
			Logger.Warn("load_lfb, see different lfb",
				zap.Int64("round", lfb.Round),
				zap.String("local lfb", lfb.Hash),
				zap.String("remote lfb", lfnb.Hash))
			continue
		}

		return // got it
	}

	return nil, common.NewError("load_lfb", "no valid lfb found")
}

// iterate over rounds from latest to zero looking for LFB and ignoring
// missing blocks in blockstore
func (sc *Chain) iterateRoundsLookingForLFB(ctx context.Context) *blocksLoaded {
	bl := new(blocksLoaded)

	var (
		remd = datastore.GetEntityMetadata("round")
		rctx = ememorystore.WithEntityConnection(ctx, remd)

		// the error is internal, we are using logs and rolling back to
		// genesis blocks on error
		err error
	)
	defer ememorystore.Close(rctx)

	var (
		conn = ememorystore.GetEntityCon(rctx, remd)
		iter = conn.Conn.NewIterator(conn.ReadOptions)
	)
	defer iter.Close()

	bl.r = remd.Instance().(*round.Round) //

	iter.SeekToLast() // from last

	if !iter.Valid() {
		return nil // the nil is 'use genesis'
	}

	if bl.lfb, err = sc.walkDownLookingForLFB(iter, bl.r); err != nil {
		Logger.Warn("load_lfb, can't load lfb",
			zap.Int64("round_stopped", bl.r.Number),
			zap.Error(err))
		return nil // the nil is 'use genesis'
	}

	magicBlockMiners := sc.GetMiners(bl.r.GetRoundNumber())
	bl.r.SetRandomSeedForNotarizedBlock(bl.lfb.GetRoundRandomSeed(), magicBlockMiners.Size())

	// and then, check out related LFMB can be missing
	bl.lfmb, err = sc.loadLatestFinalizedMagicBlockFromStore(ctx, bl.lfb)
	if err != nil {
		Logger.Warn("load_lfb, missing corresponding lfmb",
			zap.Int64("round", bl.r.Number),
			zap.String("block_hash", bl.r.BlockHash),
			zap.String("lfmb_hash", bl.lfb.LatestFinalizedMagicBlockHash))
		// we can't skip to starting round, because we don't know it
		return nil // the nil is 'use genesis'
	}

	// but the lfmb can be less than real latest finalized magic block,
	// the lfmb is just magic block related to the lfb, for example for
	// 502 round lfmb is 251, but lfmb of 501 round we already have and
	// it is the latest magic block, we have to load it and setup

	// using another round instance
	bl.nlfmb, err = sc.loadHighestMagicBlock(ctx, bl.lfb)
	if err != nil {
		Logger.Warn("load_lfb, loading highest magic block", zap.Error(err))
	}

	return bl // got them all (or excluding the nlfmb)
}

// LoadLatestBlocksFromStore loads LFB and LFMB from store and sets them
// to corresponding fields of the sharder's Chain.
func (sc *Chain) LoadLatestBlocksFromStore(ctx context.Context) (err error) {

	var bl = sc.iterateRoundsLookingForLFB(ctx)

	if bl == nil || bl.r == nil || bl.r.Number == 0 || bl.r.Number == 1 {
		return // use genesis blocks
	}

	Logger.Debug("load_lfb from store",
		zap.Int64("round", bl.lfb.Round),
		zap.String("hash", bl.lfb.Hash),
		zap.Int64("lfmb", bl.lfmb.Round))

	if bl.nlfmb != nil && bl.nlfmb.Round != bl.lfmb.Round {
		Logger.Debug("load_lfb from store (nlfmb)",
			zap.Int64("round", bl.nlfmb.Round))
	}

	// setup all related for a non-genesis case
	return sc.setupLatestBlocks(ctx, bl)
}

// SaveMagicBlockHandler used on sharder startup to save received
// magic blocks. It's required to be able to load previous state.
func (sc *Chain) SaveMagicBlockHandler(ctx context.Context,
	b *block.Block) (err error) {

	Logger.Info("save received magic block verifying chain",
		zap.Int64("round", b.Round), zap.String("hash", b.Hash),
		zap.Int64("starting_round", b.MagicBlock.StartingRound),
		zap.String("mb_hash", b.MagicBlock.Hash))

	if err = sc.storeBlock(b); err != nil {
		return
	}
	var bs = b.GetSummary()
	return sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
}

// SaveMagicBlock function.
func (sc *Chain) SaveMagicBlock() chain.MagicBlockSaveFunc {
	return chain.MagicBlockSaveFunc(sc.SaveMagicBlockHandler)
}

func (sc *Chain) ValidateState(b *block.Block) bool {
	if err := sc.InitBlockState(b); err != nil {
		Logger.Warn("load_lfb, init block state failed", zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return false
	}

	missing, err := b.ClientState.HasMissingNodes(context.Background())
	if err != nil {
		Logger.Warn("load_lfb, find missing nodes failed",
			zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return false
	}

	if missing {
		Logger.Warn("load_lfb, lfb has missing nodes",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return false
	}

	return true
}
