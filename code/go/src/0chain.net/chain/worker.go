package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/block"
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

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	ticker := time.NewTicker(PruneBelowCount * time.Second)
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

/*PruneBelowCount - prune nodes below these many rounds */
const PruneBelowCount = 1000

func (c *Chain) pruneClientState(ctx context.Context) {
	bc := c.BlockChain
	bc = bc.Move(-PruneBelowCount)
	for i := 0; i < 10 && bc.Value == nil; i++ {
		bc = bc.Prev()
	}
	if bc.Value == nil {
		return
	}
	bs := bc.Value.(*block.BlockSummary)
	mpt := util.NewMerklePatriciaTrie(c.StateDB)
	mpt.SetRoot(bs.ClientStateHash)
	newOrigin := util.Origin(bs.Round)
	t := time.Now()
	pctx := util.WithPruneStats(ctx)
	err := mpt.UpdateOrigin(pctx, newOrigin)
	d1 := time.Since(t)
	Logger.Info("prune client state - new origin", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)))
	if config.DevConfiguration.State {
		fmt.Fprintf(stateOut, "update to new origin: %v %v %v\n", util.ToHex(mpt.GetRoot()), bs.Round, newOrigin)
		mpt.PrettyPrint(stateOut)
	}
	t1 := time.Now()
	if err != nil {
		Logger.Error("prune client state (update origin)", zap.Error(err))
	}
	err = c.StateDB.PruneBelowOrigin(pctx, newOrigin)
	if err != nil {
		Logger.Error("prune client state error", zap.Error(err))
	}
	ps := util.GetPruneStats(pctx)
	Logger.Info("prune client state stats", zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)),
		zap.Duration("duration", time.Since(t)), zap.Duration("update", d1), zap.Duration("prune", time.Since(t1)), zap.Any("stats", ps))
}
