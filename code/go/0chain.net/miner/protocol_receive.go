package miner

import (
	"context"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

// HandleVRFShare - handles the vrf share.
func (mc *Chain) HandleVRFShare(ctx context.Context, msg *BlockMessage) {
	mr := mc.GetMinerRound(msg.VRFShare.Round)
	if mr == nil {
		mr = mc.getRound(ctx, msg.VRFShare.Round)
	}
	if mr != nil {
		mc.AddVRFShare(ctx, mr, msg.VRFShare)
	}
}

func (mc *Chain) enterOnViewChange(ctx context.Context, rn int64) {

	if !config.DevConfiguration.ViewChange {
		return
	}

	var (
		mb          = mc.GetMagicBlock(rn)
		lfb         = mc.GetLatestFinalizedBlock()
		selfNodeKey = node.Self.Underlying().GetKey()

		err error
	)

	if rn > lfb.Round && rn-lfb.Round > chain.ViewChangeOffset {
		if _, err = mc.ensureLatestFinalizedBlocks(ctx); err != nil {
			Logger.Error("get LFB/LFBM from sharder", zap.Error(err))
			return
		}
		mb = mc.GetMagicBlock(rn)          // update
		lfb = mc.GetLatestFinalizedBlock() // update
	}

	if !mb.Miners.HasNode(selfNodeKey) {
		return
	}

	var crn = mc.GetCurrentRound()
	if crn < lfb.Round {
		mc.SetCurrentRound(crn)
	}

	if mc.isJoining(crn) {
		go mc.StartNextRound(ctx, mc.GetMinerRound(crn))
	}
}

// HandleVerifyBlockMessage - handles the verify block message.
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context,
	msg *BlockMessage) {

	var b = msg.Block

	if b.Round < mc.GetCurrentRound()-1 {
		Logger.Debug("verify block (round mismatch)",
			zap.Int64("current_round", mc.GetCurrentRound()),
			zap.Int64("block_round", b.Round))
		return
	}

	var mr, pr = mc.GetMinerRound(b.Round), mc.GetMinerRound(b.Round - 1)

	if pr == nil {
		Logger.Error("handle verify block -- no previous round (ignore)",
			zap.Int64("round", b.Round), zap.Int64("prev_round", b.Round-1))
		mc.enterOnViewChange(ctx, b.Round)
		return
	}

	if mr == nil {

		Logger.Error("handle verify block -- got block proposal before starting round",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("miner", b.MinerID))

		if mr = mc.getRound(ctx, b.Round); mr == nil {
			Logger.Error("handle verify block -- far ahead of sharders (ignore)",
				zap.Int64("round", b.Round))
			return
		}
		//TODO: Byzantine
		mc.startRound(ctx, mr, b.GetRoundRandomSeed())

	} else {
		if !mr.IsVRFComplete() {
			Logger.Info("handle verify block - got block proposal before VRF is complete",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("miner", b.MinerID))

			if mr.GetTimeoutCount() < b.RoundTimeoutCount {
				Logger.Info("Insync ignoring handle verify block - got block proposal before VRF is complete",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.String("miner", b.MinerID),
					zap.Int("round_toc", mr.GetTimeoutCount()),
					zap.Int("round_toc", b.RoundTimeoutCount))
				return
			}
			//TODO: Byzantine
			mc.startRound(ctx, mr, b.GetRoundRandomSeed())
		}

		var vts = mr.GetVerificationTickets(b.Hash)

		if len(vts) > 0 {
			mc.MergeVerificationTickets(ctx, b, vts)
			if b.IsBlockNotarized() {
				if mr.GetRandomSeed() != b.GetRoundRandomSeed() {
					/* Since this is a notarized block, we are accepting it.
					   TODO: Byzantine
					*/
					b1, r1 := mc.AddNotarizedBlockToRound(mr, b)
					b = b1
					mr = r1.(*Round)
					Logger.Info("Added a notarizedBlockToRound - got notarized block with different ",
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

	if mr == nil {
		Logger.Error("this should not happen %v", zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("cround", mc.GetCurrentRound()))
		return
	}

	// else if -> mr is not nil

	if mr.IsVerificationComplete() {
		return
	}

	if mr.GetRandomSeed() != b.GetRoundRandomSeed() {
		Logger.Error("Got a block for verification with wrong random seed",
			zap.Int64("round", mr.GetRoundNumber()),
			zap.Int("roundToc", mr.GetTimeoutCount()),
			zap.Int("blockToc", b.RoundTimeoutCount),
			zap.Int64("round_rrs", mr.GetRandomSeed()),
			zap.Int64("block_rrs", b.GetRoundRandomSeed()))
		return
	}

	if !mc.ValidGenerator(mr.Round, b) {
		Logger.Error("Not a valid generator. Ignoring block with hash = " + b.Hash)
		return
	}

	Logger.Info("Added block to Round with hash = " + b.Hash)
	mc.AddToRoundVerification(ctx, mr, b)
}

// HandleVerificationTicketMessage - handles the verification ticket message.
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context,
	msg *BlockMessage) {

	var mr = msg.Round

	if mr == nil {
		mr = mc.GetMinerRound(msg.BlockVerificationTicket.Round)
		if mr == nil {
			mr = mc.getRound(ctx, msg.BlockVerificationTicket.Round)
			if mr == nil {
				return // miner is far ahead of sharders, skip for now
			}
		}
	}

	if mc.GetMinerRound(msg.BlockVerificationTicket.Round-1) == nil {
		Logger.Error("handle vt. msg -- no previous round (ignore)",
			zap.Int64("round", msg.BlockVerificationTicket.Round),
			zap.Int64("prev_round", msg.BlockVerificationTicket.Round-1))
		return
	}

	var b, err = mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
	if err != nil {
		err = mc.VerifyTicket(ctx, msg.BlockVerificationTicket.BlockID,
			&msg.BlockVerificationTicket.VerificationTicket,
			mr.GetRoundNumber())
		if err != nil {
			Logger.Debug("verification ticket", zap.Error(err))
			return
		}
		mr.AddVerificationTicket(msg.BlockVerificationTicket)
		return
	}

	var lfb = mc.GetLatestFinalizedBlock()
	if b.Round < lfb.Round {
		Logger.Debug("verification message (round mismatch)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.Int64("finalized_round", lfb.Round))
		return
	}

	err = mc.VerifyTicket(ctx, b.Hash,
		&msg.BlockVerificationTicket.VerificationTicket, mr.GetRoundNumber())
	if err != nil {
		Logger.Debug("verification ticket", zap.Error(err))
		return
	}

	mc.ProcessVerifiedTicket(ctx, mr, b,
		&msg.BlockVerificationTicket.VerificationTicket)
}

// HandleNotarizationMessage - handles the block notarization message.
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {

	var lfb = mc.GetLatestFinalizedBlock()
	if msg.Notarization.Round < lfb.Round {
		Logger.Debug("handle notarization message",
			zap.Int64("round", msg.Notarization.Round),
			zap.Int64("finalized_round", lfb.Round),
			zap.String("block", msg.Notarization.BlockID))
		return
	}

	var r = mc.GetMinerRound(msg.Notarization.Round)
	if r == nil {
		if msg.ShouldRetry() {
			Logger.Error("handle notarization message (round not started yet) retrying",
				zap.String("block", msg.Notarization.BlockID),
				zap.Int8("retry_count", msg.RetryCount))
			msg.Retry(mc.blockMessageChannel)
		} else {
			Logger.Error("handle notarization message (round not started yet)",
				zap.String("block", msg.Notarization.BlockID),
				zap.Int8("retry_count", msg.RetryCount))
		}
		return
	}

	if mc.GetMinerRound(msg.Notarization.Round-1) == nil {
		Logger.Error("handle notarization message -- no previous round",
			zap.Int64("round", msg.Notarization.Round),
			zap.Int64("prev_round", msg.Notarization.Round-1))
		return
	}

	msg.Round = r

	var b, err = mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		mc.AsyncFetchNotarizedBlock(msg.Notarization.BlockID)
		return
	}

	var vts = b.UnknownTickets(msg.Notarization.VerificationTickets)
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
		if mr = mc.getRound(ctx, nb.Round); mr == nil {
			return // miner is far ahead of sharders, skip for now
		}
		mc.startRound(ctx, mr, nb.GetRoundRandomSeed())
	} else {

		if mc.GetMinerRound(nb.Round-1) == nil {
			Logger.Error("handle not. block -- no previous round (ignore)",
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

	mc.StartNextRound(ctx, mr) // start next or skip
}
