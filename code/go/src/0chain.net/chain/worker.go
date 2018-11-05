package chain

import (
	"context"
	"time"

	"0chain.net/cache"
	. "0chain.net/logging"
	"0chain.net/node"
)

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.Miners.StatusMonitor(ctx)
	go c.Sharders.StatusMonitor(ctx)
	go c.Blobbers.StatusMonitor(ctx)
	go c.PruneClientStateWorker(ctx)
	go c.BlockFetchWorker(ctx)
	go node.Self.MemoryUsage()
}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) FinalizeRoundWorker(ctx context.Context, bsh BlockStateHandler) {
	for r := range c.finalizedRoundsChannel {
		c.finalizeRound(ctx, r, bsh)
		c.UpdateRoundInfo(r)
	}
}

//FinalizedBlockWorker - a worker that processes finalized blocks
func (c *Chain) FinalizedBlockWorker(ctx context.Context, bsh BlockStateHandler) {
	for fb := range c.finalizedBlocksChannel {
		c.finalizeBlock(ctx, fb, bsh)
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

/*BlockFetchWorker - a worker that fetches the prior missing blocks */
func (c *Chain) BlockFetchWorker(ctx context.Context) {
	fblocks := cache.NewLRUCache(100)
	for b := range c.missingLinkBlocks {
		if b.PrevBlock != nil {
			continue
		}
		pb, err := c.GetBlock(ctx, b.PrevHash)
		if err == nil {
			b.SetPreviousBlock(pb)
			continue
		}
		if _, err := fblocks.Get(b.PrevHash); err != nil {
			fblocks.Add(b.PrevHash, true)
			go c.GetPreviousBlock(ctx, b)
		}
	}
}
