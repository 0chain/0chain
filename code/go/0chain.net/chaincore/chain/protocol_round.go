package chain

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"sync"
	"time"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/config"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
)

var DELTA = 200 * time.Millisecond
var FINALIZATION_TIME = 2 * DELTA

/*SetNetworkRelayTime - setup the network relay time */
func SetNetworkRelayTime(delta time.Duration) {
	DELTA = delta
	FINALIZATION_TIME = 2 * delta
}

// SteadyStateFinalizationTimer - a metric that tracks the steady state finality time (time between two successive finalized blocks in steady state)
var SteadyStateFinalizationTimer metrics.Timer
var ssFTs time.Time

// StartToFinalizeTimer - a metric that tracks the time a block is created to finalized
var StartToFinalizeTimer metrics.Timer

// StartToFinalizeTxnTimer - a metric that trakcs the time a txn is created to finalized
var StartToFinalizeTxnTimer metrics.Timer
var StartToFinalizeTxnTypeTimer map[string]metrics.Timer

// FinalizationLagMetric - a metric that tracks how much is the lag between current round and finalization round
var FinalizationLagMetric metrics.Histogram

func init() {
	SteadyStateFinalizationTimer = metrics.GetOrRegisterTimer("ss_finalization_time", nil)
	StartToFinalizeTimer = metrics.GetOrRegisterTimer("s2f_time", nil)
	StartToFinalizeTxnTimer = metrics.GetOrRegisterTimer("s2ft_time", nil)
	StartToFinalizeTxnTypeTimer = make(map[string]metrics.Timer)

	FinalizationLagMetric = metrics.NewHistogram(metrics.NewUniformSample(1024))
	_ = metrics.Register("finalization_lag", FinalizationLagMetric)

	event.InitMetrics()
}

// ComputeFinalizedBlock iterates through all previous blocks of notarized block on round r until finds single notarized block on the round,
// returns this block
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, lfbr int64, r round.RoundI) *block.Block {
	isIn := func(blocks []*block.Block, hash string) bool {
		for _, b := range blocks {
			if b.Hash == hash {
				return true
			}
		}
		return false
	}

	var (
		roundNumber     = r.GetRoundNumber()
		rd              = r
		notarizedBlocks []*block.Block
	)

	for {
		if roundNumber <= lfbr {
			break
		}

		notarizedBlocks = rd.GetNotarizedBlocks()
		if len(notarizedBlocks) > 0 {
			break
		}
		roundNumber--
		rd = c.GetRound(roundNumber)
		if rd == nil {
			break
		}
	}

	if len(notarizedBlocks) == 0 {
		logging.Logger.Error("compute finalize block: no notarized blocks",
			zap.Int64("round", r.GetRoundNumber()))
		return nil
	}

	for {
		prevNotarizedBlocks := make([]*block.Block, 0, 1)
		for _, b := range notarizedBlocks {
			if b.PrevBlock == nil {
				pb := c.GetPreviousBlock(ctx, b)
				if pb == nil {
					logging.Logger.Error("compute finalized block: null prev block",
						zap.Int64("round", roundNumber),
						zap.Int64("block_round", b.Round),
						zap.String("block", b.Hash))
					return nil
				}
			}
			if isIn(prevNotarizedBlocks, b.PrevHash) {
				continue
			}
			prevNotarizedBlocks = append(prevNotarizedBlocks, b.PrevBlock)
		}
		notarizedBlocks = prevNotarizedBlocks
		if len(notarizedBlocks) == 1 {
			break
		}
	}

	if len(notarizedBlocks) != 1 {
		return nil
	}

	fb := notarizedBlocks[0]
	if fb.Round == r.GetRoundNumber() {
		return nil
	}
	return fb
}

// FinalizeRoundImpl - starting from the given round work backwards and identify the round that can be assumed to be finalized as all forks after
// that extend from a single block in that round.
func (c *Chain) FinalizeRoundImpl(r round.RoundI) {
	if r.IsFinalized() {
		return // round already finalized
	}
	if r.GetHeaviestNotarizedBlock() == nil {
		logging.Logger.Error("finalize round: no notarized blocks",
			zap.Int64("round", r.GetRoundNumber()))
		go c.GetHeaviestNotarizedBlock(context.Background(), r)
		// r.ResetFinalizingStateIfNotFinalized()
		time.Sleep(FINALIZATION_TIME)
	}

	if r.GetHeaviestNotarizedBlock() == nil {
		logging.Logger.Error("finalize round: no notarized blocks", zap.Int64("round", r.GetRoundNumber()))
		return
	}

	// The SetFinalizing is not condition check it changes round state.
	if !r.SetFinalizing() {
		logging.Logger.Debug("finalize_round: already finalizing",
			zap.Int64("round", r.GetRoundNumber()))
	}

	logging.Logger.Debug("finalize round", zap.Int64("round", r.GetRoundNumber()),
		zap.Int64("lf_round", c.GetLatestFinalizedBlock().Round))
	select {
	case c.finalizedRoundsChannel <- r:
	case <-time.NewTimer(500 * time.Millisecond).C: // TODO: make the timeout configurable
		r.ResetFinalizingStateIfNotFinalized()
		logging.Logger.Info("finalize round - push round to finalizedRoundsChannel timeout",
			zap.Int64("round", r.GetRoundNumber()))
	}
}

// ForceFinalizeRound force trigger the round finalization process
// to avoid sharders stop finalizing blocks. This could happen if
// one block continually timeout on block finalization due to the limited
// cpu resources.
func (c *Chain) ForceFinalizeRound() {
	rn := c.GetCurrentRound()
	r := c.GetRound(rn)
	if r == nil {
		logging.Logger.Error("force finalize round",
			zap.Error(errors.New("can not get current round")),
			zap.Int64("round", rn))
		return
	}

	logging.Logger.Debug("force finalize round", zap.Int64("round", rn))
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
		zap.Int64("plfb_round", plfb.Round),
		zap.Int("num_round_notarized", nbCount),
		zap.Int("num_chain_notarized", len(c.NotarizedBlocksCounts)))
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
		logging.Logger.Debug("finalize round - lfb round is the same as latest lfb",
			zap.Int64("round", roundNumber),
			zap.Int64("lfb round", lfb.Round),
			zap.Int64("plfb round", plfb.Round))
		return
	}

	if lfb.Round > plfb.Round {

		if roundNumber-lfb.Round >= int64(2*config.GetLFBTicketAhead()) {
			logging.Logger.Debug("finalize round - lfb round is behind too much",
				zap.Int64("round", roundNumber),
				zap.Int64("lfb_round", lfb.Round),
				zap.Int64("prev_lfb_round", plfb.Round))
			return
		}

		// maxBackDepth is the maximum number of blocks that we will fetch
		// back from new computed lfb to previous lfb.
		maxBackDepth := config.GetLFBTicketAhead()
		frchain := make([]*block.Block, 0, maxBackDepth)
		for b := lfb; b != nil && b.Hash != plfb.Hash && b.Round > plfb.Round; {
			frchain = append(frchain, b)
			if b.PrevBlock == nil {
				if node.Self.IsSharder() {
					pb := c.GetLocalPreviousBlock(ctx, b)
					if pb == nil {
						logging.Logger.Error("finalize round - previous block is missing",
							zap.Int64("round", b.Round), zap.Int64("prev_lfb", plfb.Round))
						return
					}
					b.SetPreviousBlock(pb)
				} else {
					pb := c.GetPreviousBlock(ctx, b)
					if pb == nil {
						// break to start finalizing blocks in frchain slice
						if len(frchain) >= maxBackDepth {
							break
						}

						// return and retry in next term
						logging.Logger.Debug("finalize round - could not reach to lfb, get previous block failed",
							zap.Int64("round", b.Round),
							zap.Int64("prev round", b.Round-1),
							zap.Int64("prev_lfb", plfb.Round),
							zap.String("block", b.Hash))
						return
					}
				}
			}

			b = b.PrevBlock
			if b.Round == plfb.Round && b.Hash != plfb.Hash {
				logging.Logger.Error("finalize round, computed lfb could not connect to prev lfb",
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.String("prev lfb block", plfb.Hash),
					zap.Int64("computed lfb", lfb.Round),
					zap.String("computed lfb block", lfb.Hash))
				return
			}

			if len(frchain) >= maxBackDepth {
				break
			}
		}

		fb := frchain[len(frchain)-1]
		if fb.Round-1 > plfb.Round {
			logging.Logger.Error("finalize round (missed blocks)",
				zap.Int64("round", roundNumber),
				zap.Int64("from", plfb.Round+1),
				zap.Int64("to", fb.Round-1))
		}

		FinalizationLagMetric.Update(int64(c.GetCurrentRound() - lfb.Round))

		logging.Logger.Info("finalize round - latest finalized round",
			zap.Int64("round", roundNumber),
			zap.Int64("lfb round", lfb.Round),
			zap.String("lfb block", lfb.Hash))
		for idx := range frchain {
			fb := frchain[len(frchain)-1-idx]
			if roundNumber-fb.Round < 3 {
				// finalize the block only when it has at least 3 confirmation
				logging.Logger.Debug("finalize round - block has less than 3 confirmation")
				continue
			}

			if pb := c.GetLocalPreviousBlock(ctx, fb); pb == nil {
				logging.Logger.Error("finalize round - get previous block failed",
					zap.Int64("round", fb.Round))
				return
			}

			if !fb.IsBlockNotarized() {
				nfb, err := c.GetNotarizedBlock(ctx, fb.Hash, fb.Round)
				if err != nil {
					logging.Logger.Error("finalize round - get notarized block failed",
						zap.Int64("round", fb.Round),
						zap.String("block", fb.Hash),
						zap.Error(err))
					return
				}
				fb = nfb
				frchain[len(frchain)-1-idx] = fb
			}

			//_, _ = c.createRoundIfNotExist(ctx, fb)

			logging.Logger.Info("finalize round",
				zap.Int64("round", fb.Round),
				zap.Int64("lfb round", lfb.Round),
				zap.String("block", fb.Hash))

			fbWithReply := &finalizeBlockWithReply{
				block:   fb,
				resultC: make(chan error, 1),
			}

			rd := c.GetRound(fb.Round)
			if rd == nil {
				logging.Logger.Debug("finalize round - fb with round not exist",
					zap.Int64("round", fb.Round), zap.String("block", fb.Hash))
				return
			}
			select {
			case <-ctx.Done():
				logging.Logger.Info("finalize round - context done",
					zap.Error(ctx.Err()),
					zap.Int64("round", roundNumber))
				rd.ResetFinalizingStateIfNotFinalized()
				return
			case c.finalizedBlocksChannel <- fbWithReply:
				ts := time.Now()
				select {
				case <-ctx.Done():
					logging.Logger.Error("finalize round - context done",
						zap.Error(ctx.Err()),
						zap.Int64("round", roundNumber))
					rd.ResetFinalizingStateIfNotFinalized()
					return
				case err := <-fbWithReply.resultC:
					if err != nil {
						logging.Logger.Error("finalize round - finalize block failed",
							zap.Int64("round", fb.Round),
							zap.String("block", fb.Hash),
							zap.Error(err))
						rd.ResetFinalizingStateIfNotFinalized()
						return
					}

					du := time.Since(ts)
					logging.Logger.Info("finalize round - finalize block success",
						zap.Int64("round", fb.Round),
						zap.String("block", fb.Hash),
						zap.Duration("duration", du))

					if du > 3*time.Second {
						logging.Logger.Debug("finalize round slow",
							zap.Int64("round", roundNumber),
							zap.Duration("duration", time.Since(ts)))
					}
				}
			case <-time.NewTimer(500 * time.Millisecond).C: // TODO: make the timeout configurable
				logging.Logger.Error("finalize round - push fb to finalizedBlocksChannel timeout",
					zap.Int64("round", roundNumber),
					zap.Int64("fb_round", fb.Round))
				return
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
		c.SetLatestOwnFinalizedBlockRound(b.Round)
		c.SetLatestFinalizedBlock(b)
		return
	}
	logging.Logger.Error("finalize round - missing common ancestor", zap.Int64("cf_round", plfb.Round), zap.String("cf_block", plfb.Hash), zap.Int64("nf_round", lfb.Round), zap.String("nf_block", lfb.Hash))
}

type finalizeBlockWithReply struct {
	block   *block.Block
	resultC chan error
}

func (c *Chain) createRoundIfNotExist(ctx context.Context, b *block.Block) (round.RoundI, *block.Block) {
	if r := c.GetRound(b.Round); r != nil {
		b, r = c.AddNotarizedBlockToRound(r, b)
		return r, b
	}

	// create the round if it does not exist
	r := c.RoundF.CreateRoundF(b.Round)
	b, r = c.AddNotarizedBlockToRound(r, b)

	// Add the round if chain does not have it
	r = c.AddRound(r)
	return r, b
}

// GetHeaviestNotarizedBlock - get a notarized block for a round.
func (c *Chain) GetHeaviestNotarizedBlock(ctx context.Context, r round.RoundI) *block.Block {

	rn := r.GetRoundNumber()
	nb, err := c.GetNotarizedBlockFromMiners(ctx, "", rn, true)
	if err != nil {
		return nil
	}

	if nb.RoundTimeoutCount != r.GetTimeoutCount() {
		logging.Logger.Info("Timeout count on Round and NB are out-of-sync",
			zap.Int64("round", rn),
			zap.Int("nb_toc", nb.RoundTimeoutCount),
			zap.Int("round_toc", r.GetTimeoutCount()))
	}

	logging.Logger.Debug("get notarized block, add block to round",
		zap.Int64("round", rn),
		zap.String("block", nb.Hash))
	// This is a notarized block. So, use this method to sync round info with the notarized block.
	c.AddNotarizedBlockToRound(r, nb)

	return r.GetHeaviestNotarizedBlock()
}

// GetLatestFinalizedMagicBlockFromShardersOn - request for latest finalized
// magic blocks from all the sharders. It uses provided MagicBlock to get list
// of sharders to request data from, and returns the block with highest magic
// block starting round.
func (c *Chain) GetLatestFinalizedMagicBlockFromShardersOn(ctx context.Context, mb *block.MagicBlock) *block.Block {
	if mb == nil {
		return nil
	}

	var (
		sharders  = mb.Sharders
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

	lfmb := c.GetLatestFinalizedMagicBlock(ctx)
	var params *url.Values
	if lfmb != nil {
		params = &url.Values{}
		params.Add("node-lfmb-hash", lfmb.Hash)
	}

	sharders.RequestEntityFromAll(ctx, LatestFinalizedMagicBlockRequestor, params, handler)

	if len(magicBlocks) == 0 && len(errs) > 0 {
		logging.Logger.Error("Get latest finalized magic block from sharders failed", zap.Errors("errors", errs))
	}

	if len(magicBlocks) == 0 {
		// When sharders return 304 Not Modified code, this magicBlocks will be empty,
		// return the local lfmb if it's empty is a workaround, we should have
		// specific code to indicate the 304 Not Modified response here.
		if lfmb != nil {
			return lfmb
		}

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
	magicBlock := c.getLatestFinalizedMagicBlock(ctx)
	if magicBlock == nil {
		return nil
	}
	return c.GetLatestFinalizedMagicBlockFromShardersOn(ctx, magicBlock)
}

// GetLatestFinalizedMagicBlockRound returns LFMB for given round number
func (c *Chain) GetLatestFinalizedMagicBlockRound(rn int64) *block.Block {
	rn = mbRoundOffset(rn) // round number with mb offset
	c.lfmbMutex.RLock()
	if len(c.magicBlockStartingRoundsMap) > 0 {
		r := c.magicBlockStartingRounds.Prev()
		ringSize := c.magicBlockStartingRounds.Len()
		for i := 0; i < ringSize; i++ {
			startRound := r.Value.(int64)
			r = r.Prev()
			if startRound <= rn {
				b := c.magicBlockStartingRoundsMap[startRound]
				c.lfmbMutex.RUnlock()
				return b
			}
		}
	}
	c.lfmbMutex.RUnlock()
	lfmb := c.GetLatestFinalizedMagicBlock(common.GetRootContext())
	logging.Logger.Warn("GetLatestFinalizedMagicBlockRound: no magic block found for round, use the latest one",
		zap.Int64("round", rn),
		zap.Int64("lfmb starting round", lfmb.StartingRound),
		zap.Int64("mb number", lfmb.MagicBlockNumber),
		zap.String("mb hash", lfmb.MagicBlock.Hash))
	return lfmb
}

func getMagicBlockBrief(b *block.Block) *MagicBlockBrief {
	if b.MagicBlock == nil {
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
