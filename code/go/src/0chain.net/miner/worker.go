package miner

import (
	"context"
	"time"

	"0chain.net/logging"
	. "0chain.net/logging"

	"go.uber.org/zap"
)

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.RoundWorker(ctx)
	go mc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx, mc)  // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks
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
		}
	}
}

//RoundWorker - a worker that monitors the round progress
func (mc *Chain) RoundWorker(ctx context.Context) {
	var cround = mc.CurrentRound
	var ticker = time.NewTicker(time.Second)
	var tickerCount = 0
	var protocol Protocol = mc
	for true {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if cround == mc.CurrentRound {
				round := mc.GetMinerRound(cround)
				tickerCount++

				//Something bad happened.
				/*
					n := node.Self
					common.LogRuntime(logging.MemUsage, zap.Any(n.Description, n.SetIndex))
					buf := new(bytes.Buffer)
					pprof.Lookup("goroutine").WriteTo(buf, 1)
					logging.Logger.Info("Round timeout", zap.String("Go routine output", buf.String()))
				*/

				logging.Logger.Info("Round timeout", zap.Any("Number", round.Number),
					zap.Int("#of VRF_shares", len(round.GetVRFShares())),
					zap.Int("#of proposedBlocks", len(round.GetProposedBlocks())),
					zap.Int("#of notarizedBlocks", len(round.GetNotarizedBlocks())))
				protocol.HandleRoundTimeout(ctx, tickerCount)
			} else {
				cround = mc.CurrentRound
				mc.ResetRoundTimeoutCount()
				tickerCount = 0
			}
		}
	}
}
