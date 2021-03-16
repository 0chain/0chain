package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*VerifyTicket - verify the ticket */
func (c *Chain) VerifyTicket(ctx context.Context, blockHash string,
	bvt *block.VerificationTicket, round int64) error {

	var sender = c.GetMiners(round).GetNode(bvt.VerifierID)
	if sender == nil {
		return common.InvalidRequest(fmt.Sprintf("Verifier unknown or not authorized at this time: %v", bvt.VerifierID))
	}

	if ok, _ := sender.Verify(bvt.Signature, blockHash); !ok {
		return common.InvalidRequest("Couldn't verify the signature")
	}
	return nil
}

// VerifyNotarization - verify that the notarization is correct.
func (c *Chain) VerifyNotarization(ctx context.Context, b *block.Block,
	bvt []*block.VerificationTicket, round int64) (err error) {

	if bvt == nil {
		return common.NewError("no_verification_tickets",
			"No verification tickets for this block")
	}

	if err = c.VerifyRelatedMagicBlockPresence(b); err != nil {
		return
	}

	var ticketsMap = make(map[string]bool, len(bvt))
	for _, vt := range bvt {
		if vt == nil {
			Logger.Error("verify notarization - null ticket",
				zap.String("block", b.Hash))
			return common.NewError("null_ticket", "Verification ticket is null")
		}
		if _, ok := ticketsMap[vt.VerifierID]; ok {
			return common.NewError("duplicate_ticket_signature",
				"Found duplicate signatures in the notarization of the block")
		}
		ticketsMap[vt.VerifierID] = true
	}

	if !c.reachedNotarization(round, bvt) {
		return common.NewError("block_not_notarized",
			"Verification tickets not sufficient to reach notarization")
	}

	for _, vt := range bvt {
		if err := c.VerifyTicket(ctx, b.Hash, vt, round); err != nil {
			return err
		}
	}

	return nil
}

// VerifyRelatedMagicBlockPresence check is there related magic block and
// returns detailed error or nil for successful case. Since GetMagicBlock
// is optimistic it can returns different magic block for requested round.
func (c *Chain) VerifyRelatedMagicBlockPresence(b *block.Block) (err error) {

	// return // force ok to check

	var (
		lfb        = c.GetLatestFinalizedBlock()
		relatedmbr = b.LatestFinalizedMagicBlockRound
		mb         = c.GetMagicBlock(b.Round)
	)

	if mb.StartingRound != relatedmbr {
		return common.NewErrorf("verify_related_mb_presence",
			"no corresponding MB, want_mb_sr: %d, got_mb_sr: %d",
			relatedmbr, mb.StartingRound)
	}

	if b.Round < lfb.Round {
		return // don't verify for blocks before LFB
	}

	// we can't check MB hash here, because we got magic block, but hash is
	// hash of block with the magic block

	return // ok, there is
}

// IsBlockNotarized - check if the block is notarized.
func (c *Chain) IsBlockNotarized(ctx context.Context, b *block.Block) bool {
	if b.IsBlockNotarized() {
		return true
	}

	if err := c.VerifyRelatedMagicBlockPresence(b); err != nil {
		Logger.Error("is_block_notarized", zap.Error(err))
		return false // false
	}

	var notarized = c.reachedNotarization(b.Round, b.GetVerificationTickets())
	if notarized {
		b.SetBlockNotarized()
	}
	return notarized
}

func (c *Chain) reachedNotarization(round int64,
	bvt []*block.VerificationTicket) bool {

	var (
		mb  = c.GetMagicBlock(round)
		num = mb.Miners.Size()
	)

	if c.ThresholdByCount > 0 {
		var numSignatures = len(bvt)
		if numSignatures < c.GetNotarizationThresholdCount(num) {
			//ToDo: Remove this comment
			Logger.Info("not reached notarization",
				zap.Int64("mb_sr", mb.StartingRound),
				zap.Int("miners", num),
				zap.Int("threshold", c.GetNotarizationThresholdCount(num)),
				zap.Int("num_signatures", numSignatures),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int64("round", round))
			return false
		}
	}
	if c.ThresholdByStake > 0 {
		verifiersStake := 0
		for _, ticket := range bvt {
			verifiersStake += c.getMiningStake(ticket.VerifierID)
		}
		if verifiersStake < c.ThresholdByStake {
			return false
		}
	}

	Logger.Info("Reached notarization!!!",
		zap.Int64("mb_sr", mb.StartingRound),
		zap.Int("miners", num),
		zap.Int64("round", round),
		zap.Int64("current_cound", c.GetCurrentRound()),
		zap.Int("num_signatures", len(bvt)),
		zap.Int("threshold", c.GetNotarizationThresholdCount(num)))

	return true
}

/*UpdateNodeState - based on the incoming valid blocks, update the nodes that notarized the block to be active
 Useful to increase the speed of node status discovery which increases the reliablity of the network
Simple 3 miner scenario :

1) a discovered b & c.
2) b discovered a.
3) b and c are yet to discover each other
4) a generated a block and sent it to b & c, got it notarized and next round started
5) c is the generator who generated the block. He will only send it to a as b is not discovered to be active.
    But if the prior block has b's signature (may or may not, but if it did), c can discover b is active before generating the block and so will send it to b
*/
func (c *Chain) UpdateNodeState(b *block.Block) {
	r := c.GetRound(b.Round)
	if r == nil {
		Logger.Error("UpdateNodeState: round unexpected nil")
		return
	}
	for _, vt := range b.GetVerificationTickets() {
		miners := c.GetMiners(r.GetRoundNumber())
		if miners == nil {
			Logger.Error("UpdateNodeState: miners unexpected nil")
			continue
		}
		signer := miners.GetNode(vt.VerifierID)
		if signer == nil {
			Logger.Error("this should not happen!")
			continue
		}
		if signer.GetStatus() != node.NodeStatusActive {
			signer.SetStatus(node.NodeStatusActive)
		}
	}
}

/*AddVerificationTicket - add a verified ticket to the list of verification tickets of the block */
func (c *Chain) AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool {
	added := b.AddVerificationTicket(bvt)
	if added {
		c.IsBlockNotarized(ctx, b)
	}
	return added
}

/*MergeVerificationTickets - merge a set of verification tickets (already validated) for a given block */
func (c *Chain) MergeVerificationTickets(ctx context.Context, b *block.Block, vts []*block.VerificationTicket) {
	vtlen := b.VerificationTicketsSize()
	b.MergeVerificationTickets(vts)
	if b.VerificationTicketsSize() != vtlen {
		c.IsBlockNotarized(ctx, b)
	}
}

func (c *Chain) finalizeBlock(ctx context.Context, fb *block.Block, bsh BlockStateHandler) {
	Logger.Info("finalize block", zap.Int64("round", fb.Round), zap.Int64("current_round", c.GetCurrentRound()),
		zap.Int64("lf_round", c.GetLatestFinalizedBlock().Round), zap.String("hash", fb.Hash),
		zap.Int("round_rank", fb.RoundRank), zap.Int8("state", fb.GetBlockState()))
	if fb.RoundRank >= c.NumGenerators || fb.RoundRank < 0 {
		Logger.Warn("FB round rank is invalid or greater than num_generators",
			zap.Int("round_rank", fb.RoundRank),
			zap.Int("num_generators", c.NumGenerators))
	} else {
		var bNode = node.GetNode(fb.MinerID)
		if bNode != nil {
			if bNode.ProtocolStats != nil {
				//FIXME: fix node stats
				ms := bNode.ProtocolStats.(*MinerStats)
				ms.FinalizationCountByRank[fb.RoundRank]++ // stat
			}
		} else {
			Logger.Error("generator is not registered",
				zap.Int64("round", fb.Round),
				zap.String("miner", fb.MinerID))
		}
	}
	fr := c.GetRound(fb.Round)
	Logger.Info("finalize block -- round", zap.Any("round", fr))
	if fr != nil {
		generators := c.GetGenerators(fr)
		for idx, g := range generators {
			ms := g.ProtocolStats.(*MinerStats)
			ms.GenerationCountByRank[idx]++
		}
	}
	if time.Since(ssFTs) < 20*time.Second {
		SteadyStateFinalizationTimer.UpdateSince(ssFTs)
	}
	if time.Since(fb.ToTime()) < 100*time.Second {
		StartToFinalizeTimer.UpdateSince(fb.ToTime())
	}

	ssFTs = time.Now()
	c.UpdateChainInfo(fb)
	if err := c.SaveChanges(ctx, fb); err != nil {
		Logger.Error("Finaliz block save changes failed",
			zap.Error(err),
			zap.Int64("round", fb.Round),
			zap.String("hash", fb.Hash))
		return
	}
	c.rebaseState(fb)
	c.updateFeeStats(fb)

	if fb.MagicBlock != nil {
		c.SetLatestFinalizedMagicBlock(fb)
	}
	if config.Development() {
		ts := time.Now()
		for _, txn := range fb.Txns {
			StartToFinalizeTxnTimer.Update(ts.Sub(common.ToTime(txn.CreationDate)))
		}
	}
	go bsh.UpdateFinalizedBlock(ctx, fb)
	c.BlockChain.Value = fb.GetSummary()
	c.BlockChain = c.BlockChain.Next()

	for pfb := fb; pfb != nil && pfb != c.LatestDeterministicBlock; pfb = pfb.PrevBlock {
		if c.IsFinalizedDeterministically(pfb) {
			c.SetLatestDeterministicBlock(pfb)
			break
		}
	}

	// Deleting dead blocks from a couple of rounds before (helpful for visualizer and potential rollback scenrio)
	pfb := fb
	for idx := 0; idx < 10 && pfb != nil; idx, pfb = idx+1, pfb.PrevBlock {

	}
	if pfb == nil {
		return
	}
	frb := c.GetRoundBlocks(pfb.Round)
	var deadBlocks []*block.Block
	for _, b := range frb {
		if b.Hash != pfb.Hash {
			deadBlocks = append(deadBlocks, b)
		}
	}
	// Prune all the dead blocks
	c.DeleteBlocks(deadBlocks)
}

//IsFinalizedDeterministically - checks if a block is finalized deterministically
func (c *Chain) IsFinalizedDeterministically(b *block.Block) bool {
	//TODO: The threshold count should happen w.r.t the view of the block
	mb := c.GetMagicBlock(b.Round)
	if c.GetLatestFinalizedBlock().Round < b.Round {
		return false
	}
	if len(b.UniqueBlockExtensions)*100 >= mb.Miners.Size()*c.ThresholdByCount {
		return true
	}
	return false
}

// GetLocalPreviousBlock returns previous block for the block. Without a network
// request. And without a storage lookup.
func (c *Chain) GetLocalPreviousBlock(ctx context.Context, b *block.Block) (
	pb *block.Block) {

	if b.PrevBlock != nil {
		return b.PrevBlock
	}
	pb, _ = c.GetBlock(ctx, b.PrevHash)
	return
}

// GetPreviousBlock - get the previous block from the network and compute its state.
// TODO: decouple the block fetching and state computation.
func (c *Chain) GetPreviousBlock(ctx context.Context, b *block.Block) *block.Block {
	// check if the previous block points to itself
	if b.PrevBlock == b || b.PrevHash == b.Hash {
		Logger.DPanic("block->PrevBlock points to itself",
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash),
			zap.String("prev_hash", b.PrevHash))
	}

	if b.PrevBlock != nil {
		return b.PrevBlock
	}

	pb, err := c.GetBlock(ctx, b.PrevHash)
	if err == nil && pb.IsStateComputed() {
		b.SetPreviousBlock(pb)
		return pb
	}

	blocks := make([]*block.Block, 0, 10)
	Logger.Info("fetch previous block", zap.Int64("round", b.Round),
		zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))

	cb := b
	for idx := 0; idx < 10; idx++ {
		Logger.Debug("fetching previous block", zap.Int("idx", idx),
			zap.Int64("cround", cb.Round), zap.String("cblock", cb.Hash),
			zap.String("cprev_block", cb.PrevHash))

		nb := c.GetNotarizedBlock(ctx, cb.PrevHash, cb.Round-1)
		if nb == nil {
			Logger.Error("get previous block (unable to get prior blocks)",
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int("idx", idx), zap.Int64("round", b.Round),
				zap.String("block", b.Hash), zap.Int64("cround", cb.Round),
				zap.String("cblock", cb.Hash),
				zap.String("cprev_block", cb.PrevHash))
			return nil
		}

		cb = nb
		blocks = append(blocks, cb)
		pb, err = c.GetBlock(ctx, cb.PrevHash)
		if pb != nil {
			cb.SetPreviousBlock(pb)
			break
		}
	}

	// This happens after fetching as far as per the previous for loop and
	// still not having the prior block.
	if cb.PrevBlock == nil {
		Logger.Error("get previous block (missing continuity)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.Int64("oldest_fetched_round", cb.Round),
			zap.String("oldest_fetched_block", cb.Hash),
			zap.String("missing_prior_block", cb.PrevHash))
		return nil
	}

	for idx := len(blocks) - 1; idx >= 0; idx-- {
		cb := blocks[idx]
		if cb.PrevBlock == nil {
			pb, err := c.GetBlock(ctx, cb.PrevHash)
			if err != nil {
				Logger.Error("get previous block (missing continuity)",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.Int64("cb_round", cb.Round),
					zap.String("cb_block", cb.Hash),
					zap.String("missing_prior_block", cb.PrevHash))
				return nil
			}
			cb.SetPreviousBlock(pb)
		}
		// TODO (sfxdx): complex deadlock is here
		c.ComputeState(ctx, cb)
	}

	pb, err = c.GetBlock(ctx, b.PrevHash)
	if err == nil {
		b.SetPreviousBlock(pb)
	}

	return pb
}

// fetchPreviousBlock fetches a previous block from network
func (c *Chain) fetchPreviousBlock(ctx context.Context, b *block.Block) *block.Block {
	Logger.Info("fetch previous block", zap.Int64("round", b.Round),
		zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))

	pb := c.GetNotarizedBlock(ctx, b.PrevHash, b.Round-1)
	if pb == nil {
		Logger.Error("get previous block (unable to get prior blocks)",
			zap.Int64("current_round", c.GetCurrentRound()),
			zap.Int64("round", b.Round),
			zap.Int64("prev_round", b.Round),
			zap.String("block", b.Hash),
			zap.String("prev_block", b.PrevHash))
		return nil
	}

	// get the previous block from local blocks again, the GetNotarizedBlock() function
	// called above should have updated the chain's blocks in memory.
	pb, err := c.GetBlock(ctx, b.PrevHash)
	if err != nil {
		Logger.DPanic("get previous notarized block failed", zap.Error(err))
	}

	b.SetPreviousBlock(pb)

	// set the pre pre block if it could be acquired from local blocks
	ppb, err := c.GetBlock(ctx, pb.PrevHash)
	if err == nil {
		pb.SetPreviousBlock(ppb)
	}

	return pb
}

//Note: this is expected to work only for small forks
func (c *Chain) commonAncestor(ctx context.Context, b1 *block.Block, b2 *block.Block) *block.Block {
	if b1 == nil || b2 == nil {
		return nil
	}
	if b1 == b2 || b1.Hash == b2.Hash {
		return b1
	}
	if b2.Round < b1.Round {
		b1, b2 = b2, b1
	}
	for b2.Round != b1.Round {
		b2 = c.GetPreviousBlock(ctx, b2)
		if b2 == nil {
			return nil
		}
	}
	for b1 != b2 {
		b1 = c.GetPreviousBlock(ctx, b1)
		if b1 == nil {
			return nil
		}
		b2 = c.GetPreviousBlock(ctx, b2)
		if b2 == nil {
			return nil
		}
	}
	return b1
}

func (c *Chain) updateFeeStats(fb *block.Block) {
	var totalFees int64
	for _, txn := range fb.Txns {
		totalFees += txn.Fee
	}
	meanFees := totalFees / int64(len(fb.Txns))
	c.FeeStats.MeanFees = meanFees
	if meanFees > c.FeeStats.MaxFees {
		c.FeeStats.MaxFees = meanFees
	}
	if meanFees < c.FeeStats.MinFees {
		c.FeeStats.MinFees = meanFees
	}
}
