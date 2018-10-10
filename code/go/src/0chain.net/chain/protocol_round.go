package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/config"
	"0chain.net/node"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
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

//SteadyStateFinalizationTimer - a metric that tracks the steady state finality time (time between two successive finalized blocks in steady state)
var SteadyStateFinalizationTimer metrics.Timer
var ssFTs time.Time

//StartToFinalizeTimer - a metric that tracks the time a block is created to finalized
var StartToFinalizeTimer metrics.Timer

func init() {
	if SteadyStateFinalizationTimer != nil {
		metrics.Unregister("ss_finalization_time")
	}
	SteadyStateFinalizationTimer = metrics.GetOrRegisterTimer("ss_finalization_time", nil)
	if StartToFinalizeTimer != nil {
		metrics.Unregister("s2f_time")
	}
	StartToFinalizeTimer = metrics.GetOrRegisterTimer("s2f_time", nil)
}

/*FinalizeRound - starting from the given round work backwards and identify the round that can be
  assumed to be finalized as only one chain has survived.
  Note: It is that round and prior that actually get finalized.
*/
func (c *Chain) FinalizeRound(ctx context.Context, r round.RoundI, bsh BlockStateHandler) {
	if !r.SetFinalizing() {
		return
	}
	if r.GetBestNotarizedBlock() == nil {
		Logger.Error("finalize round: no notarized blocks", zap.Int64("round", r.GetRoundNumber()))
		go c.GetNotarizedBlockForRound(r)
	} else {
		for _, nb := range r.GetNotarizedBlocks() {
			if nb.PrevBlock == nil {
				Logger.Error("finalize round: get previous block", zap.Int64("round", r.GetRoundNumber()), zap.String("block", nb.Hash), zap.String("prev_block", nb.PrevHash))
				go c.GetPreviousBlock(ctx, nb)
			}
		}
	}
	time.Sleep(FINALIZATION_TIME)
	c.finalizedRoundsChannel <- r
}

func (c *Chain) finalizeRound(ctx context.Context, r round.RoundI, bsh BlockStateHandler) {
	notarizedBlocks := len(r.GetNotarizedBlocks())
	lfb := c.ComputeFinalizedBlock(ctx, r)
	roundNumber := r.GetRoundNumber()
	if lfb == nil {
		Logger.Debug("finalize round - no decisive block to finalize yet or don't have all the necessary blocks", zap.Int64("round", roundNumber), zap.Int("notarized_blocks", notarizedBlocks))
		return
	}
	if lfb.Hash == c.LatestFinalizedBlock.Hash {
		return
	}
	if lfb.Round <= c.LatestFinalizedBlock.Round {
		b := c.commonAncestor(ctx, c.LatestFinalizedBlock, lfb)
		if b != nil {
			// Recovering from incorrectly finalized block
			c.RollbackCount++
			rl := c.LatestFinalizedBlock.Round - b.Round
			if c.LongestRollbackLength < int8(rl) {
				c.LongestRollbackLength = int8(rl)
			}
			Logger.Error("finalize round - rolling back finalized block",
				zap.Int64("cf_round", c.LatestFinalizedBlock.Round), zap.String("cf_block", c.LatestFinalizedBlock.Hash), zap.String("cf_prev_block", c.LatestFinalizedBlock.PrevHash),
				zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash), zap.Int64("caf_round", b.Round), zap.String("caf_block", b.Hash))
			for cfb := c.LatestFinalizedBlock.PrevBlock; cfb != nil && cfb != b; cfb = cfb.PrevBlock {
				Logger.Error("finalize round - rolling back finalized block -> ", zap.Int64("round", cfb.Round), zap.String("block", cfb.Hash))
			}
			c.LatestFinalizedBlock = b
			return
		} else {
			Logger.Error("finalize round - missing common ancestor", zap.Int64("cf_round", c.LatestFinalizedBlock.Round), zap.String("cf_block", c.LatestFinalizedBlock.Hash), zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash))
		}
	}
	plfb := c.LatestFinalizedBlock
	plfbHash := plfb.Hash
	frchain := make([]*block.Block, 0, 1)
	for b := lfb; b != nil && b.Hash != plfbHash; b = b.PrevBlock {
		frchain = append(frchain, b)
	}
	fb := frchain[len(frchain)-1]

	/*
		if fb.Round > roundNumber-1-int64(c.LongestRollbackLength) {
			// Avoid early finalization if we only have one block
			if notarizedBlocks == 1 {
				Logger.Error("compute finalized block - too early to decide", zap.Int64("round", roundNumber), zap.Int64("fb_round", fb.Round))
				return
			}
		}*/

	if fb.PrevBlock == nil {
		pb := c.GetPreviousBlock(ctx, fb)
		if pb == nil {
			Logger.Error("finalize round (missed blocks)", zap.Int64("from", plfb.Round+1), zap.Int64("to", fb.Round-1))
			c.MissedBlocks += fb.Round - 1 - plfb.Round
		}
	}
	c.LatestFinalizedBlock = lfb
	deadBlocks := make([]*block.Block, 0, 1)
	for idx := range frchain {
		fb := frchain[len(frchain)-1-idx]
		bNode := node.GetNode(fb.MinerID)
		ms := bNode.ProtocolStats.(*MinerStats)
		ms.FinalizationCountByRank[fb.RoundRank]++
		Logger.Info("finalize round", zap.Int64("round", roundNumber), zap.Int64("finalized_round", fb.Round), zap.String("hash", fb.Hash), zap.Int8("state", fb.GetBlockState()))
		if time.Since(ssFTs) < 20*time.Second {
			SteadyStateFinalizationTimer.UpdateSince(ssFTs)
		}
		StartToFinalizeTimer.UpdateSince(fb.ToTime())
		ssFTs = time.Now()
		c.UpdateChainInfo(fb)
		if fb.GetStateStatus() != block.StateSuccessful {
			err := c.ComputeState(ctx, fb)
			if err != nil {
				if config.DevConfiguration.State {
					Logger.Error("finalize round state not successful", zap.Int64("round", roundNumber), zap.Int64("finalized_round", fb.Round), zap.String("hash", fb.Hash), zap.Int8("state", fb.GetBlockState()), zap.Error(err))
					Logger.DPanic("finalize block - state not successful")
				}
			}
		}
		if fb.ClientState != nil {
			ts := time.Now()
			err := fb.ClientState.SaveChanges(c.stateDB, false)
			if err != nil {
				Logger.Error("finalize round - save state", zap.Int64("round", fb.Round), zap.String("block", fb.Hash), zap.String("client_state", util.ToHex(fb.ClientStateHash)), zap.Int("changes", len(fb.ClientState.GetChangeCollector().GetChanges())), zap.Duration("time", time.Since(ts)), zap.Error(err))
			} else {
				Logger.Info("finalize round - save state", zap.Int64("round", fb.Round), zap.String("block", fb.Hash), zap.String("client_state", util.ToHex(fb.ClientStateHash)), zap.Int("changes", len(fb.ClientState.GetChangeCollector().GetChanges())), zap.Duration("time", time.Since(ts)))
			}
			c.rebaseState(fb)
		}
		bsh.UpdateFinalizedBlock(ctx, fb)
		c.BlockChain.Value = fb.GetSummary()
		c.BlockChain = c.BlockChain.Next()
		frb := c.GetRoundBlocks(fb.Round)
		for _, b := range frb {
			if b.Hash != fb.Hash {
				deadBlocks = append(deadBlocks, b)
			}
		}
	}
	// Prune all the dead blocks
	c.DeleteBlocks(deadBlocks)
	// Prune the chain from the oldest finalized block
	c.PruneChain(ctx, frchain[len(frchain)-1])
}

/*GetNotarizedBlockForRound - get a notarized block for a round */
func (c *Chain) GetNotarizedBlockForRound(r round.RoundI) *block.Block {
	nbrequestor := MinerNotarizedBlockRequestor
	roundNumber := r.GetRoundNumber()
	params := map[string]string{"round": fmt.Sprintf("%v", roundNumber)}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Info("get notarized block for round", zap.Int64("round", roundNumber), zap.String("block", entity.GetKey()))
		if roundNumber+1 != c.CurrentRound {
			cancelf()
			return nil, nil
		}
		if r.GetBestNotarizedBlock() != nil {
			cancelf()
			return nil, nil
		}
		nb, ok := entity.(*block.Block)
		if !ok {
			return nil, common.NewError("invalid_entity", "Invalid entity")
		}
		if nb.Round != roundNumber {
			return nil, common.NewError("invalid_block", "Block not from the requested round")
		}
		if err := c.VerifyNotarization(ctx, nb.Hash, nb.VerificationTickets); err != nil {
			Logger.Error("get notarized block for round - validate notarization", zap.Int64("round", roundNumber), zap.Error(err))
			return nil, err
		}
		if err := nb.Validate(ctx); err != nil {
			Logger.Error("get notarized block for round - validate", zap.Int64("round", roundNumber), zap.Error(err))
			return nil, err
		}
		b := c.AddBlock(nb)
		//TODO: this may not be the best round block or the best chain weight block. Do we do that extra work?
		b, _ = r.AddNotarizedBlock(b)
		Logger.Info("get notarized block", zap.Int64("round", roundNumber), zap.String("block", b.Hash), zap.String("state", util.ToHex(b.ClientStateHash)), zap.String("prev_block", b.PrevHash))
		return b, nil
	}
	n2n := c.Miners
	n2n.RequestEntity(ctx, nbrequestor, params, handler)
	return r.GetBestNotarizedBlock()
}
