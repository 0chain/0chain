package miner

import (
	"context"
	"time"

	. "0chain.net/logging"
	"0chain.net/node"
	"go.uber.org/zap"
)

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx, mc)  // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks
	go mc.NodeStatusWorker(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	var RoundTimeout = 10 * time.Second
	var protocol Protocol = mc
	var cround = mc.CurrentRound
	var roundTimeout = time.NewTicker(RoundTimeout)
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
			case MessageVRFShare:
				protocol.HandleVRFShare(ctx, msg)
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
			if cround == mc.CurrentRound {
				protocol.HandleRoundTimeout(ctx)
			} else {
				cround = mc.CurrentRound
			}
		}
	}
}

func (mc *Chain) NodeStatusWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case n := <-node.Self.NodeStatusChannel:
			node.Self.ActiveNodes[n.ID] = n
			/* if n.Type == node.NodeTypeMiner {
				Logger.Info("New Active Miner @Index", zap.Int("idx", n.SetIndex))

			} else if n.Type == node.NodeTypeSharder {
				Logger.Info("New Active Sharder @Index", zap.Int("idx", n.SetIndex))
			} */
		}
	}
}
