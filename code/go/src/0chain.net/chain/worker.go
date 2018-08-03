package chain

import (
	"context"
	"time"

	. "0chain.net/logging"
	"0chain.net/util"
	"go.uber.org/zap"
)

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.Miners.StatusMonitor(ctx)
	go c.Sharders.StatusMonitor(ctx)
	go c.Blobbers.StatusMonitor(ctx)
	go c.PruneClientStateWorker(ctx)
}

/*BlockFinalizationWorker - a worker that handles the finalized blocks */
func (c *Chain) BlockFinalizationWorker(ctx context.Context, bsh BlockStateHandler) {
	for r := range c.FinalizedRoundsChannel {
		nbCount := len(r.GetNotarizedBlocks())
		if nbCount == 0 {
			c.ZeroNotarizedBlocksCount++
		}
		if nbCount > 1 {
			c.MultiNotarizedBlocksCount++
		}
		c.finalizeRound(ctx, r, bsh)
		c.UpdateRoundInfo(r)
	}
}

/*PruneBelowCount - prune nodes below these many rounds */
const PruneBelowCount = 100

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	ticker := time.NewTicker(PruneBelowCount * time.Second)
	pruning := false
	for true {
		select {
		case t := <-ticker.C:
			if pruning {
				Logger.Info("pruning still going on")
				continue
			}
			if c.CurrentRound < PruneBelowCount {
				Logger.Info("prune client state (not enough rounds)", zap.Int64("round", c.CurrentRound))
				continue
			}
			pruning = true
			mpt := util.NewMerklePatriciaTrie(c.StateDB)
			newOrigin := util.Origin(c.CurrentRound - PruneBelowCount)
			pctx := util.WithPruneStats(ctx)
			err := mpt.UpdateOrigin(pctx, newOrigin)
			d1 := time.Since(t)
			t1 := time.Now()
			if err != nil {
				Logger.Info("prune client state (update origin)", zap.Error(err))
			}
			err = mpt.PruneBelowOrigin(pctx, newOrigin)
			if err != nil {
				Logger.Error("prune client state error", zap.Error(err))
			}
			ps := util.GetPruneStats(pctx)
			Logger.Info("client state prune time", zap.Duration("duration", time.Since(t)), zap.Duration("update", d1), zap.Duration("prune", time.Since(t1)), zap.Any("stats", ps))
			pruning = false
		}
	}
}
