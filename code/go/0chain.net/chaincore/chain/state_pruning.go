package chain

import (
	"context"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//StatePruneUpdateTimer - a metric that tracks the time it takes to update older nodes still referrred from the given version
var StatePruneUpdateTimer metrics.Timer

//StatePruneDeleteTimer - a metric that tracks the time it takes to delete all the obsolete nodes w.r.t a given version
var StatePruneDeleteTimer metrics.Timer

func init() {
	StatePruneUpdateTimer = metrics.GetOrRegisterTimer("state_prune_update_timer", nil)
	StatePruneDeleteTimer = metrics.GetOrRegisterTimer("state_prune_delete_timer", nil)
}

func (c *Chain) pruneClientState(ctx context.Context) {
	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil {
		return
	}

	if lfb.Round <= int64(c.PruneStateBelowCount()) {
		return
	}

	var bc = c.BlockChain
	bc = bc.Move(-c.PruneStateBelowCount())

	for i := 0; i < 10 && bc.Value == nil; i++ {
		bc = bc.Prev()
	}

	var bs *block.BlockSummary

	if bc.Value != nil {
		bs = bc.Value.(*block.BlockSummary)
		for bs.Round%100 != 0 {
			bc = bc.Prev()
			if bc.Value == nil {
				break
			}
			bs = bc.Value.(*block.BlockSummary)
		}
	} else {
		if lfb.Round == 0 {
			logging.Logger.Debug("Last finalized block round is 0")
			return
		}
	}

	if bs == nil {
		bs = &block.BlockSummary{
			Round:           lfb.Round,
			ClientStateHash: lfb.ClientStateHash,
		}
	}

	cr := c.GetCurrentRound()
	logging.Logger.Info("prune client state",
		zap.Int64("current_round", cr),
		zap.Int64("latest_finalized_round", lfb.Round),
		zap.Int64("round", bs.Round),
		zap.String("block", bs.Hash),
		zap.String("state_hash", util.ToHex(bs.ClientStateHash)))

	var (
		newVersion = util.Sequence(bs.Round)
		pctx       = util.WithPruneStats(ctx)
		ps         = util.GetPruneStats(pctx)
	)

	ps.Stage = util.PruneStateDelete
	c.pruneStats = ps

	if lfb.Round-int64(c.PruneStateBelowCount()) < bs.Round {
		ps.Stage = util.PruneStateAbandoned
		return
	}

	var t = time.Now()
	err := c.stateDB.(*util.PNodeDB).PruneBelowVersionV(pctx, newVersion, cr)
	if err != nil {
		logging.Logger.Error("prune client state error", zap.Error(err))
	}
	ps.Stage = util.PruneStateCommplete

	var d = time.Since(t)
	ps.DeleteTime = d
	StatePruneDeleteTimer.Update(d)

	var (
		logf   = logging.Logger.Info
		logMsg = "prune client state stats"
	)

	if d > time.Second {
		logf = logging.Logger.Error
		logMsg = logMsg + " - slow"
	}

	logf(logMsg, zap.Int64("round", bs.Round),
		zap.String("block", bs.Hash),
		zap.String("state_hash", util.ToHex(bs.ClientStateHash)),
		zap.Int64("prune_deleted", ps.Deleted),
		zap.Duration("duration", time.Since(t)), zap.Any("stats", ps),
		zap.Duration("prune_below_version_after", d))

	/*
		if stateOut != nil {
			if err = util.IsMPTValid(mpt); err != nil {
				fmt.Fprintf(stateOut, "prune validation failure: %v %v\n", util.ToHex(mpt.GetRoot()), bs.Round)
				mpt.PrettyPrint(stateOut)
				stateOut.Sync()
				panic(err)
			}
		}*/
}
