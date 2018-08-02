package miner

import (
	"context"
	"time"

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
	mc := GetMinerChain()
	go mc.BlockWorker(ctx)
	go mc.BlockFinalizationWorker(ctx, mc)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	var RoundTimeout = 10 * time.Second
	var protocol Protocol = mc
	for true {
		var roundTimeout = time.NewTimer(RoundTimeout)
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.GetBlockMessageChannel():
			roundTimeout.Stop()
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
			case MessageNotarizedBlock:
				protocol.HandleNotarizedBlockMessage(ctx, msg)
			}
			if msg.Sender != nil {
				Logger.Debug("message (done)", zap.Any("msg", GetMessageLookup(msg.Type)), zap.Any("sender_index", msg.Sender.SetIndex), zap.Any("id", msg.Sender.GetKey()))
			} else {
				Logger.Debug("message (done)", zap.Any("msg", GetMessageLookup(msg.Type)))
			}
		case <-roundTimeout.C:
			protocol.HandleRoundTimeout(ctx)
		}
	}
}

/*HandleStartRound - handles the start round message */
func (mc *Chain) HandleStartRound(ctx context.Context, msg *BlockMessage) {
	r := msg.Round
	mc.startNewRound(ctx, r)
}

/*HandleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	b := msg.Block
	if b.Round < mc.CurrentRound {
		Logger.Debug("verify block (round mismatch)", zap.Int64("current_round", mc.CurrentRound), zap.Int64("block_round", b.Round))
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
			Logger.Debug("verify block (no mr, invalid generator)", zap.Any("round", mr.Number), zap.Any("block", b.Hash))
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
	var err error
	b := msg.Block
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
		if mc.IsBlockNotarized(ctx, b) {
			Logger.Debug("verification ticket (already notarized)", zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return
		}
		err = mc.VerifyTicket(ctx, b, &msg.BlockVerificationTicket.VerificationTicket)
		if err != nil {
			Logger.Debug("verification ticket", zap.Error(err))
			return
		}
	}
	mc.ProcessVerifiedTicket(ctx, r, b, &msg.BlockVerificationTicket.VerificationTicket)
}

/*HandleNotarizationMessage - handles the block notarization message */
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
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
	if r.Number < mc.LatestFinalizedBlock.Round {
		Logger.Debug("notarization message", zap.Int64("round", r.Number), zap.String("block", msg.Notarization.BlockID), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
		return
	}
	b, err := mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		if msg.ShouldRetry() {
			Logger.Info("notarization receipt handler (block not found) retrying", zap.Any("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
			msg.Retry(mc.BlockMessageChannel)
		} else {
			Logger.Error("notarization receipt handler (block not found)", zap.Any("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount), zap.Error(err))
		}
		return
	}
	if err := mc.VerifyNotarization(ctx, b, msg.Notarization.VerificationTickets); err != nil {
		Logger.Error("notarization message", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	b.MergeVerificationTickets(msg.Notarization.VerificationTickets)
	mc.AddNotarizedBlock(ctx, &r.Round, b)
	if !r.IsVerificationComplete() {
		r.CancelVerification()
		if r.Block == nil || r.Block.Weight() < b.Weight() {
			r.Block = b
		}
	}
}

/*HandleRoundTimeout - handles the timeout of a round*/
func (mc *Chain) HandleRoundTimeout(ctx context.Context) {
	if mc.CurrentRound == 0 {
		if !mc.CanStartNetwork() {
			return
		}
	}
	Logger.Info("round timeout occured", zap.Any("round", mc.CurrentRound))
	r := mc.GetRound(mc.CurrentRound)
	r.Round.Block = nil
	if mc.CanGenerateRound(&r.Round, node.GetSelfNode(ctx).Node) {
		go mc.GenerateRoundBlock(ctx, r)
	} else if r.Number > 1 {
		pr := mc.GetRound(r.Number - 1)
		go mc.BroadcastNotarizedBlocks(ctx, pr, r)
	}
}

/*HandleNotarizedBlockMessage - handles a notarized block for a previous round*/
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context, msg *BlockMessage) {
	r := mc.GetRound(msg.Block.Round)
	if r == nil {
		return
	}
	nb := r.GetNotarizedBlocks()
	for _, blk := range nb {
		if blk.Hash == msg.Block.Hash {
			return
		}
	}
	if !mc.ValidGenerator(&r.Round, msg.Block) {
		Logger.Debug("verify block (yes mr, invalid generator)", zap.Any("round", r.Number), zap.Any("block", msg.Block.Hash))
		return
	}
	if err := mc.VerifyNotarization(ctx, msg.Block, msg.Block.VerificationTickets); err != nil {
		Logger.Error("notarized block", zap.Int64("round", msg.Block.Round), zap.String("block", msg.Block.Hash), zap.Error(err))
		return
	}
	b, err := mc.GetBlock(ctx, msg.Block.Hash)
	if err == nil {
		b.MergeVerificationTickets(msg.Block.VerificationTickets)
	} else {
		b = msg.Block
		mc.AddBlock(b)
	}
	mc.AddNotarizedBlock(ctx, &r.Round, b)
	if !r.IsVerificationComplete() {
		r.CancelVerification()
		if r.Block == nil || r.Block.Weight() < b.Weight() {
			r.Block = b
		}
	}
}
