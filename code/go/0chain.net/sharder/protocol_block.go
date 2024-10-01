package sharder

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/config"
	"0chain.net/core/util/waitgroup"
	"github.com/rcrowley/go-metrics"

	"0chain.net/sharder/blockstore"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	. "github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

var blockSaveTimer metrics.Timer
var bsHistogram metrics.Histogram
var syncCatchupTime metrics.Histogram

func init() {
	blockSaveTimer = metrics.GetOrRegisterTimer("block_save_time", nil)
	bsHistogram = metrics.GetOrRegisterHistogram("bs_histogram", nil, metrics.NewUniformSample(1024))
	syncCatchupTime = metrics.GetOrRegisterHistogram("sync_catch_up_time", nil, metrics.NewUniformSample(1024))
}

/*UpdatePendingBlock - update the pending block */
func (sc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {

}

/*UpdateFinalizedBlock - updates the finalized block */
func (sc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) error {
	fr := sc.GetRoundClone(b.Round)
	if fr == nil {
		fr = round.NewRound(b.Round)
	}

	b = b.Clone()
	fr.Finalize(b)
	wg := waitgroup.New(5)
	Logger.Info("update finalized block",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("round block hash", fr.GetBlockHash()),
		zap.Int64("lf_round", sc.GetLatestFinalizedBlock().Round),
		zap.Int64("current_round", sc.GetCurrentRound()))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}

	if err := sc.BlockCache.Add(b.Hash, b); err != nil {
		Logger.Panic(
			fmt.Sprintf("update finalized block, add block to cache failed round: %d, block: %s, error: %s",
				b.Round, b.Hash, err.Error()))
	}

	self := node.GetSelfNode(ctx)
	bsHistogram.Update(int64(len(b.Txns)))
	self.Underlying().Info.AvgBlockTxns = int(math.Round(bsHistogram.Mean()))

	wg.Run("store transactions", b.Round, func() error {
		if err := sc.StoreTransactions(b); err != nil {
			Logger.Panic(fmt.Sprintf("db store transaction failed. Error: %v", err))
		}
		return nil
	})

	wg.Run("store block summary", b.Round, func() error {
		if err := sc.StoreBlockSummaryFromBlock(b); err != nil {
			Logger.Panic(
				fmt.Sprintf("db error (store block summary) round: %d, block: %s, error: %s", b.Round, b.Hash, err.Error()))
		}

		return nil
	})

	if b.MagicBlock != nil {
		wg.Run("store magic block", b.Round, func() error {
			bs := b.GetSummary()
			if err := sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap()); err != nil {
				Logger.DPanic("failed to store magic block map", zap.Error(err))
			}
			return nil
		})
	}

	if sc.IsBlockSharder(b, self.Underlying()) {
		wg.Run("store block", b.Round, func() error {
			sc.SharderStats.ShardedBlocksCount++
			ts := time.Now()
			if err := blockstore.GetStore().Write(b); err != nil {
				Logger.Panic(fmt.Sprintf("store block failed, round: %d, error: %v", b.Round, err))
			}

			duration := time.Since(ts)
			blockSaveTimer.UpdateSince(ts)
			p95 := blockSaveTimer.Percentile(.95)
			if blockSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
				Logger.Warn("block save - slow", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
			}
			return nil
		})
	}

	go sc.DeleteRoundsBelow(b.Round)

	// Wait for all group goroutines to exit and check error before continue. Otherwise, if panic
	// happens in any of the goroutine, we will see wait() finish and the code will continue to persist the rounds
	// rather than before the panic exit the program completely. While we don't want the round to be store actually.
	if err := wg.Wait(); err != nil {
		if waitgroup.ErrIsPanic(err) {
			// continue throw panic up so that it behaviors the same as before.
			panic(err)
		}

		Logger.Error("update finalized block failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Error(err))
		return err
	}

	// Persist LFB, do this after all above succeed to make sure the LFB will not be set
	// if panic happens. If we do it in goroutine the same as above, as long as round and block
	// summary is saved successfully, even other process panic, restarting the sharder would
	// consider this block as LFB, but those data didn't get saved previously will be lost.
	if err := sc.StoreRound(fr.(*round.Round)); err != nil {
		Logger.Panic("db error (save round)", zap.Int64("round", fr.GetRoundNumber()), zap.Error(err))
	}

	//nolint:errcheck
	notifyConductor(b)

	// return if view change is off
	if !sc.IsViewChangeEnabled() {
		Logger.Debug("update finalized blocks storage success",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return nil
	}

	pn, err := sc.GetPhaseOfBlock(b)
	if err != nil && err != util.ErrValueNotPresent {
		logging.Logger.Error("[mvc] update finalized block - get phase of block failed", zap.Error(err))
		return err
	}

	if pn == nil {
		return nil
	}

	logging.Logger.Debug("[mvc] update finalized block - send phase node",
		zap.Int64("round", b.Round),
		zap.Int64("start_round", pn.StartRound),
		zap.String("phase", pn.Phase.String()))
	go sc.SendPhaseNode(context.Background(), chain.PhaseEvent{Phase: *pn})

	Logger.Debug("update finalized blocks storage success",
		zap.Int64("round", b.Round), zap.String("block", b.Hash))
	return nil
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
