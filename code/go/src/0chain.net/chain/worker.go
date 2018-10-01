package chain

import (
	"context"
	"time"

	. "0chain.net/logging"
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
	for r := range c.finalizedRoundsChannel {
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

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.PruneStateBelowCount) * time.Second)
	pruning := false
	for true {
		select {
		case <-ticker.C:
			if pruning {
				Logger.Info("pruning still going on")
				continue
			}
			pruning = true
			c.pruneClientState(ctx)
			pruning = false

		}
	}
}
