package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/config"
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
			lfb := c.LatestFinalizedBlock
			for ; lfb != nil && !lfb.IsStateComputed(); lfb = lfb.PrevBlock {
			}
			if lfb == nil {
				continue
			}
			if lfb.Round < PruneBelowCount {
				Logger.Info("prune client state (not enough rounds)", zap.Int64("round", c.CurrentRound))
				continue
			}
			pruning = true
			mpt := lfb.ClientState // TODO: We actually need the root hash at the newOrigin, we shouldn't be pruning w.r.t nodes reachable from current
			no := lfb.Round - PruneBelowCount
			no -= no % 100
			newOrigin := util.Origin(no)
			pctx := util.WithPruneStats(ctx)
			err := mpt.UpdateOrigin(pctx, newOrigin)
			d1 := time.Since(t)
			if config.DevConfiguration.State {
				fmt.Fprintf(stateOut, "update to new origin: %v %v %v %v\n", util.ToHex(mpt.GetRoot()), lfb.Round, lfb.IsStateComputed(), newOrigin)
				mpt.PrettyPrint(stateOut)
			}
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
