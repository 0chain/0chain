package miner

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers() {
	ctx := common.GetRootContext()
	go GetMinerChain().BlockWorker(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	var protocol Protocol = mc
	for true {
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.GetBlockMessageChannel():
			if msg.Sender != nil {
				Logger.Debug("message", zap.Any("msg", GetMessageLookup(msg.Type)), zap.Any("sender_index", msg.Sender.SetIndex), zap.Any("id", msg.Sender.GetKey()))
			} else {
				Logger.Debug("message", zap.Any("msg", GetMessageLookup(msg.Type)))
			}
			switch msg.Type {
			case MessageStartRound:
				protocol.HandleStartRound(ctx, msg)
			case MessageVerify:
				protocol.HandleVerifyBlockMessage(ctx, msg)
			case MessageVerificationTicket:
				protocol.HandleVerificationTicketMessage(ctx, msg)
			case MessageNotarization:
				protocol.HandleNotarizationMessage(ctx, msg)
			}
		}
	}
}

/*HandleStartRound - handles the start round message */
func (mc *Chain) HandleStartRound(ctx context.Context, msg *BlockMessage) {
	r := msg.Round
	mc.startNewRound(ctx, r)
}

func (mc *Chain) startNewRound(ctx context.Context, mr *Round) {
	pr := mc.GetRound(mr.Number - 1)
	//TODO: If for some reason the server is lagging behind (like network outage) we need to fetch the previous round info
	// before proceeding
	if pr == nil {
		Logger.Debug("start new round (previous round not found)", zap.Int64("round", mr.Number))
		return
	}
	if !mc.AddRound(mr) {
		Logger.Debug("start new round (round already exists)", zap.Int64("round", mr.Number))
		return
	}
	self := node.GetSelfNode(ctx)
	rank := mr.GetRank(self.SetIndex)
	Logger.Info("*** starting round ***", zap.Any("round", mr.Number), zap.Any("index", self.SetIndex), zap.Any("rank", rank))
	if !mc.CanGenerateRound(&mr.Round, self.Node) {
		return
	}
	//NOTE: If there are not enough txns, this will not advance further even though rest of the network is. That's why this is a goroutine
	go mc.GenerateRoundBlock(ctx, mr)
}

/*HandleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	b := msg.Block
	if b.Round < mc.CurrentRound {
		return
	}
	mr := mc.GetRound(b.Round)
	if mr == nil {
		// TODO: This can happen because
		// 1) This is past round that is no longer applicable - reject it
		// 2) This is a future round we didn't know about yet as our network is slow or something
		// 3) The verify message received before the start round message
		// WARNING: Because of this, we don't know the ranks of the round as we don't have the seed in this implementation
		r := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		r.Number = b.Round
		r.RandomSeed = b.RoundRandomSeed
		mr = mc.CreateRound(r)
		if !mc.ValidGenerator(&mr.Round, b) {
			Logger.Debug("verify block (yes mr, invalid generator)", zap.Any("round", mr.Number), zap.Any("block", b.Hash))
			return
		}
		mc.startNewRound(ctx, mr)
		mr = mc.GetRound(b.Round) // Need this again just in case there is another round already setup and the start didn't happen
	} else {
		if !mc.ValidGenerator(&mr.Round, b) {
			Logger.Debug("verify block (yes mr, invalid generator)", zap.Any("round", mr.Number), zap.Any("block", b.Hash))
			return
		}
	}
	mc.AddToRoundVerification(ctx, mr, b)
}

/*HandleVerificationTicketMessage - handles the verification ticket message */
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	b, err := mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
	if err != nil {
		// TODO: If we didn't see this block so far, may be it's better to ask for it
		Logger.Debug("verification message (no block)", zap.String("block", msg.BlockVerificationTicket.BlockID), zap.Error(err))
		return
	}
	if b.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("verification message (round mismatch)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
		return
	}
	r := mc.GetRound(b.Round)
	if r == nil {
		Logger.Debug("verification message (no round)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
		return
	}
	if mc.IsBlockNotarized(ctx, b) {
		return
	}
	err = mc.VerifyTicket(ctx, b, &msg.BlockVerificationTicket.VerificationTicket)
	if err != nil {
		Logger.Debug("verification ticket", zap.Error(err))
		return
	}
	mc.ProcessVerifiedTicket(ctx, r, b, &msg.BlockVerificationTicket.VerificationTicket)
}

/*HandleNotarizationMessage - handles the block notarization message */
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	r := msg.Round
	if r.Number < mc.LatestFinalizedBlock.Round {
		Logger.Debug("notarization message", zap.Int64("round", r.Number), zap.String("block", msg.Notarization.BlockID), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
		return
	}
	b, err := mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		// TODO: If we didn't see this block so far, may be it's better to ask for it
		Logger.Debug("notarization message", zap.Any("block", msg.Notarization.BlockID), zap.Error(err))
		return
	}
	if err := mc.VerifyNotarization(ctx, b, msg.Notarization.VerificationTickets); err != nil {
		Logger.Debug("notarization message", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	if !r.IsVerificationComplete() {
		r.CancelVerification()
		r.Block = b
	}
	r.AddNotarizedBlock(b)
	pr := mc.GetRound(b.Round - 1)
	if pr != nil {
		if pr.Number != 0 && pr.Block != nil {
			mc.FinalizeRound(ctx, &pr.Round, mc)
		}
	}
}
