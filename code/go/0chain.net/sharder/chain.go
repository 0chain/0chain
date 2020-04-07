package sharder

import (
	"context"
	// "strconv"
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
	lfb, lfmb *block.Block) (err error) {

	// using ClientState of genesis block

	if err = sc.InitBlockState(lfb); err != nil {
		return common.NewError("load_lfb",
			"can't init block state: "+err.Error()) // fatal
	}
	lfb.SetStateStatus(block.StateSuccessful)

	if err = sc.UpdateMagicBlock(lfmb.MagicBlock); err != nil {
		return common.NewError("load_lfb",
			"can't update magic block: "+err.Error()) // fatal
	}
	sc.UpdateNodesFromMagicBlock(lfmb.MagicBlock)

	sc.SetRandomSeed(round, round.GetRandomSeed())
	round.ComputeMinerRanks(lfmb.MagicBlock.Miners)
	round.Block = lfb

	// set LFB and LFMB of the Chain, add the block to internal Chain's map
	sc.AddLoadedFinalizedBlocks(lfb, lfmb)

	// check is it notarized
	err = sc.VerifyNotarization(ctx, lfb.Hash, lfb.GetVerificationTickets(),
		round.GetRoundNumber())
	if err != nil {
		err = nil // not a real error
		return    // do nothing, if not notarized
	}

	// add as notarized
	lfb.SetBlockState(block.StateNotarized)
	round.AddNotarizedBlock(lfb)
	return
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

	Logger.Debug("load lfmb from store",
		zap.String("block_with_magic_block_hash",
			lfb.LatestFinalizedMagicBlockHash))

	lfmb, err = blockstore.GetStore().Read(lfb.LatestFinalizedMagicBlockHash)
	if err != nil {
		// fatality, can't find related LFMB
		return nil, common.NewError("load_lfb",
			"related magic block not found: "+err.Error())
	}

	// with current implementation it's a case
	if lfmb == nil {
		// fatality, can't find related LFMB
		return nil, common.NewError("load_lfb",
			"related magic block not found (no error)")
	}

	Logger.Debug("load lfmb from store", zap.Int64("round", lfmb.Round),
		zap.String("hash", lfmb.Hash))

	if lfmb.MagicBlock == nil {
		// fatal
		return nil, common.NewError("load_lfb", "missing MagicBlock field")
	}

	return
}

// iterate over rounds from latest to zero looking for LFB and ignoring
// missing blocks in blockstore
func (sc *Chain) iterateRoundsLookingForLFB(ctx context.Context) (
	lfb, lfmb *block.Block, r *round.Round, err error) {

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
		return nil, nil, r, nil // round 0 (genesis, initial)
	}

	// loop

	for ; iter.Valid(); iter.Prev() {
		if err = datastore.FromJSON(iter.Value().Data(), r); err != nil {
			return nil, nil, nil, common.NewError("load_lfb",
				"decoding round info: "+err.Error()) // critical
		}

		Logger.Debug("load_lfb, got round", zap.Int64("round", r.Number),
			zap.String("block_hash", r.BlockHash))

		lfb, err = sc.GetBlockFromStore(r.BlockHash, r.Number)
		if err != nil {
			continue // TODO: can we use os.IsNotExist(err) or should not
		}

		// check out required corresponding state

		if !sc.HasClientStateStored(lfb.ClientStateHash) {
			Logger.Warn("load_lfb, missing corresponding state",
				zap.Int64("round", r.Number),
				zap.String("block_hash", r.BlockHash))
			// we can't use this block, because of missing or malformed state
			continue
		}

		// and then, check out related LFMB can be missing
		lfmb, err = sc.loadLatestFinalizedMagicBlockFromStore(ctx, lfb)
		if err != nil {
			Logger.Warn("load_lfb, missing corresponding lfmb",
				zap.Int64("round", r.Number),
				zap.String("block_hash", r.BlockHash),
				zap.String("lfmb_hash", lfb.LatestFinalizedMagicBlockHash))
			// we can't skip to starting round, because we don't know it
			continue
		}

		return // got them
	}

	if r.Number == 1 {
		r.Number = 0
		return nil, nil, r, nil
	}

	// not found

	return nil, nil, nil, common.NewError("load_lfb", "lfb not found")
}

// -------------------------------------------------------------------------- //
// frozen until a sharder can receive blocks, while the
// sharder doesn't have corresponding MB
//

// func itoa(n int64) string {
// 	return strconv.FormatInt(n, 10)
// }

// func (sc *Chain) loadPreviousMagicBlock(ctx context.Context, n int64) (
// 	plfmb *block.Block, err error) {

// 	var mbm *block.MagicBlockMap
// 	if mbm, err = sc.GetMagicBlockMap(ctx, itoa(n)); err != nil {
// 		return nil, common.NewError("load_lfb",
// 			"related previous magic block not found in map: "+err.Error())
// 	}

// 	plfmb, err = blockstore.GetStore().Read(mbm.Hash)
// 	if err != nil {
// 		return nil, common.NewError("load_lfb",
// 			"related previous magic block not found: "+err.Error())
// 	}

// 	return
// }

// -------------------------------------------------------------------------- //

// LoadLatestBlocksFromStore loads LFB and LFMB from store and sets them
// to corresponding fields of the sharder's Chain.
func (sc *Chain) LoadLatestBlocksFromStore(ctx context.Context) (err error) {

	var (
		round     *round.Round
		lfb, lfmb *block.Block
	)
	if lfb, lfmb, round, err = sc.iterateRoundsLookingForLFB(ctx); err != nil {
		return // can't find a finalized block or has malformed rounds DB
	}

	if round.Number == 0 {
		return // ok, genesis block
	}

	Logger.Debug("load lfb from store", zap.Int64("round", lfb.Round),
		zap.String("hash", lfb.Hash))

	// ---------------------------------------------------------------------- //
	// frozen until a sharder can receive blocks, while the
	// sharder doesn't have corresponding MB
	//

	// // load and setup previous magic block if any
	// if lfmb.MagicBlock.MagicBlockNumber > 1 {
	// 	var plfmb *block.Block
	// 	plfmb, err = sc.loadPreviousMagicBlock(ctx,
	// 		lfmb.MagicBlock.MagicBlockNumber-1)
	// 	if err != nil {
	// 		return
	// 	}

	// 	if plfmb.MagicBlock == nil {
	// 		return common.NewError("load_lfb",
	// 			"missing MagicBlock field of block of previous magic block")

	// 	}

	// sc.SetInitialPreviousMagicBlock(plfmb.MagicBlock)
	// }
	// ---------------------------------------------------------------------- //

	// setup all related for a non-genesis case
	return sc.setupLatestBlocks(ctx, round, lfb, lfmb)
}

// SaveMagicBlockHandler used on sharder startup to save received
// magic blocks. It's required to be able to load previous state.
func (sc *Chain) SaveMagicBlockHandler(ctx context.Context,
	b *block.Block) (err error) {

	Logger.Info("save received magic block verifying chain",
		zap.Int64("round", b.Round), zap.String("hash", b.Hash),
		zap.Int64("starting_round", b.MagicBlock.StartingRound),
		zap.String("mb_hash", b.MagicBlock.Hash))

	if err = sc.storeBlock(ctx, b); err != nil {
		return
	}
	var bs = b.GetSummary()
	return sc.StoreMagicBlockMapFromBlock(ctx, bs.GetMagicBlockMap())
}

// SaveMagicBlock function.
func (sc *Chain) SaveMagicBlock() chain.MagicBlockSaveFunc {
	return chain.MagicBlockSaveFunc(sc.SaveMagicBlockHandler)
}
