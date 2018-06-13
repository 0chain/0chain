package miner

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"go.uber.org/zap"
)

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers() {
	ctx := common.GetRootContext()
	go GetMinerChain().BlockWorker(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	// var protocol Protocol = mc
	for true {
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.GetBlockMessageChannel():
			if msg.Sender != nil {
				Logger.Info("message", zap.Any("msg", GetMessageLookup(msg.Type)), zap.Any("sender_index", msg.Sender.SetIndex), zap.Any("id", msg.Sender.GetKey()))
			} else {
				Logger.Info("message", zap.Any("msg", GetMessageLookup(msg.Type)))
			}
			switch msg.Type {
			case MessageStartRound:
				mc.HandleStartRound(ctx, msg)
			case MessageVerify:
				mc.HandleVerifyBlockMessage(ctx, msg)
			case MessageVerificationTicket:
				mc.HandleVerificationTicketMessage(ctx, msg)
			case MessageNotarization:
				mc.HandleNotarizationMessage(ctx, msg)
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
		return
	}
	if !mc.AddRound(mr) {
		return
	}
	/* TODO: We need time based pruning which will happen when we also start building blocks with transactions that are within certain timeframe.
	ppr := mc.GetRound(mr.Number - 2)
	if ppr != nil {
		mc.DeleteRound(ctx, ppr)
	} */
	self := node.GetSelfNode(ctx)
	rank := mr.GetRank(self.SetIndex)
	Logger.Info("*** starting round ***", zap.Any("round", mr.Number), zap.Any("index", self.SetIndex), zap.Any("rank", rank))
	//TODO: For now, if the rank happens to be in the bottom half, we assume no need to generate block
	if 2*rank > mc.Miners.Size() {
		return
	}
	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnEntityMetadata)
	defer memorystore.Close(ctx)
	mc.GenerateRoundBlock(ctx, mr)
}

/*HandleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	mc.AddToVerification(ctx, msg.Block)
}

/*HandleVerificationTicketMessage - handles the verification ticket message */
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	b, err := mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
	if err != nil {
		// TODO: If we didn't see this block so far, may be it's better to ask for it
		return
	}
	r := mc.GetRound(b.Round)
	if r == nil {
		return
	}
	if mc.IsBlockNotarized(ctx, b) {
		return
	}
	err = mc.VerifyTicket(ctx, b, &msg.BlockVerificationTicket.VerificationTicket)
	if err != nil {
		return
	}
	mc.ProcessVerifiedTicket(ctx, r, b, &msg.BlockVerificationTicket.VerificationTicket)
}

/*HandleNotarizationMessage - handles the block notarization message */
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	b, err := mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		// TODO: If we didn't see this block so far, may be it's better to ask for it
		return
	}
	if err := mc.VerifyNotarization(ctx, b, msg.Notarization.VerificationTickets); err != nil {
		return
	}
	r := msg.Round
	if !r.IsVerificationComplete() {
		r.CancelVerification()
		r.Block = b
	}

	//TODO: Check this condition carefully
	if r.Number < mc.CurrentRound-1 || r.Number > mc.CurrentRound {
		return
	}
	r.AddNotarizedBlock(b)
	pr := mc.GetRound(b.Round - 1)
	if pr != nil {
		if pr.Number != 0 && pr.Block != nil {
			mc.FinalizeRound(ctx, pr)
		}
	}
}
