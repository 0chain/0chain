package miner

import (
	"context"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*HandleVRFShare - handles the vrf share */
func (mc *Chain) HandleVRFShare(ctx context.Context, msg *BlockMessage) {
	mr := mc.GetMinerRound(msg.VRFShare.Round)
	if mr == nil {
		mr = mc.getRound(ctx, msg.VRFShare.Round)
	}
	if mr != nil {
		mc.AddVRFShare(ctx, mr, msg.VRFShare)
	}
}

/*handleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	b := msg.Block
	if b.Round < mc.GetCurrentRound()-1 {
		Logger.Debug("verify block (round mismatch)", zap.Int64("current_round", mc.GetCurrentRound()), zap.Int64("block_round", b.Round))
		return
	}
	mr := mc.GetMinerRound(b.Round)
	if mr == nil {
		Logger.Error("handle verify block - got block proposal before starting round", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner", b.MinerID))
		if mr = mc.getRound(ctx, b.Round); mr == nil {
			// first round after joining BC case, the node have to
			// handle the block, even if it's far ahead of sharders,
			// because sharders don't send LFB tickets to it (not
			// active in chain) and sharders don't allow to get LFB
			// because node it not registered in the sharders yet;
			// this block contains MB with this node, that joining
			// BC -- that is the case; thus, a block generator makes
			// this node joining BC

			if b.MagicBlock == nil {
				Logger.Error("handle verify block - far ahead of sharders, no MB case",
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.String("miner", b.MinerID))
				return
			}

			var selfKey = node.Self.GetKey()
			if !b.MagicBlock.Miners.HasNode(selfKey) {
				Logger.Error("handle verify block - far ahead of sharders, MB hasn't this miner",
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.String("miner", b.MinerID))
				return
			}

			// advance LFB ticket for this case
			var pmb = mc.GetMagicBlock(b.Round - 1)
			if !pmb.Miners.HasNode(selfKey) {
				mc.AddReceivedLFBTicket(ctx, &chain.LFBTicket{
					Round: b.Round,
				})
			}

			var r = round.NewRound(b.Round)
			mr = mc.CreateRound(r)
			mr = mc.AddRound(mr).(*Round)
			// mc.SetCurrentRound(mr.GetRoundNumber()) // use it as current ?
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
					zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner", b.MinerID),
					zap.Int("round_toc", mr.GetTimeoutCount()), zap.Int("round_toc", b.RoundTimeoutCount))
				return
			}
			//TODO: Byzantine
			mc.startRound(ctx, mr, b.GetRoundRandomSeed())
		}
		vts := mr.GetVerificationTickets(b.Hash)
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
						zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner", b.MinerID),
						zap.Int("round_toc", mr.GetTimeoutCount()), zap.Int("round_toc", b.RoundTimeoutCount))

				} else {
					b = mc.AddRoundBlock(mr, b)
				}

				mc.checkBlockNotarization(ctx, mr, b)
				return
			}
		}
	}
	if mr != nil {
		if mr.IsVerificationComplete() {
			return
		}
		if mr.GetRandomSeed() != b.GetRoundRandomSeed() {
			Logger.Error("Got a block for verification with wrong randomseed", zap.Int64("roundNum", mr.GetRoundNumber()),
				zap.Int("roundToc", mr.GetTimeoutCount()), zap.Int("blockToc", b.RoundTimeoutCount),
				zap.Int64("roundrrs", mr.GetRandomSeed()), zap.Int64("blockrrs", b.GetRoundRandomSeed()))
			return
		}
		if !mc.ValidGenerator(mr.Round, b) {
			Logger.Error("Not a valid generator. Ignoring block with hash = " + b.Hash)
			return
		}
		Logger.Info("Added block to Round with hash = " + b.Hash)
		mc.AddToRoundVerification(ctx, mr, b)
	} else {
		Logger.Error("this should not happen %v", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("cround", mc.GetCurrentRound()))
	}
}

/*HandleVerificationTicketMessage - handles the verification ticket message */
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	var err error
	mr := msg.Round
	if mr == nil {
		mr = mc.GetMinerRound(msg.BlockVerificationTicket.Round)
		if mr == nil {
			mr = mc.getRound(ctx, msg.BlockVerificationTicket.Round)
			if mr == nil {
				return // miner is far ahead of sharders, skip for now
			}
		}
	}
	b, err := mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
	if err != nil {
		if mr != nil {
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
		return
	}
	lfb := mc.GetLatestFinalizedBlock()
	if b.Round < lfb.Round {
		Logger.Debug("verification message (round mismatch)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", lfb.Round))
		return
	}
	err = mc.VerifyTicket(ctx, b.Hash,
		&msg.BlockVerificationTicket.VerificationTicket, mr.GetRoundNumber())
	if err != nil {
		Logger.Debug("verification ticket", zap.Error(err))
		return
	}
	mc.ProcessVerifiedTicket(ctx, mr, b, &msg.BlockVerificationTicket.VerificationTicket)
}

/*HandleNotarizationMessage - handles the block notarization message */
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	lfb := mc.GetLatestFinalizedBlock()
	if msg.Notarization.Round < lfb.Round {
		Logger.Debug("notarization message", zap.Int64("round", msg.Notarization.Round), zap.Int64("finalized_round", lfb.Round), zap.String("block", msg.Notarization.BlockID))
		return
	}
	r := mc.GetMinerRound(msg.Notarization.Round)
	if r == nil {
		if msg.ShouldRetry() {
			Logger.Error("notarization receipt handler (round not started yet) retrying", zap.String("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount))
			msg.Retry(mc.blockMessageChannel)
		} else {
			Logger.Error("notarization receipt handler (round not started yet)", zap.String("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount))
		}
		return
	}
	msg.Round = r
	b, err := mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		mc.AsyncFetchNotarizedBlock(msg.Notarization.BlockID)
		return
	}
	vts := b.UnknownTickets(msg.Notarization.VerificationTickets)
	if len(vts) == 0 {
		return
	}
	go mc.MergeNotarization(ctx, r, b, vts)
}

/*HandleNotarizedBlockMessage - handles a notarized block for a previous round*/
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context, msg *BlockMessage) {
	mb := msg.Block
	mr := mc.GetMinerRound(mb.Round)
	if mr == nil {
		if mr = mc.getRound(ctx, mb.Round); mr == nil {
			return // miner is far ahead of sharders, skip for now
		}
		mc.startRound(ctx, mr, mb.GetRoundRandomSeed())
	} else {
		if mr.IsVerificationComplete() {
			return // verification for the round complete
		}
		for _, blk := range mr.GetNotarizedBlocks() {
			if blk.Hash == mb.Hash {
				return // already have
			}
		}
		if !mr.IsVRFComplete() {
			if mc.isNeedViewChange(mb.Round + 1) {
				// kick new miners, joining the VC
				//
				// since the AddReceivedLFBTicket uses buffered channel
				// we have to make sure, the ticket set before the
				// startRound
				for mc.isAheadOfSharders(ctx, mb.Round) {
					mc.AddReceivedLFBTicket(ctx, &chain.LFBTicket{
						Round: mb.Round,
					})
				}

				// and take previous block required to move on
				go mc.AsyncFetchNotarizedBlock(mb.PrevHash)

			}
			mc.startRound(ctx, mr, mb.GetRoundRandomSeed())
		}
	}
	b := mc.AddRoundBlock(mr, mb)
	if !mc.AddNotarizedBlock(ctx, mr, b) {
		return
	}
	mc.StartNextRound(ctx, mr) // start next or skip
}
