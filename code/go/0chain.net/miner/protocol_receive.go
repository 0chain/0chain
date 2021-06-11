package miner

import (
	"context"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

// HandleVRFShare - handles the vrf share.
func (mc *Chain) HandleVRFShare(ctx context.Context, msg *BlockMessage) {

	var mr = mc.getOrStartRoundNotAhead(ctx, msg.VRFShare.Round)
	if mr == nil {
		return
	}

	// add the VRFS
	mc.AddVRFShare(ctx, mr, msg.VRFShare)
}

// HandleVerifyBlockMessage - handles the verify block message.
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context,
	msg *BlockMessage) {

	var b = msg.Block

	if b.Round < mc.GetCurrentRound()-1 {
		logging.Logger.Debug("verify block (round mismatch)",
			zap.Int64("current_round", mc.GetCurrentRound()),
			zap.Int64("block_round", b.Round))
		return
	}

	var mr, pr = mc.GetMinerRound(b.Round), mc.GetMinerRound(b.Round - 1)

	if pr == nil {
		logging.Logger.Error("handle verify block -- no previous round (ignore)",
			zap.Int64("round", b.Round), zap.Int64("prev_round", b.Round-1))
		return
	}

	if mr == nil {

		logging.Logger.Error("handle verify block -- got block proposal before starting round",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("miner", b.MinerID))

		var mr = mc.getOrStartRoundNotAhead(ctx, b.Round)
		if mr == nil {
			logging.Logger.Error("handle verify block -- can't start new round",
				zap.Int64("round", b.Round))
			return
		}

		mc.startRound(ctx, mr, b.GetRoundRandomSeed())
	} else {
		if !mr.IsVRFComplete() {
			logging.Logger.Info("handle verify block - got block proposal before VRF is complete",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("miner", b.MinerID))

			if mr.GetTimeoutCount() < b.RoundTimeoutCount {
				logging.Logger.Info("Insync ignoring handle verify block - got block proposal before VRF is complete",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.String("miner", b.MinerID),
					zap.Int("round_toc", mr.GetTimeoutCount()),
					zap.Int("round_toc", b.RoundTimeoutCount))
				return
			}
			mc.startRound(ctx, mr, b.GetRoundRandomSeed())
		}

		var vts = mr.GetVerificationTickets(b.Hash)

		if len(vts) > 0 {
			mc.MergeVerificationTickets(ctx, b, vts)
			if b.IsBlockNotarized() {
				if mr.GetRandomSeed() != b.GetRoundRandomSeed() {
					/* Since this is a notarized block, we are accepting it. */
					b1, r1, err := mc.AddNotarizedBlockToRound(mr, b)
					if err != nil {
						logging.Logger.Error("handle verify block failed",
							zap.Int64("round", b.Round),
							zap.String("block", b.Hash),
							zap.String("miner", b.MinerID),
							zap.Error(err))
						return
					}
					b = b1
					mr = r1.(*Round)
					logging.Logger.Info("Added a notarizedBlockToRound - got notarized block with different ",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.String("miner", b.MinerID),
						zap.Int("round_toc", mr.GetTimeoutCount()),
						zap.Int("round_toc", b.RoundTimeoutCount))

				} else {
					b = mc.AddRoundBlock(mr, b)
				}

				mc.checkBlockNotarization(ctx, mr, b)
				return
			}
		}
	}
	// reassign the 'mr' variable, the miner should not be nil, but somehow
	//, this happened!! how could it happen?
	mr = mc.GetMinerRound(b.Round)
	if mr == nil {
		logging.Logger.Error("this should not happen", zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("cround", mc.GetCurrentRound()))
		return
	}

	// else if -> mr is not nil

	if mr.IsVerificationComplete() {
		return
	}

	if mr.GetRandomSeed() != b.GetRoundRandomSeed() {
		logging.Logger.Error("Got a block for verification with wrong random seed",
			zap.Int64("round", mr.GetRoundNumber()),
			zap.Int("roundToc", mr.GetTimeoutCount()),
			zap.Int("blockToc", b.RoundTimeoutCount),
			zap.Int64("round_rrs", mr.GetRandomSeed()),
			zap.Int64("block_rrs", b.GetRoundRandomSeed()))
		return
	}

	if !mc.ValidGenerator(mr.Round, b) {
		logging.Logger.Error("Not a valid generator. Ignoring block",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return
	}

	logging.Logger.Info("Added block to Round",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("magic block", b.LatestFinalizedMagicBlockHash),
		zap.Int64("magic block round", b.LatestFinalizedMagicBlockRound))
	mc.AddToRoundVerification(ctx, mr, b)
}

// HandleVerificationTicketMessage - handles the verification ticket message.
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context,
	msg *BlockMessage) {

	var (
		bvt = msg.BlockVerificationTicket
		rn  = bvt.Round
	)

	var mr = mc.getOrStartRoundNotAhead(ctx, rn)
	if mr == nil {
		logging.Logger.Error("handle vt. msg -- ahead of sharders or no pr",
			zap.Int64("round", rn))
		return
	}

	if mc.GetMinerRound(rn-1) == nil {
		logging.Logger.Error("handle vt. msg -- no previous round (ignore)",
			zap.Int64("round", rn), zap.Int64("pr", rn-1))
		return
	}

	var b, err = mc.GetBlock(ctx, bvt.BlockID)
	if err != nil {
		err = mc.VerifyTicket(ctx, bvt.BlockID, &bvt.VerificationTicket, rn)
		if err != nil {
			logging.Logger.Debug("verification ticket", zap.Error(err))
			return
		}
		mr.AddVerificationTicket(bvt)
		return
	}

	logging.Logger.Debug("verification ticket",
		zap.Int64("round", rn),
		zap.String("block hash", b.Hash),
		zap.String("block id", bvt.BlockID))

	var lfb = mc.GetLatestFinalizedBlock()
	if b.Round < lfb.Round {
		logging.Logger.Debug("verification message (round mismatch)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.Int64("lfb", lfb.Round))
		return
	}

	err = mc.VerifyTicket(ctx, b.Hash, &bvt.VerificationTicket, rn)
	if err != nil {
		logging.Logger.Debug("verification ticket", zap.Error(err))
		return
	}

	mc.ProcessVerifiedTicket(ctx, mr, b, &bvt.VerificationTicket)
}

// HandleNotarizationMessage - handles the block notarization message.
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {

	var (
		lfb = mc.GetLatestFinalizedBlock()
		not = msg.Notarization
	)

	if not.Round < lfb.Round {
		logging.Logger.Debug("handle notarization message",
			zap.Int64("round", not.Round),
			zap.Int64("finalized_round", lfb.Round),
			zap.String("block", not.BlockID))
		return
	}

	var r = mc.GetMinerRound(not.Round)
	if r == nil {
		if msg.ShouldRetry() {
			logging.Logger.Error("handle notarization message (round not started yet) retrying",
				zap.String("block", not.BlockID),
				zap.Int8("retry_count", msg.RetryCount))
			msg.Retry(mc.blockMessageChannel)
		} else {
			logging.Logger.Error("handle notarization message (round not started yet)",
				zap.String("block", not.BlockID),
				zap.Int8("retry_count", msg.RetryCount))
		}
		return
	}

	if mc.GetMinerRound(not.Round-1) == nil {
		logging.Logger.Error("handle notarization message -- no previous round",
			zap.Int64("round", not.Round),
			zap.Int64("prev_round", not.Round-1))
		return
	}

	msg.Round = r

	var b, err = mc.GetBlock(ctx, not.BlockID)
	if err != nil {
		go mc.GetNotarizedBlock(ctx, not.BlockID, not.Round)
		return
	}

	var vts = b.UnknownTickets(not.VerificationTickets)
	if len(vts) == 0 {
		return
	}

	go mc.MergeNotarization(ctx, r, b, vts)
}

// HandleNotarizedBlockMessage - handles a notarized block for a previous round.
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context,
	msg *BlockMessage) {

	var (
		nb = msg.Block
		mr = mc.GetMinerRound(nb.Round)
	)

	if mr == nil {

		if mr = mc.getOrStartRoundNotAhead(ctx, nb.Round); mr == nil {
			logging.Logger.Error("handle not. block -- ahead of sharders or no pr",
				zap.Int64("round", nb.Round),
				zap.Bool("has_pr", mc.GetMinerRound(nb.Round-1) != nil))
			return
		}

		mc.startRound(ctx, mr, nb.GetRoundRandomSeed())

	} else {

		if mc.GetMinerRound(nb.Round-1) == nil {
			logging.Logger.Error("handle not. block -- no previous round (ignore)",
				zap.Int64("round", nb.Round),
				zap.Int64("prev_round", nb.Round-1))
			return
		}

		if mr.IsVerificationComplete() {
			return // verification for the round complete
		}

		for _, blk := range mr.GetNotarizedBlocks() {
			if blk.Hash == nb.Hash {
				return // already have
			}
		}

		if !mr.IsVRFComplete() {
			mc.startRound(ctx, mr, nb.GetRoundRandomSeed())
		}

	}

	var b = mc.AddRoundBlock(mr, nb)
	if !mc.AddNotarizedBlock(ctx, mr, b) {
		return
	}

	if mc.isAheadOfSharders(ctx, nb.Round+1) {
		logging.Logger.Error("handle not. block -- ahead of sharders",
			zap.Int64("round", nb.Round+1))
		return
	}

	mc.StartNextRound(ctx, mr) // start next or skip
}
