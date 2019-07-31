package sharder

import (
	"context"
	"math"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
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
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("lf_round", sc.GetLatestFinalizedBlock().Round), zap.Any("current_round", sc.CurrentRound))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	sc.BlockCache.Add(b.Hash, b)
	if fr == nil {
		fr = round.NewRound(b.Round)
	}
	fr.Finalize(b)
	bsHistogram.Update(int64(len(b.Txns)))
	node.Self.Node.Info.AvgBlockTxns = int(math.Round(bsHistogram.Mean()))
	sc.StoreTransactions(ctx, b)
	err := sc.StoreBlockSummaryFromBlock(ctx, b)
	if err != nil {
		Logger.Error("db error (store block summary)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	}
	self := node.GetSelfNode(ctx)
	if sc.IsBlockSharder(b, self.Node) {
		sc.SharderStats.ShardedBlocksCount++
		ts := time.Now()
		blockstore.GetStore().Write(b)
		duration := time.Since(ts)
		blockSaveTimer.UpdateSince(ts)
		p95 := blockSaveTimer.Percentile(.95)
		if blockSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
			Logger.Error("block save - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
		}
	}
	if frImpl, ok := fr.(*round.Round); ok {
		err := sc.StoreRound(ctx, frImpl)
		if err != nil {
			Logger.Error("db error (save round)", zap.Int64("round", fr.GetRoundNumber()), zap.Error(err))
		}
	}
	sc.DeleteRoundsBelow(ctx, b.Round)
}

func (sc *Chain) processBlock(ctx context.Context, b *block.Block) {
	if err := sc.VerifyNotarization(ctx, b.Hash, b.VerificationTickets); err != nil {
		Logger.Error("notarization verification failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	if err := b.Validate(ctx); err != nil {
		Logger.Error("block validation", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return
	}
	er := sc.GetRound(b.Round)
	if er == nil {
		var r = round.NewRound(b.Round)
		er, _ = sc.AddRound(r).(*round.Round)
		sc.SetRandomSeed(er, b.RoundRandomSeed)
	}

	sc.AddNotarizedBlockToRound(er, b)
	sc.SetRoundRank(er, b)
	Logger.Info("received block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)))
	sc.AddNotarizedBlock(ctx, er, b)
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
		err := sc.StoreRound(ctx, r)
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

	var b *block.Block
	b = sc.requestForBlock(ctx, params, r)
	return b
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
func (sc *Chain) syncBlock(ctx context.Context, r *round.Round, canShard bool) *block.Block {
	scanMode := DeepScan
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scanMode)
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(r.Number, 10))
	params.Add("hash", r.BlockHash)
	var b *block.Block
	b = sc.requestForBlock(ctx, params, r)
	if b == nil {
		HCLogger.Info("HC-MissingObject",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("object", "Block"),
			zap.Int64("round", r.Number),
			zap.String("hash", r.BlockHash))
		return nil
	}

	if canShard {
		// Save the block
		err := sc.storeBlock(ctx, b)
		if err != nil {
			Logger.Error("HC-DSWriteFailure",
				zap.String("object", "block"),
				zap.Int64("cycle", cc.CycleCount),
				zap.Int64("round", r.Number),
				zap.Error(err))
		}
	}
	return b
}

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
	var rs *RoundSummaries
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		roundSummaries, ok := entity.(*RoundSummaries)
		if !ok {
			Logger.Error("received invalid round summaries")
			return nil, nil
		}
		rs = roundSummaries
		return rs, nil
	}
	sc.Sharders.RequestEntity(ctx, RoundSummariesRequestor, params, handler)
	return rs
}

func (sc *Chain) requestForRound(ctx context.Context, params *url.Values) *round.Round {
	var r *round.Round
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		roundEntity, ok := entity.(*round.Round)
		if !ok {
			Logger.Error("received invalid round entity")
			return nil, nil
		}
		if sc.isValidRound(roundEntity) {
			r = roundEntity
			return r, nil
		}
		return nil, nil
	}
	sc.Sharders.RequestEntity(ctx, RoundRequestor, params, handler)
	return r
}

func (sc *Chain) requestForBlockSummaries(ctx context.Context, params *url.Values) *BlockSummaries {
	var bs *BlockSummaries
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		blockSummaries, ok := entity.(*BlockSummaries)
		if !ok {
			Logger.Error("received invalid block summaries", zap.String("round", params.Get("round")), zap.String("range", params.Get("range")))
			return nil, nil
		}
		bs = blockSummaries
		return bs, nil
	}
	sc.Sharders.RequestEntity(ctx, BlockSummariesRequestor, params, handler)
	return bs
}

func (sc *Chain) requestForBlockSummary(ctx context.Context, params *url.Values) *block.BlockSummary {
	var blockS *block.BlockSummary
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		bs, ok := entity.(*block.BlockSummary)
		if !ok {
			Logger.Error("received invalid block summary entity", zap.String("hash", params.Get("hash")))
			return nil, nil
		}
		blockS = bs
		return blockS, nil
	}
	sc.Sharders.RequestEntity(ctx, BlockSummaryRequestor, params, handler)
	return blockS
}

func (sc *Chain) requestForBlock(ctx context.Context, params *url.Values, r *round.Round) *block.Block {
	self := node.GetSelfNode(ctx)
	_, nodes := sc.CanShardBlockWithReplicators(r.BlockHash, self.Node)

	if len(nodes) == 0 {
		Logger.Info("no replicators for this block (lost the block)", zap.Int64("round", r.Number))
	}

	var requestNode *node.Node
	for _, n := range nodes {
		if n == self.Node {
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
			if present == false {
				Logger.Debug("HC-StoreRoundSummaries",
					zap.String("object", "RoundSummary"),
					zap.Int64("round", roundS.Number),
					zap.String("hash", roundS.BlockHash))
				sc.StoreRound(ctx, roundS)
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
			if present == false {
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

func (sc *Chain) storeRoundSummary(ctx context.Context, r *round.Round) {
	var err error
	for true {
		err = sc.StoreRound(ctx, r)
		if err != nil {
			Logger.Error("db error (save round summary)", zap.Int64("round", r.Number), zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		break
	}
}

func (sc *Chain) storeBlockSummary(ctx context.Context, bs *block.BlockSummary) {
	var err error
	for true {
		err = sc.StoreBlockSummary(ctx, bs)
		if err == nil {
			return
		}
		Logger.Error("db error (save block summary)", zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.Error(err))
		time.Sleep(time.Second)
	}
}

func (sc *Chain) storeBlock(ctx context.Context, b *block.Block) error {
	var err error
	err = blockstore.GetStore().Write(b)
	if err == nil {
		sc.SharderStats.RepairBlocksCount++
	} else {
		sc.SharderStats.RepairBlocksFailure++
	}
	return err
	//	if err == nil {
	//		return
	//	}
	//	Logger.Error("db error (save block)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	//	time.Sleep(time.Second)
	//}
	//return err
}

func (sc *Chain) storeBlockTransactions(ctx context.Context, b *block.Block) error {
	err := sc.StoreTransactions(ctx, b)
	//	Logger.Error(caller,
	//		zap.Int64("round", b.Round),
	//		zap.String("block", b.Hash),
	//		zap.Error(err))
	//}
	return err
}
// NotarizedBlockFetched -
func (sc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {

}
