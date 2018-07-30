package chain

import (
	"context"
	"time"

	"0chain.net/block"
	. "0chain.net/logging"
	"0chain.net/round"
	"0chain.net/util"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

var DELTA = 200 * time.Millisecond
var FINALIZATION_TIME = 2 * DELTA

/*SetNetworkRelayTime - setup the network relay time */
func SetNetworkRelayTime(delta time.Duration) {
	DELTA = delta
	FINALIZATION_TIME = 2 * delta
}

var FinalizationTimer metrics.Timer
var fts time.Time

func init() {
	if FinalizationTimer != nil {
		metrics.Unregister("finalization_time")
	}
	FinalizationTimer = metrics.GetOrRegisterTimer("finalization_time", nil)
}

/*FinalizeRound - starting from the given round work backwards and identify the round that can be
  assumed to be finalized as only one chain has survived.
  Note: It is that round and prior that actually get finalized.
*/
func (c *Chain) FinalizeRound(ctx context.Context, r *round.Round, bsh BlockStateHandler) {
	if !r.SetFinalizing() {
		return
	}
	time.Sleep(FINALIZATION_TIME)
	c.FinalizedRoundsChannel <- r
}

func (c *Chain) finalizeRound(ctx context.Context, r *round.Round, bsh BlockStateHandler) {
	lfb := c.ComputeFinalizedBlock(ctx, r)
	if lfb == nil {
		Logger.Debug("finalization - no decisive block to finalize yet or don't have all the necessary blocks", zap.Any("round", r.Number))
		return
	}
	if lfb.Hash == c.LatestFinalizedBlock.Hash {
		return
	}
	if lfb.Round < c.LatestFinalizedBlock.Round {
		Logger.Info("finalize round - TODO: need to repair", zap.Any("lf_round", c.LatestFinalizedBlock.Round), zap.Int64("new_lf_round", lfb.Round))
		return
	}
	lfbHash := c.LatestFinalizedBlock.Hash
	c.LatestFinalizedBlock = lfb
	frchain := make([]*block.Block, 0, 1)
	for b := lfb; b != nil && b.Hash != lfbHash; b = b.PrevBlock {
		frchain = append(frchain, b)
	}
	if len(frchain) == 0 {
		return
	}
	deadBlocks := make([]*block.Block, 0, 1)
	for idx := range frchain {
		fb := frchain[len(frchain)-1-idx]
		Logger.Info("finalize round", zap.Int64("round", r.Number), zap.Int64("finalized_round", fb.Round), zap.String("hash", fb.Hash))
		if time.Since(fts) < 10*time.Second {
			FinalizationTimer.UpdateSince(fts)
		}
		fts = time.Now()
		UpdateInfo(fb)
		if fb.ClientState != nil {
			fb.ClientState.SaveChanges(c.StateDB, util.Origin(fb.Round), false)
			Logger.Info("finalize round - save state", zap.Int64("round", fb.Round), zap.String("block", fb.Hash), zap.String("hash", util.ToHex(fb.ClientState.GetRoot())), zap.Int("changes", len(fb.ClientState.GetChangeCollector().GetChanges())))
		}
		bsh.UpdateFinalizedBlock(ctx, fb)
		frb := c.GetRoundBlocks(fb.Round)
		for _, b := range frb {
			if b.Hash != fb.Hash {
				deadBlocks = append(deadBlocks, b)
			}
		}
	}
	if lfb.ClientState != nil {
		ndb := lfb.ClientState.GetNodeDB()
		lfb.ClientState.SetNodeDB(c.StateDB)
		if lndb, ok := ndb.(*util.LevelNodeDB); ok {
			lndb.C = c.StateDB
			lndb.P = util.NewMemoryNodeDB() // break the chain to reclaim memory
		}
	}
	// Prune all the dead blocks
	c.DeleteBlocks(deadBlocks)
	// Prune the chain from the oldest finalized block
	c.PruneChain(ctx, frchain[len(frchain)-1])
}

/*PruneChain - prunes the chain */
func (c *Chain) PruneChain(ctx context.Context, b *block.Block) {
	c.DeleteBlocksBelowRound(b.Round - 50)
}
