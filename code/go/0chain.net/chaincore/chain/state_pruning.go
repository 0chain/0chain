package chain

import (
	"context"
	"time"

	. "0chain.net/core/logging"
	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/block"
	"0chain.net/core/util"
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
	bc := c.BlockChain
	bc = bc.Move(-c.PruneStateBelowCount)
	for i := 0; i < 10 && bc.Value == nil; i++ {
		bc = bc.Prev()
	}
	if bc.Value == nil {
		return
	}
	bs := bc.Value.(*block.BlockSummary)
	for bs.Round%100 != 0 {
		bc = bc.Prev()
		if bc.Value == nil {
			return
		}
		bs = bc.Value.(*block.BlockSummary)
	}
	newVersion := util.Sequence(bs.Round)

	if c.pruneStats != nil && c.pruneStats.Version == newVersion {
		return // already done with pruning this
	}
	mpt := util.NewMerklePatriciaTrie(c.stateDB, newVersion)
	mpt.SetRoot(bs.ClientStateHash)
	Logger.Info("prune client state - new version", zap.Int64("current_round", c.CurrentRound), zap.Int64("latest_finalized_round", c.LatestFinalizedBlock.Round), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)))
	pctx := util.WithPruneStats(ctx)
	ps := util.GetPruneStats(pctx)
	var missingKeys []util.Key
	missingNodesHandler := func(ctx context.Context, path util.Path, key util.Key) error {
		missingKeys = append(missingKeys, key)
		if len(missingKeys) == 1000 {
			stage := ps.Stage
			ps.Stage = util.PruneStateSynch
			c.GetStateNodes(ctx, missingKeys[:])
			ps.Stage = stage
			missingKeys = nil
		}
		return nil
	}
	ps.Stage = util.PruneStateUpdate
	c.pruneStats = ps
	t := time.Now()
	err := mpt.UpdateVersion(pctx, newVersion, missingNodesHandler)
	d1 := time.Since(t)
	ps.UpdateTime = d1
	StatePruneUpdateTimer.Update(d1)
	if err != nil {
		Logger.Error("prune client state (update origin)", zap.Error(err))
		if ps.MissingNodes > 0 {
			if len(missingKeys) > 0 {
				c.GetStateNodes(ctx, missingKeys[:])
			}
			return
		}
	} else {
		Logger.Info("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)), zap.Duration("time", d1))
	}
	t1 := time.Now()
	ps.Stage = util.PruneStateDelete
	err = c.stateDB.PruneBelowVersion(pctx, newVersion)
	if err != nil {
		Logger.Error("prune client state error", zap.Error(err))
	}
	ps.Stage = util.PruneStateCommplete
	d2 := time.Since(t1)
	ps.DeleteTime = d2
	StatePruneDeleteTimer.Update(d2)
	logf := Logger.Info
	if d1 > time.Second || d2 > time.Second {
		logf = Logger.Error
	}
	logf("prune client state stats", zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)),
		zap.Duration("duration", time.Since(t)), zap.Duration("update", d1), zap.Duration("prune", d2), zap.Any("stats", ps))
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
