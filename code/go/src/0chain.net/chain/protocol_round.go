package chain

import (
	"context"
	"fmt"
	"time"
  
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

//StartToFinalizeTxnTimer - a metric that trakcs the time a txn is created to finalized
var StartToFinalizeTxnTimer metrics.Timer

//FinalizationLagMetric - a metric that tracks how much is the lag between current round and finalization round
var FinalizationLagMetric metrics.Histogram

func init() {
	SteadyStateFinalizationTimer = metrics.GetOrRegisterTimer("ss_finalization_time", nil)
	StartToFinalizeTimer = metrics.GetOrRegisterTimer("s2f_time", nil)
	StartToFinalizeTxnTimer = metrics.GetOrRegisterTimer("s2ft_time", nil)
	FinalizationLagMetric = metrics.NewHistogram(metrics.NewUniformSample(1024))
	metrics.Register("finalization_lag", FinalizationLagMetric)
}

/*ComputeFinalizedBlock - compute the block that has been finalized */
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, r round.RoundI) *block.Block {
	isIn := func(blocks []*block.Block, hash string) bool {
		for _, b := range blocks {
			if b.Hash == hash {
				return true
			}
		}
		return false
	}
	roundNumber := r.GetRoundNumber()
	tips := r.GetNotarizedBlocks()
	if len(tips) == 0 {
		Logger.Error("compute finalize block: no notarized blocks", zap.Int64("round", r.GetRoundNumber()))
		return nil
	}
	for true {
		ntips := make([]*block.Block, 0, 1)
		for _, b := range tips {
			if b.PrevBlock == nil {
				pb := c.GetPreviousBlock(ctx, b)
				if pb == nil {
					Logger.Error("compute finalized block: null prev block", zap.Int64("round", roundNumber), zap.Int64("block_round", b.Round), zap.String("block", b.Hash))
					return nil
				}
			}
			if isIn(ntips, b.PrevHash) {
				continue
			}
			ntips = append(ntips, b.PrevBlock)
		}
		tips = ntips
		if len(tips) == 1 {
			break
		}
	}
	if len(tips) != 1 {
		return nil
	}
	fb := tips[0]
	if fb.Round == r.GetRoundNumber() {
		return nil
	}
	return fb
}

/*FinalizeRound - starting from the given round work backwards and identify the round that can be assumed to be finalized as all forks after
that extend from a single block in that round. */
func (c *Chain) FinalizeRound(ctx context.Context, r round.RoundI, bsh BlockStateHandler) {
	if !r.SetFinalizing() {
		return
	}
	if r.GetHeaviestNotarizedBlock() == nil {
		Logger.Error("finalize round: no notarized blocks", zap.Int64("round", r.GetRoundNumber()))
		go c.GetHeaviestNotarizedBlock(r)
	}
	time.Sleep(FINALIZATION_TIME)
	Logger.Debug("finalize round", zap.Int64("round", r.GetRoundNumber()), zap.Int64("lf_round", c.LatestFinalizedBlock.Round))
	c.finalizedRoundsChannel <- r
}

func (c *Chain) finalizeRound(ctx context.Context, r round.RoundI, bsh BlockStateHandler) {
	roundNumber := r.GetRoundNumber()
	Logger.Debug("finalize round worker", zap.Int64("round", roundNumber), zap.Int64("lf_round", c.LatestFinalizedBlock.Round))
	notarizedBlocks := r.GetNotarizedBlocks()
	nbCount := len(notarizedBlocks)
	if nbCount == 0 {
		c.ZeroNotarizedBlocksCount++
	}
	if nbCount > 1 {
		c.MultiNotarizedBlocksCount++
	}
	c.NotariedBlocksCounts[nbCount]++
	//This check is useful when we allow the finalizeRound route is not sequential and end up with out-of-band execution
	if r.GetRoundNumber() <= c.LatestFinalizedBlock.Round {
		Logger.Error("finalize round worker - round number <= latest finalized round", zap.Int64("round", r.GetRoundNumber()), zap.Int64("lf_round", c.LatestFinalizedBlock.Round))
		return
	}
	lfb := c.ComputeFinalizedBlock(ctx, r)
	if lfb == nil {
		Logger.Debug("finalize round - no decisive block to finalize yet or don't have all the necessary blocks", zap.Int64("round", roundNumber), zap.Int("notarized_blocks", nbCount))
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
			Logger.Error("finalize round - rolling back finalized block", zap.Int64("round", roundNumber),
				zap.Int64("cf_round", c.LatestFinalizedBlock.Round), zap.String("cf_block", c.LatestFinalizedBlock.Hash), zap.String("cf_prev_block", c.LatestFinalizedBlock.PrevHash),
				zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash), zap.Int64("caf_round", b.Round), zap.String("caf_block", b.Hash))
			for r := roundNumber; r >= b.Round; r-- {
				round := c.GetRound(r)
				if round != nil {
					for _, nb := range round.GetNotarizedBlocks() {
						Logger.Error("finalize round - rolling back, round nb", zap.Int64("round", nb.Round), zap.Int("round_rank", nb.RoundRank), zap.String("block", nb.Hash))
					}
				}
			}
			for cfb := c.LatestFinalizedBlock.PrevBlock; cfb != nil && cfb != b; cfb = cfb.PrevBlock {
				Logger.Error("finalize round - rolling back finalized block -> ", zap.Int64("round", cfb.Round), zap.String("block", cfb.Hash))
			}
			c.LatestFinalizedBlock = b
			return
		}
		Logger.Error("finalize round - missing common ancestor", zap.Int64("cf_round", c.LatestFinalizedBlock.Round), zap.String("cf_block", c.LatestFinalizedBlock.Hash), zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash))
	}
	plfb := c.LatestFinalizedBlock
	plfbHash := plfb.Hash
	frchain := make([]*block.Block, 0, 1)
	for b := lfb; b != nil && b.Hash != plfbHash; b = b.PrevBlock {
		frchain = append(frchain, b)
	}
	fb := frchain[len(frchain)-1]
	if fb.PrevBlock == nil {
		pb := c.GetPreviousBlock(ctx, fb)
		if pb == nil {
			Logger.Error("finalize round (missed blocks)", zap.Int64("from", plfb.Round+1), zap.Int64("to", fb.Round-1))
			c.MissedBlocks += fb.Round - 1 - plfb.Round
		}
	}
	c.LatestFinalizedBlock = lfb
	FinalizationLagMetric.Update(int64(c.CurrentRound - c.LatestFinalizedBlock.Round))
	Logger.Info("finalize round - latest finalized round", zap.Int64("round", c.LatestFinalizedBlock.Round), zap.String("block", c.LatestFinalizedBlock.Hash))
	for idx := range frchain {
		fb := frchain[len(frchain)-1-idx]
		c.finalizedBlocksChannel <- fb
	}
	// Prune the chain from the oldest finalized block
	c.PruneChain(ctx, frchain[len(frchain)-1])
}

/*GetHeaviestNotarizedBlock - get a notarized block for a round */
func (c *Chain) GetHeaviestNotarizedBlock(r round.RoundI) *block.Block {
	nbrequestor := MinerNotarizedBlockRequestor
	roundNumber := r.GetRoundNumber()
	params := map[string]string{"round": fmt.Sprintf("%v", roundNumber)}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Info("get notarized block for round", zap.Int64("round", roundNumber), zap.String("block", entity.GetKey()))
		if b := r.GetHeaviestNotarizedBlock(); b != nil {
			cancelf()
			return b, nil
		}
		nb, ok := entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if nb.Round != roundNumber {
			return nil, common.NewError("invalid_block", "Block not from the requested round")
		}
		if err := c.VerifyNotarization(ctx, nb.Hash, nb.VerificationTickets); err != nil {
			Logger.Error("get notarized block for round - validate notarization", zap.Int64("round", roundNumber), zap.String("block", nb.Hash), zap.Error(err))
			return nil, err
		}
		if err := nb.Validate(ctx); err != nil {
			Logger.Error("get notarized block for round - validate", zap.Int64("round", roundNumber), zap.String("block", nb.Hash), zap.Error(err))
			return nil, err
		}
		c.SetRandomSeed(r, nb.RoundRandomSeed)
		b := c.AddRoundBlock(r, nb)
		//TODO: this may not be the best round block or the best chain weight block. Do we do that extra work?
		b, _ = r.AddNotarizedBlock(b)
		Logger.Info("get notarized block", zap.Int64("round", roundNumber), zap.String("block", b.Hash), zap.String("state", util.ToHex(b.ClientStateHash)), zap.String("prev_block", b.PrevHash))
		return b, nil
	}
	n2n := c.Miners
	n2n.RequestEntity(ctx, nbrequestor, params, handler)
	return r.GetHeaviestNotarizedBlock()
}
