package chain

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
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

// ComputeFinalizedBlock - compute the block that has been finalized */
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, lfbr int64, r round.RoundI) *block.Block {
	var (
		rn = r.GetRoundNumber()
		rd = r
	)

	if rn < lfbr {
		return r.GetHeaviestNotarizedBlock()
	}

	hnb := rd.GetHeaviestNotarizedBlock()
	if hnb == nil {
	}

	for rd != nil && rn > lfbr {
		if hnb := rd.GetHeaviestNotarizedBlock(); hnb != nil {
			return hnb
		}

		rn--
		rd = c.GetRound(rn)
	}

	logging.Logger.Error("finalize round - compute lfb failed",
		zap.Int64("pre round", rn),
		zap.Int64("round", r.GetRoundNumber()))
	return nil
}

/*FinalizeRound - starting from the given round work backwards and identify the round that can be assumed to be finalized as all forks after
that extend from a single block in that round. */
func (c *Chain) FinalizeRound(r round.RoundI) {
	if r.IsFinalized() {
		return // round already finalized
	}
	// The SetFinalizing is not condition check it changes round state.
	if !r.SetFinalizing() {
		logging.Logger.Debug("finalize_round: already finalizing",
			zap.Int64("round", r.GetRoundNumber()))
		if node.Self.Type == node.NodeTypeSharder {
			return
		}
	}
	if r.GetHeaviestNotarizedBlock() == nil {
		logging.Logger.Error("finalize round: no notarized blocks",
			zap.Int64("round", r.GetRoundNumber()))
		go c.GetHeaviestNotarizedBlock(context.Background(), r)
		time.Sleep(FINALIZATION_TIME)
	}

	logging.Logger.Debug("finalize round", zap.Int64("round", r.GetRoundNumber()),
		zap.Int64("lf_round", c.GetLatestFinalizedBlock().Round))
	select {
	case c.finalizedRoundsChannel <- r:
	case <-time.NewTimer(500 * time.Millisecond).C: // TODO: make the timeout configurable
		logging.Logger.Info("finalize round - push round to finalizedRoundsChannel timeout",
			zap.Int64("round", r.GetRoundNumber()))
	}
}

func (c *Chain) finalizeRound(ctx context.Context, r round.RoundI) {
	roundNumber := r.GetRoundNumber()
	notarizedBlocks := r.GetNotarizedBlocks()
	nbCount := len(notarizedBlocks)
	plfb := c.GetLatestFinalizedBlock()
	logging.Logger.Info("finalize round",
		zap.Int64("round", roundNumber),
		zap.Int64("lf_round", plfb.Round),
		zap.Int("num_round_notarized", nbCount),
		zap.Int("num_chain_notarized", len(c.NotarizedBlocksCounts)))
	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du > 3*time.Second {
			logging.Logger.Debug("finalize round slow",
				zap.Int64("round", roundNumber),
				zap.Any("duration", time.Since(ts)))
		}
	}()

	if nbCount == 0 {
		c.ZeroNotarizedBlocksCount++
	} else if nbCount > 1 {
		c.MultiNotarizedBlocksCount++
	}

	genNum := c.GetGeneratorsNumOfRound(roundNumber)
	if nbCount > genNum {
		for _, blk := range notarizedBlocks {
			logging.Logger.Info("too many notarized blocks",
				zap.Int64("round", roundNumber),
				zap.String("hash", blk.Hash),
				zap.Int64("RRS", blk.GetRoundRandomSeed()),
				zap.Int("block_timeout_count", blk.RoundTimeoutCount))
		}
	}

	// expand NotarizedBlocksCount array size if generators number is greater than it
	if genNum > len(c.NotarizedBlocksCounts) {
		newCounts := make([]int64, genNum+1)
		copy(newCounts, c.NotarizedBlocksCounts)
		c.NotarizedBlocksCounts = newCounts
	}

	if nbCount < len(c.NotarizedBlocksCounts) {
		c.NotarizedBlocksCounts[nbCount]++
	}

	// This check is useful when we allow the finalizeRound route is not sequential and end up with out-of-band execution
	if rn := r.GetRoundNumber(); rn <= plfb.Round {
		logging.Logger.Error("finalize round - round number <= latest finalized round",
			zap.Int64("round", roundNumber),
			zap.Int64("lf_round", plfb.Round))
		return
	}

	lfb := c.ComputeFinalizedBlock(ctx, plfb.Round, r)
	if lfb == nil {
		logging.Logger.Debug("finalize round - no decisive block to finalize yet"+
			" or don't have all the necessary blocks",
			zap.Int64("round", roundNumber),
			zap.Int("notarized_blocks_count", nbCount))
		return
	}
	if lfb.Hash == plfb.Hash {
		logging.Logger.Debug("lfb round is the same as latest lfb",
			zap.Int64("round", roundNumber),
			zap.Int64("lfb round", lfb.Round),
			zap.Int64("plfb round", plfb.Round))
		return
	}

	if lfb.Round > plfb.Round {
		frchain := make([]*block.Block, 0, 1)
		for b := lfb; b != nil && b.Hash != plfb.Hash; b = b.PrevBlock {
			frchain = append(frchain, b)
		}
		if len(frchain) == 0 {
			logging.Logger.Error("finalize round - could not reach to latest finalized block",
				zap.Int64("round", roundNumber),
				zap.Int64("lfb", plfb.Round))
			return
		}

		fb := frchain[len(frchain)-1]
		if fb.PrevBlock == nil {
			pb := c.GetPreviousBlock(ctx, fb)
			if pb == nil {
				logging.Logger.Error("finalize round (missed blocks)",
					zap.Int64("round", roundNumber),
					zap.Int64("from", plfb.Round+1),
					zap.Int64("to", fb.Round-1))
				c.MissedBlocks += fb.Round - 1 - plfb.Round
			}
		}

		// perform view change (or not perform)
		if err := c.viewChanger.ViewChange(ctx, lfb); err != nil {
			logging.Logger.Error("view_changing_lfb",
				zap.Int64("round", roundNumber),
				zap.Int64("lfb_round", lfb.Round),
				zap.Error(err))
			return
		}
		FinalizationLagMetric.Update(int64(c.GetCurrentRound() - lfb.Round))

		logging.Logger.Info("finalize round - latest finalized round",
			zap.Int64("round", roundNumber),
			zap.Int64("lfb round", lfb.Round),
			zap.String("lfb block", lfb.Hash))
		for idx := range frchain {
			fb := frchain[len(frchain)-1-idx]
			if pb := c.GetLocalPreviousBlock(ctx, fb); pb == nil {
				logging.Logger.Error("finalize round - get previous block failed",
					zap.Int64("round", fb.Round))
				return
			}

			_, _, err := c.createRoundIfNotExist(ctx, fb)
			if err != nil {
				logging.Logger.Error("create round for finalize block failed",
					zap.Int64("round", fb.Round),
					zap.String("hash", fb.Hash))
			}

			select {
			case <-ctx.Done():
				logging.Logger.Info("finalize round - context done",
					zap.Error(ctx.Err()),
					zap.Int64("round", roundNumber))
				return
			case c.finalizedBlocksChannel <- fb:
				logging.Logger.Info("finalize round",
					zap.Int64("round", fb.Round),
					zap.String("block", fb.Hash))
			case <-time.NewTimer(500 * time.Millisecond).C: // TODO: make the timeout configurable
				logging.Logger.Error("finalize round - push fb to finalizedBlocksChannel timeout",
					zap.Int64("round", roundNumber),
					zap.Int64("fb_round", fb.Round))
				continue
			}
		}
		// Prune the chain from the oldest finalized block
		c.PruneChain(ctx, frchain[len(frchain)-1])
		return
	}

	logging.Logger.Info("finalize round - lfb round <= plfb round",
		zap.Int64("lfb round", lfb.Round),
		zap.Int64("plfb round", plfb.Round))
	b := c.commonAncestor(ctx, plfb, lfb)
	if b != nil {
		// Recovering from incorrectly finalized block
		c.RollbackCount++
		rl := plfb.Round - b.Round
		if c.LongestRollbackLength < int8(rl) {
			c.LongestRollbackLength = int8(rl)
		}
		logging.Logger.Error("finalize round - rolling back finalized block", zap.Int64("round", roundNumber),
			zap.Int64("cf_round", plfb.Round), zap.String("cf_block", plfb.Hash), zap.String("cf_prev_block", plfb.PrevHash),
			zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash), zap.Int64("caf_round", b.Round), zap.String("caf_block", b.Hash))
		for r := roundNumber; r >= b.Round; r-- {
			rr := c.GetRound(r)
			if rr != nil {
				for _, nb := range rr.GetNotarizedBlocks() {
					logging.Logger.Error("finalize round - rolling back, round nb",
						zap.Int64("round", roundNumber),
						zap.Int64("notarized_round", nb.Round),
						zap.Int("notarized_round_rank", nb.RoundRank),
						zap.String("notarized_block", nb.Hash))
				}
			}
		}
		for cfb := plfb.PrevBlock; cfb != nil && cfb != b; cfb = cfb.PrevBlock {
			logging.Logger.Error("finalize round - rolling back finalized block -> ",
				zap.Int64("round", cfb.Round),
				zap.String("block", cfb.Hash))
		}
		// perform view change or not perform
		if err := c.viewChanger.ViewChange(ctx, b); err != nil {
			logging.Logger.Error("view_changing_lfb",
				zap.Int64("lfb_round", b.Round),
				zap.Error(err))
			return
		}
		c.SetLatestOwnFinalizedBlockRound(b.Round)
		c.SetLatestFinalizedBlock(b)
		return
	}
	logging.Logger.Error("finalize round - missing common ancestor", zap.Int64("cf_round", plfb.Round), zap.String("cf_block", plfb.Hash), zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash))
}

func (c *Chain) createRoundIfNotExist(ctx context.Context, b *block.Block) (round.RoundI, *block.Block, error) {
	if r := c.GetRound(b.Round); r != nil {
		return r, b, nil
	}

	currentRound := c.GetCurrentRound()
	// create the round if it does not exist
	r := c.RoundF.CreateRoundF(b.Round)
	var err error
	b, r, err = c.AddNotarizedBlockToRound(r, b)
	if err != nil {
		logging.Logger.Error("createRoundIfNotExist - add notarized block to round failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("current_round", currentRound),
			zap.Error(err))
		return nil, nil, err
	}
	b, _, err = r.AddNotarizedBlock(b)
	if err != nil {
		logging.Logger.Error("createRoundIfNotExist - add notarized block failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("current_round", currentRound),
			zap.Error(err))
		return nil, nil, err
	}

	// Add the round if chain does not have it
	r = c.AddRound(r)
	return r, b, nil
}

// GetHeaviestNotarizedBlock - get a notarized block for a round.
// TODO: move to the place where getNotarizedBlockFromMiners() is implemented, this is kind of
// duplicate actions here
func (c *Chain) GetHeaviestNotarizedBlock(ctx context.Context, r round.RoundI) *block.Block {

	var (
		rn     = r.GetRoundNumber()
		params = &url.Values{}
	)

	params.Add("round", fmt.Sprintf("%v", rn))

	cctx, cancel := context.WithTimeout(ctx, node.TimeoutLargeMessage)
	defer cancel()

	notarizedBlockC := make(chan *block.Block, 10)
	var handler = func(_ context.Context, entity datastore.Entity) (
		resp interface{}, err error) {
		logging.Logger.Info("get notarized block for round in handler", zap.Int64("round", rn),
			zap.String("block", entity.GetKey()))

		// cancel further requests and return when a notarized block is acquired
		if b := r.GetHeaviestNotarizedBlock(); b != nil {
			select {
			case notarizedBlockC <- b:
			default:
			}
			cancel()
			return b, nil
		}

		var nb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if nb.Round != rn {
			return nil, common.NewError("invalid_block",
				"Block not from the requested round")
		}

		if err = nb.Validate(cctx); err != nil {
			logging.Logger.Error("get notarized block for round - validate",
				zap.Int64("round", rn), zap.String("block", nb.Hash),
				zap.Error(err))
			return nil, err
		}

		cancel()
		select {
		case notarizedBlockC <- nb:
		default:
		}
		return nb, nil
	}

	c.RequestEntityFromMinersOnMB(cctx, c.getLatestFinalizedMagicBlock(cctx), MinerNotarizedBlockRequestor, params, handler)
	var nb *block.Block
	select {
	case nb = <-notarizedBlockC:
	default:
	}

	if nb == nil {
		logging.Logger.Error("get no notarized block", zap.Int64("round", rn))
		return nil
	}

	err := c.VerifyNotarization(ctx, nb, nb.GetVerificationTickets(), rn)
	if err != nil {
		logging.Logger.Error("get notarized block for round - validate notarization",
			zap.Int64("round", rn), zap.String("block", nb.Hash),
			zap.Error(err))
		return nil
	}

	if nb.RoundTimeoutCount != r.GetTimeoutCount() {
		logging.Logger.Info("Timeout count on Round and NB are out-of-sync",
			zap.Int64("round", rn),
			zap.Int("nb_toc", nb.RoundTimeoutCount),
			zap.Int("round_toc", r.GetTimeoutCount()))
	}

	// This is a notarized block. So, use this method to sync round info with the notarized block.
	b, r, err := c.AddNotarizedBlockToRound(r, nb)
	if err != nil {
		logging.Logger.Error("get notarized block for round failed",
			zap.Int64("round", rn),
			zap.String("block", nb.Hash),
			zap.String("miner", nb.MinerID),
			zap.Error(err))
		return nil
	}

	// TODO: this may not be the best round block or the best chain weight
	// block. Do we do that extra work?
	logging.Logger.Debug("get notarized block, add block to round",
		zap.Int64("round", rn),
		zap.String("block", b.Hash))
	b, _, err = r.AddNotarizedBlock(b)
	if err != nil {
		logging.Logger.Error("get notarized block for round failed",
			zap.Int64("round", rn),
			zap.String("block", nb.Hash),
			zap.String("miner", nb.MinerID),
			zap.Error(err))
		return nil
	}
	return r.GetHeaviestNotarizedBlock()
}

// GetHeaviestNotarizedBlockLight - get a notarized block for a round.
func (c *Chain) GetHeaviestNotarizedBlockLight(ctx context.Context, r int64) *block.Block {
	params := &url.Values{}

	params.Add("round", fmt.Sprintf("%v", r))

	cctx, cancel := context.WithTimeout(ctx, node.TimeoutLargeMessage)
	defer cancel()

	notarizedBlockC := make(chan *block.Block, 1)
	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {
		logging.Logger.Info("get notarized block for round", zap.Int64("round", r),
			zap.String("block", entity.GetKey()))

		var nb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if nb.Round != r {
			return nil, common.NewError("invalid_block",
				"Block not from the requested round")
		}

		select {
		case notarizedBlockC <- nb:
		default:
		}
		cancel()
		return nb, nil
	}

	c.RequestEntityFromMinersOnMB(cctx, c.GetCurrentMagicBlock(), MinerNotarizedBlockRequestor, params, handler)
	var nb *block.Block
	select {
	case nb = <-notarizedBlockC:
	default:
	}

	return nb
}

// GetLatestFinalizedMagicBlockFromShardersOn - request for latest finalized
// magic blocks from all the sharders. It uses provided MagicBlock to get list
// of sharders to request data from, and returns the block with highest magic
// block starting round.
func (c *Chain) GetLatestFinalizedMagicBlockFromShardersOn(ctx context.Context,
	mb *block.MagicBlock) *block.Block {
	if mb == nil {
		return nil
	}

	var (
		sharders = mb.Sharders

		listMutex sync.Mutex
	)

	magicBlocks := make([]*block.Block, 0, 1)

	var errs []error
	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {
		var mb, ok = entity.(*block.Block)
		if !ok || mb == nil {
			errs = append(errs, datastore.ErrInvalidEntity)
			return nil, datastore.ErrInvalidEntity
		}

		listMutex.Lock()
		defer listMutex.Unlock()

		for _, b := range magicBlocks {
			if b.Hash == mb.Hash {
				return mb, nil
			}
		}
		magicBlocks = append(magicBlocks, mb)

		return mb, nil
	}

	sharders.RequestEntityFromAll(ctx, LatestFinalizedMagicBlockRequestor, nil, handler)

	if len(magicBlocks) == 0 && len(errs) > 0 {
		logging.Logger.Error("Get latest finalized magic block from sharders failed", zap.Errors("errors", errs))
	}

	if len(magicBlocks) == 0 {
		return nil
	}

	if len(magicBlocks) > 1 {
		sort.Slice(magicBlocks, func(i, j int) bool {
			if magicBlocks[i].StartingRound == magicBlocks[j].StartingRound {
				return magicBlocks[i].Round > magicBlocks[j].Round
			}

			return magicBlocks[i].StartingRound > magicBlocks[j].StartingRound
		})
	}

	logging.Logger.Debug("get latest finalized magic block from sharders",
		zap.Int64("mb_num", magicBlocks[0].MagicBlockNumber),
		zap.Int64("mb_sr", magicBlocks[0].StartingRound))
	return magicBlocks[0]
}

// GetLatestFinalizedMagicBlockFromSharders - request for latest finalized magic
// block from all the sharders. It uses GetLatestFinalizedMagicBlock to get latest
// finalized magic block of sharders to request data from.
func (c *Chain) GetLatestFinalizedMagicBlockFromSharders(ctx context.Context) *block.Block {
	return c.GetLatestFinalizedMagicBlockFromShardersOn(ctx, c.getLatestFinalizedMagicBlock(ctx))
}

// GetLatestFinalizedMagicBlockRound returns LFMB for given round number
func (c *Chain) GetLatestFinalizedMagicBlockRound(rn int64) *block.Block {
	lfmb := c.GetLatestFinalizedMagicBlock(context.Background())
	// TODO: improve this lfmbMutex
	c.lfmbMutex.RLock()
	defer c.lfmbMutex.RUnlock()
	rn = mbRoundOffset(rn) // round number with mb offset
	if len(c.magicBlockStartingRounds) > 0 {
		lfmbr := int64(-1)
		for r := range c.magicBlockStartingRounds {
			if r <= rn && r > lfmbr {
				lfmbr = r
			}
		}
		if lfmbr >= 0 {
			lfmb = c.magicBlockStartingRounds[lfmbr]
		}
	}
	return lfmb
}

func getMagicBlockBrief(b *block.Block) *MagicBlockBrief {
	if b == nil || b.MagicBlock == nil {
		return nil
	}
	return &MagicBlockBrief{
		Round:            b.Round,
		MagicBlockNumber: b.MagicBlock.MagicBlockNumber,
		MagicBlockHash:   b.MagicBlock.Hash,
		StartingRound:    b.MagicBlock.StartingRound,
		MinersN2NURLs:    b.MagicBlock.Miners.N2NURLs(),
		ShardersN2NURLs:  b.MagicBlock.Sharders.N2NURLs(),
	}
}
