package sharder

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/config"
	"0chain.net/sharder/blockstore"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var blockSaveTimer metrics.Timer
var bsHistogram metrics.Histogram

func init() {
	blockSaveTimer = metrics.GetOrRegisterTimer("block_save_time", nil)
	bsHistogram = metrics.GetOrRegisterHistogram("bs_histogram", nil, metrics.NewUniformSample(1024))
}

/*UpdatePendingBlock - update the pending block */
func (sc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {

}

/*UpdateFinalizedBlock - updates the finalized block */
func (sc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	fr := sc.GetRound(b.Round)
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("lf_round", sc.GetLatestFinalizedBlock().Round), zap.Any("current_round", sc.GetCurrentRound()))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	if err := sc.BlockCache.Add(b.Hash, b); err != nil {
		Logger.Warn("update finalized block, add block to cache failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Error(err))
	}

	if fr == nil {
		fr = round.NewRound(b.Round)
	}
	fr.Finalize(b)
	bsHistogram.Update(int64(len(b.Txns)))
	node.Self.Underlying().Info.AvgBlockTxns = int(math.Round(bsHistogram.Mean()))
	err := sc.StoreTransactions(b)
	if err != nil {
		Logger.Error("db store transaction failed", zap.Error(err))
	}
	err = sc.StoreBlockSummaryFromBlock(b)
	if err != nil {
		Logger.Error("db error (store block summary)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	}
	self := node.GetSelfNode(ctx)
	if b.MagicBlock != nil {
		bs := b.GetSummary()
		err = sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
		if err != nil {
			Logger.DPanic("failed to store magic block map", zap.Any("error", err))
		}
	}
	if sc.IsBlockSharder(b, self.Underlying()) {
		sc.SharderStats.ShardedBlocksCount++
		ts := time.Now()
		if err := blockstore.GetStore().Write(b); err != nil {
			Logger.Error("store block failed",
				zap.Int64("round", b.Round),
				zap.Error(err))
		}
		duration := time.Since(ts)
		blockSaveTimer.UpdateSince(ts)
		p95 := blockSaveTimer.Percentile(.95)
		if blockSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
			Logger.Error("block save - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
		}
	}
	if frImpl, ok := fr.(*round.Round); ok {
		err := sc.StoreRound(frImpl)
		if err != nil {
			Logger.Error("db error (save round)", zap.Int64("round", fr.GetRoundNumber()), zap.Error(err))
		}
	}
	sc.DeleteRoundsBelow(b.Round)
}

func (sc *Chain) ViewChange(ctx context.Context, b *block.Block) error { //nolint: unused
	if !sc.ChainConfig.IsViewChangeEnabled() {
		return nil
	}

	mb := b.MagicBlock
	if mb == nil {
		return nil // no MB, no VC
	}

	if err := sc.UpdateMagicBlock(mb); err != nil {
		return err
	}

	sc.SetLatestFinalizedMagicBlock(b)
	return nil
}

// The hasRelatedMagicBlock reports true if the Chain has MB related to the
// given block (checked by round number). It never checks persistent store,
// checking MB in round magic block store (memory) only.
func (sc *Chain) hasRelatedMagicBlock(b *block.Block) (ok bool) {
	var (
		relatedmbr = b.LatestFinalizedMagicBlockRound
		mb         = sc.GetMagicBlock(b.Round)
	)
	if mb.StartingRound != relatedmbr {
		Logger.Warn("do not have related MB",
			zap.Int64("mb", mb.StartingRound),
			zap.Int64("relatedMb", relatedmbr))
	}
	return mb.StartingRound == relatedmbr
}

// pull related magic block if missing (sync)
func (sc *Chain) pullRelatedMagicBlock(ctx context.Context, b *block.Block) (
	err error) {

	if sc.hasRelatedMagicBlock(b) {
		return // already have the MB, nothing to do
	}

	// TODO (sfxdx): get magic block by number/hash/round to be sure its
	//               really related, not just latest
	if err = sc.UpdateLatestMagicBlockFromSharders(ctx); err != nil {
		return // got error
	}

	if !sc.hasRelatedMagicBlock(b) {
		return fmt.Errorf("can't pull related magic block for %d", b.Round)
	}

	return
}

// AfterFetch used to pull related MB (if missing) for blocks fetched by
// AsyncFetch* function in LFB-tickets worker. E.g. for blocks kicked by
// a LFB ticket.
func (sc *Chain) AfterFetch(ctx context.Context, b *block.Block) (err error) {

	// pull related magic block if missing
	if err = sc.pullRelatedMagicBlock(ctx, b); err != nil {
		Logger.Error("after_fetch -- pulling related magic block",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.Error(err))
		return
	}

	// ok, already have or just pulled, check out LFB

	var lfb = sc.GetLatestFinalizedBlock()
	if lfb.Round < b.Round {
		Logger.Warn("after_fetch - newer finalize round",
			zap.Int64("round", b.Round),
			zap.Int64("lfb round", lfb.Round))
	}

	return // everything is done
}

func (sc *Chain) processBlock(ctx context.Context, b *block.Block) error {
	if !sc.cacheProcessingBlock(b.Hash) {
		Logger.Debug("process block, being processed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash))
		return nil
	}

	Logger.Debug("process notarized block",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash))

	ts := time.Now()
	defer func() {
		sc.removeProcessingBlock(b.Hash)
		Logger.Debug("process notarized block end",
			zap.Int64("round", b.Round),
			zap.Duration("duration", time.Since(ts)))
	}()
	var er = sc.GetRound(b.Round)
	if er == nil {
		var r = round.NewRound(b.Round)
		er, _ = sc.AddRound(r).(*round.Round)
		if b.GetRoundRandomSeed() == 0 {
			Logger.Error("process block - block has no seed",
				zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return fmt.Errorf("block has no seed")
		}
		sc.SetRandomSeed(er, b.GetRoundRandomSeed()) // incorrect round seed ?
	}

	// pull related magic block if missing
	var err error
	if err = sc.pullRelatedMagicBlock(ctx, b); err != nil {
		Logger.Error("pulling related magic block", zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("related mbr", b.LatestFinalizedMagicBlockRound))
		return fmt.Errorf("could not pull related magic block, err: %v", err)
	}

	if err = b.Validate(ctx); err != nil {
		Logger.Error("block validation", zap.Any("round", b.Round),
			zap.Any("hash", b.Hash), zap.Error(err))
		return fmt.Errorf("validate block failed, err: %v", err)
	}

	err = sc.VerifyBlockNotarization(ctx, b)
	if err != nil {
		Logger.Error("notarization verification failed",
			zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash))
		return fmt.Errorf("verify block notarization failed, err: %v", err)
	}

	//TODO remove it since verify block adds this block to round
	b, _ = sc.AddNotarizedBlockToRound(er, b)
	sc.SetRoundRank(er, b)
	Logger.Info("received notarized block", zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("client_state", util.ToHex(b.ClientStateHash)))
	return sc.AddNotarizedBlock(ctx, er, b)
}

func (sc *Chain) syncRoundSummary(ctx context.Context, roundNum int64, roundRange int64, scan HealthCheckScan) *round.Round {
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scan)
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(roundNum, 10))
	params.Add("range", strconv.FormatInt(roundRange, 10))

	// Send request to all sharders requesting round summary
	rs := sc.requestForRoundSummaries(ctx, params)

	if rs != nil {
		// Received reply for roundRange of blocks starting at roundNum
		sc.storeRoundSummaries(ctx, rs)
	} else {
		HCLogger.Info("HC-MissingObject",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("object", "RoundSummaries"),
			zap.Int64("round", roundNum),
			zap.Int64("range", roundRange))
		return nil
	}

	// Check the block we are interested in.
	r, ok := sc.hasRoundSummary(ctx, roundNum)
	if ok {
		return r
	}
	// Have round summary - Request for round information
	params.Del("range")
	r = sc.requestForRound(ctx, params)
	if sc.isValidRound(r) {
		err := sc.StoreRound(r)
		if err != nil {
			Logger.Error("HC-DSWriteFailure",
				zap.String("object", "RoundSummary"),
				zap.Int64("cycle", cc.CycleCount),
				zap.Int64("round", roundNum),
				zap.Error(err))
			// Return failure
			r = nil
		}
	} else {
		// Missing round summary. Log it.
		HCLogger.Info("HC-MissingObject",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("object", "RoundSummary"),
			zap.Int64("round", roundNum))
		r = nil
	}
	return r
}

func (sc *Chain) syncBlockSummary(ctx context.Context, r *round.Round, roundRange int64, scan HealthCheckScan) *block.BlockSummary {
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scan)
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(r.Number, 10))
	params.Add("range", strconv.FormatInt(roundRange, 10))

	// Step 1: Request range of
	bs := sc.requestForBlockSummaries(ctx, params)
	if bs != nil {
		sc.storeBlockSummaries(ctx, bs)
	}

	// Check if the block summary was acquired
	blockS, ok := sc.hasBlockSummary(ctx, r.BlockHash)
	if ok {
		return blockS
	}
	// No block summary for this round.
	params.Del("round")
	params.Del("range")
	params.Add("hash", r.BlockHash)

	blockS = sc.requestForBlockSummary(ctx, params)
	if blockS != nil {
		// Store errors will be displayed by the function.
		sc.storeBlockSummary(ctx, blockS)
	} else {
		HCLogger.Info("HC-MissingObject",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("object", "BlockSummary"),
			zap.Int64("cycle", cc.CycleCount),
			zap.Int64("round", r.Number),
			zap.String("hash", r.BlockHash))
	}
	return blockS
}

func (sc *Chain) requestBlock(ctx context.Context, r *round.Round) *block.Block {
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(r.Number, 10))
	params.Add("hash", r.BlockHash)

	return sc.requestForBlock(ctx, params, r)
}

//func (sc *Chain) storeBlock(ctx context.Context, r *round.Round, canShard bool) *block.Block {
//	bss := sc.BlockSyncStats
//	params := &url.Values{}
//	params.Add("round", strconv.FormatInt(r.Number, 10))
//	params.Add("hash", r.BlockHash)
//
//	var b *block.Block
//	b = sc.requestForBlock(ctx, params, r)
//	if b == nil {
//		Logger.Info("health-check: MissingObject",
//			zap.String("object", "Block"),
//			zap.Int64("cycle", bss.CycleCount),
//			zap.Int64("round", r.Number),
//			zap.String("hash", r.BlockHash))
//		return nil
//	}
//	if canShard {
//		// Save the block
//		err := sc.storeBlock(ctx, b)
//		if err != nil {
//			Logger.Error("health-check: DataStoreWriteFailure",
//				zap.String("object", "block"),
//				zap.Int64("cycle", bss.CycleCount),
//				zap.Int64("round", r.Number),
//				zap.Error(err))
//		}
//	}
//	return b
//}

func (sc *Chain) isValidRound(r *round.Round) bool {
	if r == nil {
		return false
	}
	if r.Number <= 0 {
		return false
	}
	if r.BlockHash == "" {
		return false
	}
	return true
}

func (sc *Chain) requestForRoundSummaries(ctx context.Context, params *url.Values) *RoundSummaries {
	rsC := make(chan *RoundSummaries, 1)
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		roundSummaries, ok := entity.(*RoundSummaries)
		if !ok {
			Logger.Error("received invalid round summaries")
			return nil, common.NewError("request_for_round_summaries", "invalid round summaries")
		}
		select {
		case rsC <- roundSummaries:
		default:
		}
		cancel()
		return roundSummaries, nil
	}
	sc.RequestEntityFromShardersOnMB(cctx, sc.GetCurrentMagicBlock(), RoundSummariesRequestor, params, handler)
	var rs *RoundSummaries
	select {
	case rs = <-rsC:
	default:
	}
	return rs
}

func (sc *Chain) requestForRound(ctx context.Context, params *url.Values) *round.Round {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	rC := make(chan *round.Round, 1)
	handler := func(_ context.Context, entity datastore.Entity) (interface{}, error) {
		r, ok := entity.(*round.Round)
		if !ok {
			Logger.Error("received invalid round entity")
			return nil, common.NewError("request_for_round", "received invalid round entity")
		}

		if sc.isValidRound(r) {
			select {
			case rC <- r:
			default:
			}
			cancel()
			return r, nil
		}
		return nil, common.NewError("request_for_round", "invalid response round")
	}

	sc.RequestEntityFromShardersOnMB(cctx, sc.GetCurrentMagicBlock(), RoundRequestor, params, handler)
	var r *round.Round
	select {
	case r = <-rC:
	default:
	}

	return r
}

func (sc *Chain) requestForBlockSummaries(ctx context.Context, params *url.Values) *BlockSummaries {
	bsC := make(chan *BlockSummaries, 1)
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	handler := func(_ context.Context, entity datastore.Entity) (interface{}, error) {
		bs, ok := entity.(*BlockSummaries)
		if !ok {
			Logger.Error("received invalid block summaries", zap.String("round", params.Get("round")), zap.String("range", params.Get("range")))
			return nil, common.NewError("request_for_block_summaries", "invalid block summaries")
		}
		select {
		case bsC <- bs:
		default:
		}
		cancel()
		return bs, nil
	}
	sc.RequestEntityFromShardersOnMB(cctx, sc.GetCurrentMagicBlock(), BlockSummariesRequestor, params, handler)
	var bs *BlockSummaries
	select {
	case bs = <-bsC:
	default:
	}
	return bs
}

func (sc *Chain) requestForBlockSummary(ctx context.Context, params *url.Values) *block.BlockSummary {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	bsC := make(chan *block.BlockSummary, 1)
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		bs, ok := entity.(*block.BlockSummary)
		if !ok {
			Logger.Error("received invalid block summary entity", zap.String("hash", params.Get("hash")))
			return nil, common.NewError("request_for_block_summary", "invalid block summary entity")
		}

		select {
		case bsC <- bs:
		default:
		}
		cancel()
		return bs, nil
	}
	sc.RequestEntityFromShardersOnMB(cctx, sc.GetCurrentMagicBlock(), BlockSummaryRequestor, params, handler)
	var bs *block.BlockSummary
	select {
	case bs = <-bsC:
	default:
	}
	return bs
}

func (sc *Chain) requestForBlock(ctx context.Context, params *url.Values, r *round.Round) *block.Block {
	self := node.GetSelfNode(ctx)

	_, nodes := sc.CanShardBlockWithReplicators(r.Number, r.BlockHash,
		self.Underlying())

	if len(nodes) == 0 {
		Logger.Info("no replicators for this block (lost the block)", zap.Int64("round", r.Number))
	}

	var requestNode *node.Node
	for _, n := range nodes {
		if self.IsEqual(n) {
			continue
		}
		requestNode = n
		var b *block.Block
		handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
			blockEntity, ok := entity.(*block.Block)
			if !ok {
				Logger.Error("invalid request block", zap.Int64("round", r.Number))
				return nil, nil
			}
			err := blockEntity.Validate(ctx)
			if err == nil {
				b = blockEntity
				return blockEntity, nil
			}
			return nil, err
		}
		requestNode.RequestEntityFromNode(ctx, BlockRequestor, params, handler)
		if b != nil {
			return b
		}
		Logger.Info("request round info - block is nil", zap.Int64("round", r.Number), zap.String("block-hash", r.BlockHash))
	}
	return nil
}

func (sc *Chain) storeRoundSummaries(ctx context.Context, rs *RoundSummaries) {
	//roundEntityMetadata := datastore.GetEntityMetadata("round")
	//
	//rsEntities := make([]datastore.Entity, 0, 1)
	Logger.Debug("HC-StoreRoundSummaries",
		zap.Int("round-count", len(rs.RSummaryList)))

	for _, roundS := range rs.RSummaryList {
		if roundS != nil {
			_, present := sc.hasRoundSummary(ctx, roundS.Number)
			// Store only rounds that are not present.
			if !present {
				Logger.Debug("HC-StoreRoundSummaries",
					zap.String("object", "RoundSummary"),
					zap.Int64("round", roundS.Number),
					zap.String("hash", roundS.BlockHash))
				err := sc.StoreRound(roundS)
				if err != nil {
					Logger.Error("store round failed", zap.Error(err))
				}
			}
		} else {
			Logger.Debug("HC-StoreRoundSummaries",
				zap.String("round", "nil"))
		}
	}

	//if len(rsEntities) > 0 {
	//	rsStore := roundEntityMetadata.GetStore()
	//	rsctx := ememorystore.WithEntityConnection(ctx, roundEntityMetadata)
	//	defer ememorystore.Close(rsctx)
	//	err := rsStore.MultiWrite(rsctx, roundEntityMetadata, rsEntities)
	//	if err != nil {
	//		Logger.Info("write round summaries failed", zap.Error(err))
	//	}
	//	Logger.Info("write round summaries successful")
	//}
}

func (sc *Chain) storeBlockSummaries(ctx context.Context, bs *BlockSummaries) {
	Logger.Debug("HC-StoreBlockSummaries",
		zap.Int("round-count", len(bs.BSummaryList)))

	for _, blockS := range bs.BSummaryList {
		if blockS != nil {
			_, present := sc.hasBlockSummary(ctx, blockS.Hash)
			if !present {
				Logger.Debug("HC-StoreBlockSummaries",
					zap.String("object", "BlockSummary"),
					zap.Int64("block", blockS.Round),
					zap.String("hash", blockS.Hash))
				storeError := sc.StoreBlockSummary(ctx, blockS)
				if storeError != nil {
					HCLogger.Error("HC-StoreBlockSummary",
						zap.Int64("round", blockS.Round),
						zap.String("hash", blockS.Hash),
						zap.Error(storeError))
				}
			}
		} else {
			Logger.Debug("HC-StoreBlockSummaries", zap.String("blockSummary", "nil"))
		}
	}
	//blockSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	//
	//bsEntities := make([]datastore.Entity, 0, 1)
	//for _, blockS := range bs.BSummaryList {
	//	if blockS != nil {
	//		bsEntities = append(bsEntities, blockS)
	//	}
	//}
	//
	//if len(bsEntities) > 0 {
	//	bsStore := blockSummaryEntityMetadata.GetStore()
	//	bsctx := ememorystore.WithEntityConnection(ctx, blockSummaryEntityMetadata)
	//	defer ememorystore.Close(bsctx)
	//	err := bsStore.MultiWrite(bsctx, blockSummaryEntityMetadata, bsEntities)
	//	if err != nil {
	//		Logger.Info("write block summaries failed", zap.Error(err))
	//	}
	//	Logger.Info("write block summaries successful")
	//}
}

func (sc *Chain) storeBlockSummary(ctx context.Context, bs *block.BlockSummary) {
	var err error
	for {
		err = sc.StoreBlockSummary(ctx, bs)
		if err == nil {
			return
		}
		Logger.Error("db error (save block summary)", zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.Error(err))
		time.Sleep(time.Second)
	}
}

func (sc *Chain) storeBlock(b *block.Block) error {
	var err error
	err = blockstore.GetStore().Write(b)
	if err == nil {
		sc.SharderStats.RepairBlocksCount++
	} else {
		Logger.Error("save block failed",
			zap.Int64("round", b.Round),
			zap.Error(err))
		sc.SharderStats.RepairBlocksFailure++
	}
	if b.MagicBlock != nil {
		bs := b.GetSummary()
		err = sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
		if err != nil {
			return err
		}
	}
	return err
}

func (sc *Chain) storeBlockTransactions(ctx context.Context, b *block.Block) error {
	err := sc.StoreTransactions(b)
	//	Logger.Error(caller,
	//		zap.Int64("round", b.Round),
	//		zap.String("block", b.Hash),
	//		zap.Error(err))
	//}
	return err
}

// NotarizedBlockFetched -
func (sc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {
	// sc.processBlock(ctx, b)
}
