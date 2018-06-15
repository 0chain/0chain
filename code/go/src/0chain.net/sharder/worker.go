package sharder

import (
	"context"
	"log"
	"os"
	"sort"
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
	Logger.Info("clearing worker state")
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
					Logger.Info("block already received", zap.Any("round", b.Round), zap.Any("block", b.Hash))
					continue
				} else {
					Logger.Error("get block", zap.Any("block", b.Hash), zap.Error(err))
				}
			}
			sc.AddBlock(b)
			er, ok := rounds[b.Round]
			if ok {
				nb := er.GetNotarizedBlocks()
				if len(nb) > 0 {
					Logger.Error("*** different blocks for the same round ***", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("existing_block", nb[0].Hash))
				}
			} else {
				er = datastore.GetEntityMetadata("round").Instance().(*round.Round)
				er.Number = b.Round
				er.RandomSeed = b.RoundRandomSeed
				rounds[er.Number] = er
			}
			if b.Round != 1 {
				timer.UpdateSince(ts)
			}
			ts = time.Now()
			er.AddNotarizedBlock(b)
			// Sort transactions by their hash - useful for quick search
			sort.SliceStable(b.Txns, func(i, j int) bool { return b.Txns[i].Hash < b.Txns[j].Hash })
			StoreBlock(ctx, b)
		}
	}
}
