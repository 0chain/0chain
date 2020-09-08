package chain

/*
LOOKS GOOD. NEEDS MORE TESTING BEFORE COMMITED!!!!
*/

import (
	"0chain.net/chaincore/round"
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var UpdateNodes chan int64

func init() {
	UpdateNodes = make(chan int64, 10)
}

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.StatusMonitor(ctx)
	go c.PruneClientStateWorker(ctx)
	go c.BlockFetchWorker(ctx)
	go c.StartLFBTicketWorker(ctx, c.GetLatestFinalizedBlock())
	go c.StartFinalizedBlockFetcherWorker(ctx)
	go node.Self.Underlying().MemoryUsage()
}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) StatusMonitor(ctx context.Context) {
	smctx, cancel := context.WithCancel(ctx)
	mb := c.GetCurrentMagicBlock()
	go mb.Miners.StatusMonitor(smctx)
	go mb.Sharders.StatusMonitor(smctx)
	for true {
		select {
		case <-ctx.Done():
			return
		case nRound := <-UpdateNodes:
			cancel()
			Logger.Info("the status monitor is dead, long live the status monitor", zap.Any("miners", mb.Miners), zap.Any("sharders", mb.Sharders))
			smctx, cancel = context.WithCancel(ctx)
			mb := c.GetMagicBlock(nRound)
			go mb.Miners.StatusMonitor(smctx)
			go mb.Sharders.StatusMonitor(smctx)
		}
	}
}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) FinalizeRoundWorker(ctx context.Context, bsh BlockStateHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-c.finalizedRoundsChannel:
			c.finalizeRound(ctx, r, bsh)
			c.UpdateRoundInfo(r)
		}
	}
}

func isDone(ctx context.Context) (done bool) {
	select {
	case <-ctx.Done():
		return true
	default:
		return
	}
}

func sleepOrCancel(ctx context.Context) bool {
	var tm = time.NewTimer(1 * time.Second)
	defer tm.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-tm.C:
		return false
	}
}

func (c *Chain) repairChain(ctx context.Context, newMB *block.Block,
	saveFunc MagicBlockSaveFunc) (err error) {

	var latest = c.GetLatestFinalizedMagicBlock()

	if newMB.MagicBlockNumber <= latest.MagicBlockNumber {
		return common.NewError("repair_mb_chain", "already have such MB")
	}

	if newMB.MagicBlockNumber == latest.MagicBlockNumber+1 {
		if newMB.PreviousMagicBlockHash != latest.MagicBlock.Hash {
			return common.NewError("repair_mb_chain", "invalid prev-MB ref.")
		}
		return // it's just next MB
	}

	// here the newBM is not next but newer

	Logger.Info("repair_mb_chain: repair from-to mb_number",
		zap.Int64("from", latest.MagicBlockNumber),
		zap.Int64("to", newMB.MagicBlockNumber))

	// until the end of the days
	if err = c.VerifyChainHistory(ctx, newMB, saveFunc); err != nil {
		Logger.Error("repair_mb_chain", zap.Error(err))
		return common.NewErrorf("repair_mb_chain", err.Error())
	}

	// the VerifyChainHistory doesn't save the newMB
	// finalizeRound will do it next step

	return // ok
}

//FinalizedBlockWorker - a worker that processes finalized blocks
func (c *Chain) FinalizedBlockWorker(ctx context.Context, bsh BlockStateHandler) {
	for {
		select {
		case <-ctx.Done():
			return

		case fb := <-c.finalizedBlocksChannel:
			lfb := c.GetLatestFinalizedBlock()
			if fb.Round < lfb.Round-5 {
				Logger.Error("slow finalized block processing",
					zap.Int64("lfb", lfb.Round), zap.Int64("fb", fb.Round))
			}

			// TODO/TOTHINK: move the repair chain outside the finalized worker?

			// make sure we have valid verified MB chain if the block contains
			// a magic block; we already have verified and valid MB chain at this
			// moment, let's keep it updated and verified too

			if fb.MagicBlock != nil && node.Self.Type == node.NodeTypeSharder {
				var err = c.repairChain(ctx, fb, bsh.SaveMagicBlock())
				if err != nil {
					Logger.Error("repairing MB chain", zap.Error(err))
					return
				}
			}

			// finalize

			c.finalizeBlock(ctx, fb, bsh)
		}
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
	for true {
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
		case hr := <-c.blockFetcher.missingBlocks:
			_, err := c.GetBlock(ctx, hr.hash)
			if err == nil {
				continue
			}
			c.blockFetcher.FetchBlock(ctx, c, hr.hash, hr.round)
		}
	}
}

type MagicBlockSaveFunc func(context.Context, *block.Block) error

// VerifyChainHistory repairs and verifies magic blocks chain.
func (c *Chain) VerifyChainHistory(ctx context.Context,
	latestMagicBlock *block.Block, saveHandler MagicBlockSaveFunc) (err error) {

	var (
		cmb = c.GetCurrentMagicBlock()

		currentMagicBlock = c.GetLatestFinalizedMagicBlock()
		sharders          = cmb.Sharders.N2NURLs()
		magicBlock        *block.Block
	)

	// until we have got all MB from our from store to latest given
	for currentMagicBlock.Hash != latestMagicBlock.Hash {
		Logger.Debug("verify_chain_history",
			zap.Int64("get_mb_number", currentMagicBlock.MagicBlockNumber+1))

		magicBlock, err = httpclientutil.GetMagicBlockCall(sharders,
			currentMagicBlock.MagicBlockNumber+1, 1)

		if err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to get %d: %v",
					currentMagicBlock.MagicBlockNumber+1, err))
		}

		if !magicBlock.VerifyMinersSignatures(currentMagicBlock) {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to verify magic block %d: %v",
					currentMagicBlock.MagicBlockNumber+1, err))
		}

		Logger.Info("verify chain history",
			zap.Any("mb_sr", magicBlock.StartingRound),
			zap.Any("mb_hash", magicBlock.Hash))

		if err = c.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to update magic block %d: %v",
					currentMagicBlock.MagicBlockNumber+1, err))
		} else {
			c.UpdateNodesFromMagicBlock(magicBlock.MagicBlock)
		}

		c.SetLatestFinalizedMagicBlock(magicBlock)
		currentMagicBlock = magicBlock

		if saveHandler != nil {
			if err = saveHandler(ctx, magicBlock); err != nil {
				return common.NewError("get_lfmb_from_sharders",
					fmt.Sprintf("failed to save updated magic block %d: %v",
						currentMagicBlock.MagicBlockNumber, err))
			}
		}

	}

	return
}

// // MustVerifyChainHistory panics on error.
// func (c *Chain) MustVerifyChainHistory(ctx context.Context,
// 	latestMagicBlock *block.Block, saveHandler MagicBlockSaveFunc) error {
// 	err := c.VerifyChainHistory(ctx, latestMagicBlock, saveHandler)
// 	if err != nil {
// 		return common.NewErrorf("verify_chain_history", err.Error())
// 	}
// 	return nil
// }

// PruneStorageWorker pruning storage
func (c *Chain) PruneStorageWorker(ctx context.Context, d time.Duration,
	getCountRoundStorage func(storage round.RoundStorage) int,
	storage ...round.RoundStorage) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.PruneRoundStorage(ctx, getCountRoundStorage, storage...)
		}
	}
}
