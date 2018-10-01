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

/*HandleStartRound - handles the start round message */
func (mc *Chain) HandleStartRound(ctx context.Context, msg *BlockMessage) {
	r := msg.Round
	mc.startNewRound(ctx, r)
}

/*HandleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	b := msg.Block
	if b.Round < mc.CurrentRound-1 {
		Logger.Debug("verify block (round mismatch)", zap.Int64("current_round", mc.CurrentRound), zap.Int64("block_round", b.Round))
		return
	}
	mr := mc.GetRound(b.Round)
	if mr == nil {
		r := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		r.Number = b.Round
		r.RandomSeed = b.RoundRandomSeed
		mr = mc.CreateRound(r)
		mc.startNewRound(ctx, mr)
		mr = mc.GetRound(b.Round) // Need this again just in case there is another round already setup and the start didn't happen
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
	b := msg.Block // if the ticket is for own generated block, then the message contains the block
	if b == nil {
		b, err = mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
		if err != nil {
			if msg.ShouldRetry() {
				Logger.Info("verification message (no block) retrying", zap.String("block", msg.BlockVerificationTicket.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
				msg.Retry(mc.BlockMessageChannel)
			} else {
				Logger.Error("verification message (no block)", zap.String("block", msg.BlockVerificationTicket.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
			}
			return
		}
		if b.Round < mc.LatestFinalizedBlock.Round {
			Logger.Debug("verification message (round mismatch)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
			return
		}
	}
	r := msg.Round
	if msg.Sender != node.Self.Node {
		if r == nil {
			r = mc.GetRound(b.Round)
		}
		if r == nil {
			Logger.Debug("verification message (no round)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
			return
		}
		err = mc.VerifyTicket(ctx, b.Hash, &msg.BlockVerificationTicket.VerificationTicket)
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
	r := mc.GetRound(msg.Notarization.Round)
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
	b.MergeVerificationTickets(msg.Notarization.VerificationTickets)
	if !mc.AddNotarizedBlock(ctx, r.Round, b) {
		return
	}
	if !r.IsVerificationComplete() {
		r.CancelVerification()
		if r.Block == nil || r.Block.Weight() < b.Weight() {
			r.Block = b
		}
	}
	if mc.BlocksToSharder == chain.NOTARIZED {
		//We assume those who can generate a block in a round are also responsible for sending it to the sharders
		if mc.CanGenerateRound(r.Round, node.GetSelfNode(ctx).Node) {
			nb := r.GetBestNotarizedBlock()
			if nb.Hash == b.Hash {
				go mc.SendNotarizedBlock(ctx, b)
			}
		}
	}
}

/*HandleRoundTimeout - handles the timeout of a round*/
func (mc *Chain) HandleRoundTimeout(ctx context.Context) {
	if mc.CurrentRound <= 1 {
		if !mc.CanStartNetwork() {
			return
		}
	}
	Logger.Info("round timeout occured", zap.Any("round", mc.CurrentRound))
	r := mc.GetRound(mc.CurrentRound)
	if r.Number > 1 {
		pr := mc.GetRound(r.Number - 1)
		if pr != nil {
			mc.BroadcastNotarizedBlocks(ctx, pr, r)
		}
	}
	r.Round.Block = nil
	if mc.CanGenerateRound(r.Round, node.GetSelfNode(ctx).Node) {
		go mc.GenerateRoundBlock(ctx, r)
	}
}

/*HandleNotarizedBlockMessage - handles a notarized block for a previous round*/
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context, msg *BlockMessage) {
	mb := msg.Block
	mr := mc.GetRound(mb.Round)
	if mr == nil {
		r := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		r.Number = mb.Round
		r.RandomSeed = mb.RoundRandomSeed
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
	b, err := mc.GetBlock(ctx, mb.Hash)
	if err == nil {
		b.MergeVerificationTickets(mb.VerificationTickets)
	} else {
		b = mb
		mc.AddBlock(mb)
	}

	mc.AddNotarizedBlock(ctx, mr.Round, b)
	if !mr.IsVerificationComplete() {
		mr.CancelVerification()
		if mr.Block == nil || mr.Block.Weight() < b.Weight() {
			mr.Block = b
		}
	}
}
