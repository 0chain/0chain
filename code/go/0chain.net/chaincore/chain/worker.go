package chain

/*
LOOKS GOOD. NEEDS MORE TESTING BEFORE COMMITED!!!!
*/

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var UpdateNodes chan bool

func init() {
	UpdateNodes = make(chan bool, 10)
}

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.StatusMonitor(ctx)
	go c.PruneClientStateWorker(ctx)
	go c.BlockFetchWorker(ctx)
	go node.Self.MemoryUsage()
}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) StatusMonitor(ctx context.Context) {
	smctx, cancel := context.WithCancel(ctx)
	go c.Miners.StatusMonitor(smctx)
	go c.Sharders.StatusMonitor(smctx)
	for true {
		select {
		case <-ctx.Done():
			return
		case <-UpdateNodes:
			cancel()
			Logger.Info("the status monitor is dead, long live the status monitor", zap.Any("miners", c.Miners), zap.Any("sharders", c.Sharders))
			smctx, cancel = context.WithCancel(ctx)
			go c.Miners.StatusMonitor(smctx)
			go c.Sharders.StatusMonitor(smctx)
		}
	}
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
	lfb := c.GetLatestFinalizedBlock()
	for fb := range c.finalizedBlocksChannel {
		if fb.Round < lfb.Round-5 {
			Logger.Error("slow finalized block processing", zap.Int64("lfb", lfb.Round), zap.Int64("fb", fb.Round))
		}
		c.finalizeBlock(ctx, fb, bsh)
	}
}

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	tick := time.Duration(c.PruneStateBelowCount) * time.Second
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-timer.C:
			c.pruneClientState(ctx)
			if c.pruneStats == nil || c.pruneStats.MissingNodes > 0 {
				timer.Stop()
				timer = time.NewTimer(time.Second)
			} else {
				timer.Stop()
				timer = time.NewTimer(tick)
			}
		}
	}
}

/*BlockFetchWorker - a worker that fetches the prior missing blocks */
func (c *Chain) BlockFetchWorker(ctx context.Context) {
	for {
		select {
		case b := <-c.blockFetcher.missingLinkBlocks:
			if b.PrevBlock != nil {
				continue
			}
			pb, err := c.GetBlock(ctx, b.PrevHash)
			if err == nil {
				b.SetPreviousBlock(pb)
				continue
			}

			c.blockFetcher.FetchPreviousBlock(ctx, c, b)
		case bHash := <-c.blockFetcher.missingBlocks:
			_, err := c.GetBlock(ctx, bHash)
			if err == nil {
				continue
			}
			c.blockFetcher.FetchBlock(ctx, c, bHash)
		}
	}
}

func (c *Chain) VerifyChainHistory(ctx context.Context, latestMagicBlock *block.Block) error {
	currentMagicBlock := c.GetLatestFinalizedMagicBlock()
	var sharders []string
	c.Sharders.ForEach(func(sharder *node.Node) {
		sharders = append(sharders, "http://"+sharder.N2NHost+":"+strconv.Itoa(sharder.Port))
	})
	for currentMagicBlock.Hash != latestMagicBlock.Hash {
		magicBlock, err := httpclientutil.GetMagicBlockCall(sharders, currentMagicBlock.MagicBlockNumber+1, 1)
		if err != nil {
			Logger.DPanic(fmt.Sprintf("failed to get magic block(%v): %v", currentMagicBlock.MagicBlockNumber+1, err.Error()))
		}
		if !magicBlock.VerifyMinersSignatures(currentMagicBlock) {
			Logger.DPanic(fmt.Sprintf("failed to verify magic block: %v", err.Error()))
		}
		Logger.Info("verify chain history", zap.Any("magicBlock_block", magicBlock))
		err = c.UpdateMagicBlock(magicBlock.MagicBlock)
		if err != nil {
			Logger.DPanic(fmt.Sprintf("failed to update magic block: %v", err.Error()))
		}
		c.SetLatestFinalizedMagicBlock(magicBlock)
		currentMagicBlock = c.GetLatestFinalizedMagicBlock()
	}
	return nil
}
