package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/logging"
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
			logging.Logger.Error("verify notarization - null ticket",
				zap.String("block", b.Hash))
			return common.NewError("null_ticket", "Verification ticket is null")
		}
		if _, ok := ticketsMap[vt.VerifierID]; ok {
			return common.NewError("duplicate_ticket_signature",
				"Found duplicate signatures in the notarization of the block")
		}
		ticketsMap[vt.VerifierID] = true
	}

	if !c.reachedNotarization(round, b.Hash, bvt) {
		return common.NewError("block_not_notarized",
			"Verification tickets not sufficient to reach notarization")
	}

	for _, vt := range bvt {
		if err := c.VerifyTicket(ctx, b.Hash, vt, round); err != nil {
			return err
		}
	}

	b.SetBlockNotarized()

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
		logging.Logger.Error("is_block_notarized", zap.Error(err))
		return false // false
	}

	var notarized = c.reachedNotarization(b.Round, b.Hash, b.GetVerificationTickets())
	if notarized {
		b.SetBlockNotarized()
	}
	return notarized
}

func (c *Chain) reachedNotarization(round int64, hash string,
	bvt []*block.VerificationTicket) bool {

	var (
		mb        = c.GetMagicBlock(round)
		num       = mb.Miners.Size()
		threshold = c.GetNotarizationThresholdCount(num)
	)

	if c.ThresholdByCount > 0 {
		var numSignatures = len(bvt)
		if numSignatures < threshold {
			logging.Logger.Info("not reached notarization",
				zap.Int64("mb_sr", mb.StartingRound),
				zap.Int("active_miners", num),
				zap.Int("threshold", threshold),
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
			logging.Logger.Info("not reached notarization - stake < threshold stake",
				zap.Int64("mb_sr", mb.StartingRound),
				zap.Int("verify stake", verifiersStake),
				zap.Int("threshold", c.ThresholdByStake),
				zap.Int("active_miners", num),
				zap.Int("num_signatures", len(bvt)),
				zap.Int("signature threshold", threshold),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int64("round", round))
			return false
		}
	}

	logging.Logger.Info("reached notarization!!!",
		zap.Int64("round", round),
		zap.Int64("current_round", c.GetCurrentRound()),
		zap.String("block", hash),
		zap.Int64("mb_sr", mb.StartingRound),
		zap.Int("active_miners", num),
		zap.Int("num_signatures", len(bvt)),
		zap.Int("threshold", threshold))

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
		logging.Logger.Error("UpdateNodeState: round unexpected nil")
		return
	}
	for _, vt := range b.GetVerificationTickets() {
		miners := c.GetMiners(r.GetRoundNumber())
		if miners == nil {
			logging.Logger.Error("UpdateNodeState: miners unexpected nil")
			continue
		}
		signer := miners.GetNode(vt.VerifierID)
		if signer == nil {
			logging.Logger.Error("this should not happen!")
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
	logging.Logger.Info("finalize block", zap.Int64("round", fb.Round), zap.Int64("current_round", c.GetCurrentRound()),
		zap.Int64("lf_round", c.GetLatestFinalizedBlock().Round), zap.String("hash", fb.Hash),
		zap.Int("round_rank", fb.RoundRank), zap.Int8("state", fb.GetBlockState()))
	numGenerators := c.GetGeneratorsNum()
	if fb.RoundRank >= numGenerators || fb.RoundRank < 0 {
		logging.Logger.Warn("finalize block - round rank is invalid or greater than num_generators",
			zap.Int("round_rank", fb.RoundRank),
			zap.Int("num_generators", numGenerators))
	} else {
		var bNode = node.GetNode(fb.MinerID)
		if bNode != nil {
			if bNode.ProtocolStats != nil {
				//FIXME: fix node stats
				ms := bNode.ProtocolStats.(*MinerStats)
				if numGenerators > len(ms.FinalizationCountByRank) {
					newRankStat := make([]int64, numGenerators)
					copy(newRankStat, ms.FinalizationCountByRank)
					ms.FinalizationCountByRank = newRankStat
				}
				ms.FinalizationCountByRank[fb.RoundRank]++ // stat
			}
		} else {
			logging.Logger.Error("generator is not registered",
				zap.Int64("round", fb.Round),
				zap.String("miner", fb.MinerID))
		}
	}
	fr := c.GetRound(fb.Round)

	logging.Logger.Info("finalize block -- round", zap.Any("round", fr))

	if fr != nil {
		generators := c.GetGenerators(fr)
		for idx, g := range generators {
			ms := g.ProtocolStats.(*MinerStats)
			if len(generators) > len(ms.GenerationCountByRank) {
				newRankStat := make([]int64, len(generators))
				copy(newRankStat, ms.GenerationCountByRank)
				ms.GenerationCountByRank = newRankStat
			}
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
		logging.Logger.Error("finalize block save changes failed",
			zap.Error(err),
			zap.Int64("round", fb.Round),
			zap.String("hash", fb.Hash))
		return
	}
	c.rebaseState(fb)
	c.updateFeeStats(fb)

	c.SetLatestOwnFinalizedBlockRound(fb.Round)
	c.SetLatestFinalizedBlock(fb)

	if fb.MagicBlock != nil {
		c.UpdateMagicBlock(fb.MagicBlock)
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

// GetPreviousBlock gets or sync the previous block from the network and compute its state.
func (c *Chain) GetPreviousBlock(ctx context.Context, b *block.Block) *block.Block {
	// check if the previous block points to itself
	if b.PrevBlock == b || b.PrevHash == b.Hash {
		logging.Logger.DPanic("block->PrevBlock points to itself",
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash),
			zap.String("prev_hash", b.PrevHash))
	}

	if b.PrevBlock != nil && b.PrevBlock.Hash == b.PrevHash && b.PrevBlock.IsStateComputed() {
		return b.PrevBlock
	}

	pb, _ := c.GetBlock(ctx, b.PrevHash)
	if pb != nil && pb.IsStateComputed() {
		b.SetPreviousBlock(pb)
		return pb
	}

	lfb := c.GetLatestFinalizedBlock()
	if lfb != nil && lfb.Round == b.Round-1 && lfb.IsStateComputed() {
		// previous round is latest finalized round
		b.SetPreviousBlock(lfb)
		logging.Logger.Info("get_previous_block - previous block is lfb",
			zap.Int64("round", b.Round),
			zap.Int64("lfb_round", lfb.Round),
			zap.String("block", b.Hash))
		return lfb
	}

	// TODO: make this configurable
	const maxSyncDepth int64 = 50
	syncNum := maxSyncDepth
	if lfb != nil {
		syncNum = b.Round - lfb.Round
	}

	if syncNum <= 0 || syncNum > maxSyncDepth {
		syncNum = maxSyncDepth
	}

	blocks := c.SyncBlocks(ctx, b, syncNum, false)
	if len(blocks) == 0 || !blocks[0].IsStateComputed() {
		logging.Logger.Debug("get_previous_block - could not sync previous blocks",
			zap.Int64("round", b.Round-1), zap.Int64("sync_num", syncNum))
		return nil
	}

	first := blocks[0]
	last := blocks[len(blocks)-1]
	if lfb != nil {
		if first.Round <= lfb.Round && last.Round >= lfb.Round {
			syncedLfb := blocks[lfb.Round-first.Round]
			if syncedLfb.Hash != lfb.Hash {
				logging.Logger.Error("get_previous_block - sync blocks got different lfb blocks",
					zap.Int64("round", b.Round),
					zap.Int64("lfb", lfb.Round),
					zap.Int64("synced_lfb_round", syncedLfb.Round),
					zap.String("lfb_block", lfb.Hash),
					zap.String("synced_lfb_block", syncedLfb.Hash))
			}
		}
	}

	pb = last
	if !pb.IsStateComputed() {
		logging.Logger.Error("get_previous_block - could not get state computed previous block",
			zap.Int64("round", b.Round),
			zap.Int64("previous_round", pb.Round),
			zap.String("previous_block", pb.Hash))
		return nil
	}

	b.SetPreviousBlock(pb)

	logging.Logger.Info("get_previous_block - sync successfully",
		zap.Int64("round", b.Round),
		zap.Int64("previous round", b.PrevBlock.Round),
		zap.String("previous block", b.PrevHash),
		zap.Int("synced_num", len(blocks)))
	return pb
}

// SyncBlocks sync N blocks and state changes from network
func (c *Chain) SyncBlocks(ctx context.Context, b *block.Block, num int64, saveToDB bool) []*block.Block {
	blocks := c.pullNotarizedBlocks(ctx, b, num)
	if len(blocks) == 0 {
		logging.Logger.Debug("sync_blocks - pull blocks with no response")
		return nil
	}

	failedIndex := -1
	for i := range blocks {
		cb := blocks[i]
		if cb.PrevBlock == nil {
			// continue so that next block has previous block
			failedIndex = i
			continue
		}

		cb.SetStateDB(cb.PrevBlock, c.GetStateDB())

		if err := c.GetBlockStateChange(cb); err != nil {
			logging.Logger.Error("sync_blocks - sync state change failed",
				zap.Error(err), zap.Int64("round", cb.Round))

			if err := cb.ComputeStateLocal(ctx, c); err != nil {
				logging.Logger.Error("sync_blocks - compute state force failed",
					zap.Error(err), zap.Int64("round", cb.Round))
				// continue as later blocks may be able to get state changes from remote or compute state successfully
				failedIndex = i
				continue
			}
		}

		if saveToDB {
			if err := cb.SaveChanges(ctx, c); err != nil {
				logging.Logger.Error("sync_blocks - save changes failed",
					zap.Error(err), zap.Int64("round", cb.Round))
			}
			logging.Logger.Info("sync_blocks - save state changes success",
				zap.Int64("round", cb.Round),
				zap.String("block", cb.Hash))
		}

		fmt.Println("cb round", cb.Round, ", index:", i, ", len:", len(blocks))
		logging.Logger.Info("sync_blocks success", zap.Int64("round", cb.Round),
			zap.String("block", cb.Hash))
	}

	blocks = blocks[failedIndex+1:]
	if len(blocks) > 0 {
		c.UpdateBlocks(blocks)
	}

	return blocks
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
