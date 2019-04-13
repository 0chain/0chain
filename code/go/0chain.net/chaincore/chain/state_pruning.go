package chain

import (
	"0chain.net/chaincore/node"
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
	var bs *block.BlockSummary
	lfb := c.LatestFinalizedBlock
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
			return
		}
	}
	if bs == nil {
		bs = &block.BlockSummary{Round: lfb.Round, ClientStateHash: lfb.ClientStateHash}
	}
	Logger.Info("prune client state - new version", zap.Int64("current_round", c.CurrentRound), zap.Int64("latest_finalized_round", c.LatestFinalizedBlock.Round), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)))
	newVersion := util.Sequence(bs.Round)
	if c.pruneStats != nil && c.pruneStats.Version == newVersion && c.pruneStats.MissingNodes == 0 {
		return // already done with pruning this
	}
	mpt := util.NewMerklePatriciaTrie(c.stateDB, newVersion)
	mpt.SetRoot(bs.ClientStateHash)
	pctx := util.WithPruneStats(ctx)
	ps := util.GetPruneStats(pctx)
	ps.Stage = util.PruneStateUpdate
	c.pruneStats = ps
	t := time.Now()
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
	err := mpt.UpdateVersion(pctx, newVersion, missingNodesHandler)
	d1 := time.Since(t)
	ps.UpdateTime = d1
	StatePruneUpdateTimer.Update(d1)
	node.GetSelfNode(ctx).Info.StateMissingNodes = ps.MissingNodes
	if err != nil {
		Logger.Error("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)), zap.Any("prune_stats", ps), zap.Error(err))
		if ps.MissingNodes > 0 {
			if len(missingKeys) > 0 {
				c.GetStateNodes(ctx, missingKeys[:])
			}
			ps.Stage = util.PruneStateAbandoned
			return
		}
	} else {
		Logger.Info("prune client state (update origin)", zap.Int64("current_round", c.CurrentRound), zap.Int64("round", bs.Round), zap.String("block", bs.Hash), zap.String("state_hash", util.ToHex(bs.ClientStateHash)), zap.Any("prune_stats", ps))
	}
	if c.LatestFinalizedBlock.Round-int64(c.PruneStateBelowCount) < bs.Round {
		ps.Stage = util.PruneStateAbandoned
		return
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
		zap.Duration("duration", time.Since(t)), zap.Any("stats", ps))
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
