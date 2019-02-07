package sharder

import (
	"context"
	"math"
	"time"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/sharder/blockstore"
	"0chain.net/core/config"

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
	if fr != nil {
		fr.Finalize(b)
		frImpl, _ := fr.(*round.Round)
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
	if sc.AddRoundBlock(er, b) != b {
		return
	}
	sc.SetRoundRank(er, b)
	Logger.Info("received block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)))
	sc.AddNotarizedBlock(ctx, er, b)
}

func (sc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {

}
