package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/core/datastore"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/encryption"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

// VerifyTickets verifies tickets aggregately
// Note: this only works for BLS scheme keys
func (c *Chain) VerifyTickets(ctx context.Context, blockHash string, bvts []*block.VerificationTicket, round int64) error {
	return c.verifyTicketsWithContext.Run(ctx, func() error {
		aggScheme := encryption.GetAggregateSignatureScheme(c.ClientSignatureScheme(),
			len(bvts), len(bvts))
		if aggScheme == nil {
			// TODO: do ticket verification one by one when aggregate signature
			// does not exist
			panic(fmt.Sprintf("signature scheme not implemented: %v", c.ClientSignatureScheme()))
		}

		doneC := make(chan struct{})
		errC := make(chan error)
		go func() {
			for i, bvt := range bvts {
				pl := c.GetMiners(round)
				verifier := pl.GetNode(bvt.VerifierID)
				if verifier == nil {
					errC <- common.InvalidRequest(fmt.Sprintf("Verifier unknown or not authorized at this time: %v, pool size: %d", bvt.VerifierID, pl.Size()))
					return
				}

				if verifier.SigScheme == nil {
					errC <- common.NewErrorf("verify_tickets", "node has no signature scheme")
					return
				}

				if err := aggScheme.Aggregate(verifier.SigScheme, i, bvt.Signature, blockHash); err != nil {
					errC <- common.NewError("verify_tickets", err.Error())
					return
				}
			}

			if _, err := aggScheme.Verify(); err != nil {
				errC <- common.NewErrorf("verify_tickets", "failed to verify aggregate signatures: %v", err)
				return
			}

			close(doneC)
		}()

		select {
		case <-doneC:
			return nil
		case err := <-errC:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (c *Chain) VerifyBlockNotarization(ctx context.Context, b *block.Block) error {
	if err := c.VerifyNotarization(ctx, b.Hash, b.GetVerificationTickets(), b.Round); err != nil {
		return err
	}

	if err := c.VerifyRelatedMagicBlockPresence(b); err != nil {
		return err
	}

	return nil
}

// VerifyNotarization - verify that the notarization is correct.
func (c *Chain) VerifyNotarization(ctx context.Context, hash datastore.Key,
	bvt []*block.VerificationTicket, round int64) (err error) {

	if bvt == nil {
		return common.NewError("no_verification_tickets",
			"No verification tickets for this block")
	}

	var ticketsMap = make(map[string]bool, len(bvt))
	for _, vt := range bvt {
		if vt == nil {
			logging.Logger.Error("verify notarization - null ticket",
				zap.String("block", hash))
			return common.NewError("null_ticket", "Verification ticket is null")
		}
		if _, ok := ticketsMap[vt.VerifierID]; ok {
			return common.NewError("duplicate_ticket_signature",
				"Found duplicate signatures in the notarization of the block")
		}
		ticketsMap[vt.VerifierID] = true
	}

	if !c.reachedNotarization(round, hash, bvt) {
		return common.NewError("block_not_notarized",
			"Verification tickets not sufficient to reach notarization")
	}

	if err := c.VerifyTickets(ctx, hash, bvt, round); err != nil {
		return err
	}

	logging.Logger.Info("reached notarization - verify notarization",
		zap.Int64("round", round),
		zap.Int64("current_round", c.GetCurrentRound()),
		zap.String("block", hash),
		zap.Int("tickets_num", len(bvt)))

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

// UpdateBlockNotarization updates the block notarization state,
// return true if the block reached notarization
func (c *Chain) UpdateBlockNotarization(b *block.Block) bool {
	if b.IsBlockNotarized() {
		return true
	}

	if err := c.VerifyRelatedMagicBlockPresence(b); err != nil {
		logging.Logger.Error("is_block_notarized", zap.Error(err))
		return false
	}

	if c.reachedNotarization(b.Round, b.Hash, b.GetVerificationTickets()) {
		b.SetBlockNotarized()
		return true
	}

	return false
}

func (c *Chain) reachedNotarization(round int64, hash string,
	bvt []*block.VerificationTicket) bool {

	var (
		mb        = c.GetMagicBlock(round)
		num       = mb.Miners.Size()
		threshold = c.GetNotarizationThresholdCount(num)
	)

	if c.ThresholdByCount() > 0 {
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
	if c.ThresholdByStake() > 0 {
		verifiersStake := 0
		for _, ticket := range bvt {
			verifiersStake += c.getMiningStake(ticket.VerifierID)
		}
		if verifiersStake < c.ThresholdByStake() {
			logging.Logger.Info("not reached notarization - stake < threshold stake",
				zap.Int64("mb_sr", mb.StartingRound),
				zap.Int("verify stake", verifiersStake),
				zap.Int("threshold", c.ThresholdByStake()),
				zap.Int("active_miners", num),
				zap.Int("num_signatures", len(bvt)),
				zap.Int("signature threshold", threshold),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int64("round", round))
			return false
		}
	}

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
func (c *Chain) AddVerificationTicket(b *block.Block, bvt *block.VerificationTicket) bool {
	if b.AddVerificationTicket(bvt) {
		if c.UpdateBlockNotarization(b) {
			logging.Logger.Info("reached notarization - add tickets",
				zap.Int64("round", b.Round),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.String("block", b.Hash),
				zap.Int("tickets_num", len(b.GetVerificationTickets())))
		}
		return true
	}

	return false
}

// MergeVerificationTickets - merge a set of verification tickets (already validated) for a given block */
func (c *Chain) MergeVerificationTickets(b *block.Block, vts []*block.VerificationTicket) {
	b.MergeVerificationTickets(vts)
	if c.UpdateBlockNotarization(b) {
		logging.Logger.Info("reached notarization - merging tickets",
			zap.Int64("round", b.Round),
			zap.Int64("current_round", c.GetCurrentRound()),
			zap.String("block", b.Hash),
			zap.Int("tickets_num", len(b.GetVerificationTickets())))
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
		bNode := c.GetMiners(fb.Round).GetNode(fb.MinerID)
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

	logging.Logger.Info("finalize block -- round", zap.Any("round", fr), zap.String("block", fb.Hash))

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
	if err := c.updateFeeStats(fb); err != nil {
		logging.Logger.Error("finalize block - update fee stats failed",
			zap.Int64("round", fb.Round),
			zap.Int64("mb_starting_round", fb.StartingRound),
			zap.Error(err))
	}

	c.SetLatestOwnFinalizedBlockRound(fb.Round)
	c.SetLatestFinalizedBlock(fb)

	if len(fb.Events) > 0 && c.GetEventDb() != nil {
		go func(events []event.Event) {
			c.GetEventDb().AddEvents(ctx, events)
		}(fb.Events)
		fb.Events = nil
	}

	if fb.MagicBlock != nil {
		if err := c.UpdateMagicBlock(fb.MagicBlock); err != nil {
			logging.Logger.Error("finalize block - update magic block failed",
				zap.Int64("round", fb.Round),
				zap.Int64("mb_starting_round", fb.StartingRound),
				zap.Error(err))
		} else {
			c.SetLatestFinalizedMagicBlock(fb)
		}
	}

	if config.Development() {
		ts := time.Now()
		for _, txn := range fb.Txns {
			StartToFinalizeTxnTimer.Update(ts.Sub(common.ToTime(txn.CreationDate)))
		}
	}
	go bsh.UpdateFinalizedBlock(ctx, fb)

	fr.Finalize(fb)
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
	if len(b.UniqueBlockExtensions)*100 >= mb.Miners.Size()*c.ThresholdByCount() {
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

// GetPreviousBlock gets or sync the previous block from the network and fetches partial state change from the network.
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
		if b.PrevHash != lfb.Hash {
			logging.Logger.Error("get_previous_block - can't set lfb as previous block, hash mismatch")
			return nil
		}
		b.SetPreviousBlock(lfb)
		logging.Logger.Info("get_previous_block - previous block is lfb",
			zap.Int64("round", b.Round),
			zap.Int64("lfb_round", lfb.Round),
			zap.String("block", b.Hash))
		return lfb
	}

	maxSyncDepth := int64(config.GetLFBTicketAhead())
	syncNum := maxSyncDepth
	if lfb != nil {
		syncNum = b.Round - lfb.Round
		// sync lfb if its state is not computed
		if syncNum > 0 && syncNum < maxSyncDepth && !lfb.IsStateComputed() {
			syncNum++
		}

		if syncNum > maxSyncDepth {
			syncNum = maxSyncDepth
		}
	}

	// The round is equal or less than lfb, get state changes
	// of one block previous
	if syncNum <= 0 {
		//blocks := c.SyncPreviousBlocks(ctx, b, 1, false)
		//will load partial state here
		pb = c.SyncPreviousBlocks(ctx, b, 1)
		if pb == nil {
			logging.Logger.Error("get_previous_block - could not fetch block",
				zap.Int64("round", b.Round-1),
				zap.Int64("lfb_round", lfb.Round))
			return nil
		}

		b.SetPreviousBlock(pb)
		logging.Logger.Info("get_previous_block - sync successfully",
			zap.Int("sync num", 1),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("previous round", b.PrevBlock.Round),
			zap.String("previous block", b.PrevHash))
		return pb
	}

	pb = c.SyncPreviousBlocks(ctx, b, syncNum)
	if pb == nil {
		return nil
	}

	if !pb.IsStateComputed() {
		logging.Logger.Error("get_previous_block - could not get state computed previous block",
			zap.Int64("round", b.Round),
			zap.Int64("previous_round", pb.Round),
			zap.String("previous_block", pb.Hash))
		return nil
	}

	if pb.Hash != b.PrevHash {
		logging.Logger.Error("get_previous_block - got previous block with different hash",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.String("block.PrevHash", b.PrevHash),
			zap.String("prev hash", pb.Hash))
		return nil
	}

	b.SetPreviousBlock(pb)

	logging.Logger.Info("get_previous_block - sync successfully",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.Int64("previous round", b.PrevBlock.Round),
		zap.String("previous block", b.PrevHash))

	return pb
}

func (c *Chain) registerBlockSync(blockHash string, replyC chan *block.Block) (notifyAndClean func(*block.Block), ok bool) {
	var ch chan chan *block.Block
	c.bscMutex.Lock()
	ch, ok = c.blockSyncC[blockHash]
	if !ok {
		ch = make(chan chan *block.Block, 50)
		c.blockSyncC[blockHash] = ch
	}

	select {
	case ch <- replyC:
		c.bscMutex.Unlock()
	default:
		c.bscMutex.Unlock()
		logging.Logger.Debug("sync_block - block sync chan is full", zap.String("block", blockHash))
		return func(*block.Block) {}, false
	}

	notifyAndClean = func(b *block.Block) {
		c.bscMutex.Lock()
		close(ch)
		for sub := range ch {
			select {
			case sub <- b:
			default:
			}
			close(sub)
		}

		delete(c.blockSyncC, blockHash)
		c.bscMutex.Unlock()
	}
	return
}

type syncOption struct {
	Num      int64
	SaveToDB bool
}

// Option represents function signature for option that will be used by SynBlocks
type Option func(interface{})

// SaveToDB represents an option that will be used in SyncPreviousBlocks(opts ...Option)
// set ture if the block's state changes will be saved to persistent DB.
//
// Use only when the blocks need to be persisted to DB, usually when finalize blocks.
func SaveToDB(save bool) func(v interface{}) {
	return func(v interface{}) {
		opt, ok := v.(*syncOption)
		if ok {
			opt.SaveToDB = save
		}
	}
}

// SyncPreviousBlocks syncs N previous blocks that start from a block,
// returns the previous block if success
func (c *Chain) SyncPreviousBlocks(ctx context.Context, b *block.Block, num int64, opts ...Option) *block.Block {
	so := syncOption{
		Num: num,
	}

	for _, opt := range opts {
		opt(&so)
	}

	return c.syncBlocksWithCache(ctx, b, so)
}

// syncBlocksWithCache checks whether the requested block is already in syncing first,
// if yes, we will subscribe the reply channel, and wait for the responding to avoid duplicate
// requests being sent.
// if no, then we will send a request to get the block and state changes from remote.
func (c *Chain) syncBlocksWithCache(ctx context.Context, b *block.Block, opt syncOption) *block.Block {
	replyC := make(chan *block.Block, 1)
	notifyAndClean, ok := c.registerBlockSync(b.PrevHash, replyC)
	if ok {
		// block is already in syncing
		select {
		case pb, ok := <-replyC:
			if ok && pb != nil {
				logging.Logger.Info("sync_block - success, notified",
					zap.Int64("round", pb.Round),
					zap.String("block", pb.Hash),
					zap.Int64("num", opt.Num))
			}
			return pb
		case <-ctx.Done():
			return nil
		}
	}

	pb := c.syncPreviousBlock(ctx, b, opt)
	notifyAndClean(pb)
	return pb
}

func (c *Chain) syncPreviousBlock(ctx context.Context, b *block.Block, opt syncOption) *block.Block {
	pb, _ := c.GetBlock(ctx, b.PrevHash)
	if pb == nil {
		var err error
		pb, err = c.GetNotarizedBlock(ctx, b.PrevHash, b.Round-1)
		if err != nil {
			logging.Logger.Error("sync_block - could not fetch block",
				zap.Int64("round", b.Round-1),
				zap.String("block", b.PrevHash),
				zap.Error(err))
			return nil
		}
	}

	if pb.IsStateComputed() {
		return pb
	}

	logging.Logger.Debug("sync_block - previous block not computed",
		zap.Int64("round", pb.Round),
		zap.String("block", pb.Hash),
		zap.Int8("state_status", pb.GetStateStatus()))

	var ppb *block.Block
	if opt.Num-1 > 0 {
		logging.Logger.Debug("sync_block - get previous previous block",
			zap.Int64("round", pb.Round-1),
			zap.String("block", pb.PrevHash))
		ppb = c.syncBlocksWithCache(ctx, pb,
			syncOption{
				Num:      opt.Num - 1,
				SaveToDB: opt.SaveToDB,
			})
		if ppb == nil {
			return nil
		}
	}

	if ppb != nil {
		pb.SetPreviousBlock(ppb)
		//pb.SetStateDB(ppb, c.GetStateDB())
	}

	if err := c.GetBlockStateChange(pb); err != nil {
		if er := pb.InitStateDB(c.GetStateDB()); er == nil {
			logging.Logger.Debug("sync_block - client state root exist in db", zap.Int64("round", pb.Round))
			return pb
		}

		logging.Logger.Error("sync_block - sync state changes failed",
			zap.Int64("round", pb.Round),
			zap.Int64("num", opt.Num),
			zap.Error(err))
		return nil
	}

	if opt.SaveToDB {
		if err := pb.SaveChanges(ctx, c); err != nil {
			logging.Logger.Error("sync_block - save changes failed",
				zap.Error(err), zap.Int64("round", pb.Round))
		}
	}

	logging.Logger.Info("sync_block - success",
		zap.Int64("round", pb.Round),
		zap.String("block", pb.Hash),
		zap.Int64("num", opt.Num))

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

func (c *Chain) updateFeeStats(fb *block.Block) error {
	var (
		totalFees currency.Coin
		err       error
	)
	if len(fb.Txns) == 0 {
		return nil
	}

	for _, txn := range fb.Txns {
		totalFees, err = currency.AddCoin(totalFees, txn.Fee)
		if err != nil {
			return err
		}
	}
	meanFees, _, err := currency.DivideCoin(totalFees, int64(len(fb.Txns)))
	if err != nil {
		return err
	}
	c.FeeStats.MeanFees = meanFees
	if meanFees > c.FeeStats.MaxFees {
		c.FeeStats.MaxFees = meanFees
	}
	if meanFees < c.FeeStats.MinFees {
		c.FeeStats.MinFees = meanFees
	}
	return nil
}
