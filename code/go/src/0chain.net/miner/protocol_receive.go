package miner

import (
	"context"

	"0chain.net/chain"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*HandleVRFShare - handles the vrf share */
func (mc *Chain) HandleVRFShare(ctx context.Context, msg *BlockMessage) {
	mr := mc.GetMinerRound(msg.VRFShare.Round)
	if mr == nil {
		Logger.Debug("handle vrf share - got vrf share before starting a round", zap.Int64("round", msg.VRFShare.Round))
		pr := mc.GetMinerRound(msg.VRFShare.Round - 1)
		if pr != nil {
			mr = mc.StartNextRound(ctx, pr)
		} else {
			Logger.Error("handle vrf share - no prior round", zap.Int64("round", msg.VRFShare.Round))
			// We can't really provide a VRF share as we don't know the previous round's random number but we can collect the shares
			var r = datastore.GetEntityMetadata("round").Instance().(*round.Round)
			r.Number = msg.VRFShare.Round
			mr = mc.CreateRound(r)
			mc.AddRound(mr)
		}
	}
	if mr != nil {
		mc.AddVRFShare(ctx, mr, msg.VRFShare)
	}
}

/*HandleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	b := msg.Block
	mr := mc.GetMinerRound(b.Round)
	if mr != nil {
		mc.MergeVerificationTickets(ctx, b, mr.GetVerificationTickets(b.Hash))
	}
	if b.Round < mc.CurrentRound-1 {
		Logger.Debug("verify block (round mismatch)", zap.Int64("current_round", mc.CurrentRound), zap.Int64("block_round", b.Round))
		return
	}
	if mr == nil {
		Logger.Error("handle verify block - got block proposal before starting round", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner", b.MinerID))
		pr := mc.GetMinerRound(b.Round - 1)
		mr = mc.StartNextRound(ctx, pr)
		//TODO: byzantine
		mc.setRandomSeed(ctx, mr, b.RoundRandomSeed)
	} else {
		if !mr.IsVRFComplete() {
			//TODO: byzantine
			mc.setRandomSeed(ctx, mr, b.RoundRandomSeed)
		}
	}
	if mr != nil {
		if !mc.ValidGenerator(mr.Round, b) {
			return
		}
		mc.AddToRoundVerification(ctx, mr, b)
	} else {
		Logger.Error("this should not happen %v", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("cround", mc.CurrentRound))
	}
}

/*HandleVerificationTicketMessage - handles the verification ticket message */
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	var err error
	r := msg.Round
	if r == nil {
		r = mc.GetMinerRound(msg.BlockVerificationTicket.Round)
	}
	b := msg.Block // if the ticket is for own generated block, then the message contains the block
	if msg.Sender != node.Self.Node {
		if b == nil {
			b, err = mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
			if err != nil {
				if r != nil {
					err = mc.VerifyTicket(ctx, msg.BlockVerificationTicket.BlockID, &msg.BlockVerificationTicket.VerificationTicket)
					if err != nil {
						Logger.Debug("verification ticket", zap.Error(err))
						return
					}
					r.AddVerificationTicket(msg.BlockVerificationTicket)
					return
				}
				if msg.ShouldRetry() {
					Logger.Info("verification message (no block) retrying", zap.String("block", msg.BlockVerificationTicket.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
					msg.Retry(mc.BlockMessageChannel)
				} else {
					Logger.Error("verification message (no block)", zap.Int64("round", msg.BlockVerificationTicket.Round), zap.String("block", msg.BlockVerificationTicket.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
				}
				return
			}
			if b.Round < mc.LatestFinalizedBlock.Round {
				Logger.Debug("verification message (round mismatch)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
				return
			}
		}
		if r == nil {
			Logger.Debug("verification message (no round)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
			return
		}
		err := mc.VerifyTicket(ctx, b.Hash, &msg.BlockVerificationTicket.VerificationTicket)
		if err != nil {
			Logger.Debug("verification ticket", zap.Error(err))
			return
		}
	}
	mc.ProcessVerifiedTicket(ctx, r, b, &msg.BlockVerificationTicket.VerificationTicket)
}

/*HandleNotarizationMessage - handles the block notarization message */
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	if msg.Notarization.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("notarization message", zap.Int64("round", msg.Notarization.Round), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round), zap.String("block", msg.Notarization.BlockID))
		return
	}
	r := mc.GetMinerRound(msg.Notarization.Round)
	if r == nil {
		if msg.ShouldRetry() {
			Logger.Info("notarization receipt handler (round not started yet) retrying", zap.String("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount))
			msg.Retry(mc.BlockMessageChannel)
		} else {
			Logger.Error("notarization receipt handler (round not started yet)", zap.String("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount))
		}
		return
	}
	msg.Round = r
	b, err := mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		if msg.ShouldRetry() {
			Logger.Info("notarization receipt handler (block not found) retrying", zap.Any("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
			if msg.RetryCount > 2 {
				go mc.GetNotarizedBlock(msg.Notarization.BlockID) // Let's try to download the block proactively
			} else {
				msg.Retry(mc.BlockMessageChannel)
			}
		} else {
			Logger.Error("notarization receipt handler (block not found)", zap.Any("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
		}
		return
	}
	if err := mc.VerifyNotarization(ctx, b.Hash, msg.Notarization.VerificationTickets); err != nil {
		Logger.Error("notarization message", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	mc.MergeVerificationTickets(ctx, b, msg.Notarization.VerificationTickets)
	if !mc.AddNotarizedBlock(ctx, r, b) {
		return
	}
	if mc.BlocksToSharder == chain.NOTARIZED {
		if mc.VerificationTicketsTo == chain.Generator {
			//We assume those who can generate a block in a round are also responsible for sending it to the sharders
			if mc.IsRoundGenerator(r.Round, node.GetSelfNode(ctx).Node) {
				go mc.SendNotarizedBlock(ctx, b)
			}
		}
	}
	mc.StartNextRound(ctx, r)
}

/*HandleNotarizedBlockMessage - handles a notarized block for a previous round*/
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context, msg *BlockMessage) {
	mb := msg.Block
	mr := mc.GetMinerRound(mb.Round)
	if mr == nil {
		Logger.Error("handle notarized block message", zap.Int64("round", mb.Round))
		r := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		r.Number = mb.Round
		//TODO: byzantine
		mc.SetRandomSeed(r, mb.RoundRandomSeed)
		mr = mc.CreateRound(r)
		mc.AddRound(mr)
	} else {
		nb := mr.GetNotarizedBlocks()
		for _, blk := range nb {
			if blk.Hash == mb.Hash {
				return
			}
		}
	}
	b := mc.AddBlock(mb)
	if !mc.AddNotarizedBlock(ctx, mr, b) {
		return
	}
	mc.StartNextRound(ctx, mr)
}
