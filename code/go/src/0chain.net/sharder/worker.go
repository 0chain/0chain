package sharder

import (
	"context"
	"log"
	"os"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/round"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers() {
	ClearWorkerState()
	ctx := common.GetRootContext()
	go GetSharderChain().BlockWorker(ctx)
	go metrics.LogScaled(metrics.DefaultRegistry, 60*time.Second, time.Millisecond, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
}

var timer metrics.Timer

//TODO: The blocks and rounds data structures are temporary for debugging.
var rounds map[int64]*round.Round

/*ClearWorkerState - clears the worker state */
func ClearWorkerState() {
	Logger.Debug("clearing worker state")
	rounds = make(map[int64]*round.Round)
	if timer != nil {
		metrics.Unregister("block_time")
	}
	timer = metrics.GetOrRegisterTimer("block_time", nil)
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {
	var ts time.Time
	for true {
		select {
		case <-ctx.Done():
			return
		case b := <-sc.GetBlockChannel():
			eb, err := sc.GetBlock(ctx, b.Hash)
			if eb != nil {
				if err == nil {
					Logger.Debug("block already received", zap.Any("round", b.Round), zap.Any("block", b.Hash))
					continue
				} else {
					Logger.Error("get block", zap.Any("block", b.Hash), zap.Error(err))
				}
			}
			if err := sc.VerifyNotarization(ctx, b, b.VerificationTickets); err != nil {
				Logger.Error("notarization verification failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
				continue
			}
			if err := b.Validate(ctx); err != nil {
				Logger.Error("block validation", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
				continue
			}
			sc.AddBlock(b)
			er := sc.GetRound(b.Round)
			if er != nil {
				nb := er.GetNotarizedBlocks()
				if len(nb) > 0 {
					Logger.Error("*** different blocks for the same round ***", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("existing_block", nb[0].Hash))
				}
			} else {
				er = datastore.GetEntityMetadata("round").Instance().(*round.Round)
				er.Number = b.Round
				er.RandomSeed = b.RoundRandomSeed
				sc.AddRound(er)
			}
			if time.Since(ts) < 10*time.Second {
				timer.UpdateSince(ts)
			}
			ts = time.Now()
			er.AddNotarizedBlock(b)
			if b.Round > 1 {
				sc.FinalizeRound(ctx, er, sc)
			}
		}
	}
}
