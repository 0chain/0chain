package chain

import (
	"context"
	"time"

	"0chain.net/chaincore/node"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
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
		if fb.Round < c.LatestFinalizedBlock.Round-5 {
			Logger.Error("slow finalized block processing", zap.Int64("lfb", c.LatestFinalizedBlock.Round), zap.Int64("fb", fb.Round))
		}
		c.finalizeBlock(ctx, fb, bsh)
	}
}

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	tick := time.Duration(c.PruneStateBelowCount) * time.Second
	timer := time.NewTimer(time.Second)
	pruning := false
	for true {
		select {
		case <-timer.C:
			if pruning {
				Logger.Info("pruning still going on")
				continue
			}
			pruning = true
			c.pruneClientState(ctx)
			pruning = false
			if c.pruneStats == nil || c.pruneStats.MissingNodes > 0 {
				timer = time.NewTimer(time.Second)
			} else {
				timer = time.NewTimer(tick)
			}
		}
	}
}

/*BlockFetchWorker - a worker that fetches the prior missing blocks */
func (c *Chain) BlockFetchWorker(ctx context.Context) {
	Logger.Info("BlockFetchWorker started")
	for true {
		select {
		case b := <-c.blockFetcher.missingLinkBlocks:
			if b.PrevBlock != nil {
				Logger.Info("missingLinkBlocks b.PreveBlock != nil. continue...", zap.String("b_Hash", b.Hash))
				continue
			}
			pb, err := c.GetBlock(ctx, b.PrevHash)
			if err == nil {
				Logger.Info("Got PrevHash missingLinkBlocks", zap.String("b_PrevHash", b.PrevHash))
				b.SetPreviousBlock(pb)
				continue
			}
			Logger.Info("missingLinkBlocks --FetchPreviousBlock", zap.String("b_PrevHash", b.Hash))

			c.blockFetcher.FetchPreviousBlock(ctx, c, b)
		case bHash := <-c.blockFetcher.missingBlocks:
			Logger.Info("missingBlocks --GetBlock", zap.String("bHash", bHash))
			_, err := c.GetBlock(ctx, bHash)
			if err == nil {
				Logger.Info("missingBlocks --Already has it. Not fetching it", zap.String("bHash", bHash))

				continue
			}
			Logger.Info("missingBlocks --do not have it. Fetching...", zap.String("bHash", bHash))

			c.blockFetcher.FetchBlock(ctx, c, bHash)
		}
	}
}
