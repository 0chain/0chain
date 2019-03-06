package sharder

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"
	"net/url"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/sharder/blockstore"
	"0chain.net/chaincore/config"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var blockSaveTimer metrics.Timer

func init() {
	blockSaveTimer = metrics.GetOrRegisterTimer("block_save_time", nil)
}

/*UpdatePendingBlock - update the pending block */
func (sc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {

}

/*UpdateFinalizedBlock - updates the finalized block */
func (sc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	fr := sc.GetRound(b.Round)
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("lf_round", sc.LatestFinalizedBlock.Round), zap.Any("current_round", sc.CurrentRound))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	sc.BlockCache.Add(b.Hash, b)
	sc.StoreTransactions(ctx, b)
	err := sc.StoreBlockSummary(ctx, b)
	if err != nil {
		Logger.Error("db error (save block)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	}
	self := node.GetSelfNode(ctx)
	if sc.IsBlockSharder(b, self.Node) {
		sc.storeBlock(b)
	}
	if fr != nil {
		sc.storeRound(ctx, fr, b)
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
	if sc.AddRoundBlock(er, b) != b {
		return
	}
	sc.SetRoundRank(er, b)
	Logger.Info("received block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)))
	sc.AddNotarizedBlock(ctx, er, b)
}

/*GetLatestRoundFromSharders - gives the latest of rounds that the sharders are processing*/
func (sc *Chain) GetLatestRoundFromSharders(ctx context.Context, currRound int64) *round.Round {
	latestRounds := make([]*round.Round, 0, 1)

	latestRoundHandler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		r, ok := entity.(*round.Round)
		if !ok {
			Logger.Error("latest round received is not valid")
			return nil, nil
		}
		latestRounds = append(latestRounds, r)
		return r, nil
	}

	sc.Sharders.RequestEntityFromAll(ctx, LatestRoundRequestor, nil, latestRoundHandler)

	if len(latestRounds) > 0 {
		sort.Slice(latestRounds, func(i int, j int) bool { return latestRounds[i].Number >= latestRounds[j].Number })
		Logger.Info("bc-27 the latest round", zap.Int64("round", latestRounds[0].Number))
		return latestRounds[0]
	}

	Logger.Info("bc-27 no latest rounds received from sharders")
	return nil
}

/*GetMissingRounds - request for missed rounds and stores them*/
func (sc *Chain) GetMissingRounds(ctx context.Context, targetR int64, dbR int64) {
	Logger.Info("bc-27 get missing rounds", zap.Int64("target round", targetR), zap.Int64("round", dbR))
	
	//get missing rounds starting from the next round of the current round
	syncRound := dbR + 1
	for syncRound < sc.BSync.GetFinalizationRound() {
		sc.BSync.SetSyncingRound(syncRound)
		Logger.Info("bc-27 sync info", zap.Int64("sync round", syncRound), zap.Int64("accept round", sc.BSync.GetAcceptanceRound()), zap.Int64("latest round", sc.BSync.GetFinalizationRound()))
		params := &url.Values{}
		params.Add("round", strconv.FormatInt(syncRound, 10))
		
		var r *round.Round
		sc.Sharders.RequestEntityFromAll(ctx, RoundRequestor, params, func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
			roundEntity, ok := entity.(*round.Round)
			if !ok {
				Logger.Info("bc-27 invalid round entity received", zap.Int64("round", syncRound))
				return nil, nil
			}
			r = roundEntity
			return r, nil
		})

		if r == nil {
			Logger.Info("bc-27 requested round is nil", zap.Int64("round", syncRound))
			syncRound++
			sc.BSync.SetSyncingRound(syncRound)
			continue
		}

		if r.BlockHash == "" {
			Logger.Info("bc-27 requested round block hash is empty", zap.Int64("round", r.Number))
			syncRound++
			sc.BSync.SetSyncingRound(syncRound)
			continue
		}

		sc.storeMissingRoundBlock(ctx, r)
		acceptRound := sc.BSync.GetAcceptanceRound()
		if acceptRound != 0 && syncRound >= acceptRound - 1 {
			break
		}
		syncRound++
	}
}

func (sc *Chain) storeMissingRoundBlock(ctx context.Context, r *round.Round) {
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(r.Number, 10))
	params.Add("block", r.BlockHash)

	self := node.GetSelfNode(ctx)
	canStore, nodes := sc.IsBlockSharderWithNodes(r.BlockHash, self.Node)
	Logger.Info("bc-27 number of request nodes", zap.Int("nodes", len(nodes)))
	var requestNode *node.Node
	for _, n := range nodes {
		if n != self.Node {
			requestNode = n
			Logger.Info("bc-27 request round block info", zap.Int("node", requestNode.SetIndex), zap.Int64("round", r.Number))
			break
		}
	}
	if requestNode == nil {
		Logger.Info("bc-27 request node for round block not found (nil)", zap.Int64("round", r.Number))
		return
	}

	var b *block.Block
	requestNode.RequestEntityFromNode(ctx, BlockRequestor, params, func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		blockEntity, ok := entity.(*block.Block)
		if !ok {
			Logger.Error("bc-27 requested block is invalid", zap.Int64("round", r.Number))
			return nil, nil
		}
		err := blockEntity.Validate(ctx)
		if err == nil {
			b = blockEntity
			return blockEntity, nil
		}
		return nil, err
	})
	if b == nil {
		Logger.Info("bc-27 requested round block is nil", zap.Int64("round", r.Number), zap.String("block-hash", r.BlockHash))
		return
	}

	sc.StoreTransactions(ctx, b)
	err := sc.StoreBlockSummary(ctx, b)
	if err != nil {
		Logger.Error("bc-27 db error (save block)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	}
	if canStore {
		sc.storeBlock(b)
		Logger.Info("bc-27 block stored in file", zap.Int64("round", b.Round), zap.String("block-hash", b.Hash))
	}
	sc.storeRound(ctx, r, b)
	sc.LatestFinalizedBlock = b
	Logger.Info("bc-27 finalize round - latest finalized round", zap.Int64("round", sc.LatestFinalizedBlock.Round), zap.String("block", b.Hash))
}

func (sc *Chain) storeRound(ctx context.Context, r round.RoundI, b *block.Block) {
	r.Finalize(b)
	rImpl, _ := r.(*round.Round)
	err := sc.StoreRound(ctx, rImpl)
	if err != nil {
		Logger.Error("db error (save round)", zap.Int64("round", r.GetRoundNumber()), zap.Error(err))
	}
}

func (sc *Chain) storeBlock(b *block.Block) {
	sc.SharderStats.ShardedBlocksCount++
	ts := time.Now()
	err := blockstore.GetStore().Write(b)
	duration := time.Since(ts)
	blockSaveTimer.UpdateSince(ts)
	p95 := blockSaveTimer.Percentile(.95)
	if blockSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		Logger.Error("block save - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	}
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	}
}

func (sc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {

}
