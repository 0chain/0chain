package miner

import (
	"context"
	"sort"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	. "0chain.net/logging"

	"0chain.net/round"
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
				tickerCount++
				protocol.HandleRoundTimeout(ctx, tickerCount)
			} else {
				cround = mc.CurrentRound
				mc.ResetRoundTimeoutCount()
				tickerCount = 0
			}
		}
	}
}

func getLatestBlockFromSharders(ctx context.Context) *block.Block {
	mc := GetMinerChain()
	mc.Sharders.OneTimeStatusMonitor(ctx)
	lfBlocks := mc.GetLatestFinalizedBlockFromSharder(ctx)
	//Sorting as per the latest finalized blocks from all the sharders
	sort.Slice(lfBlocks, func(i int, j int) bool { return lfBlocks[i].Round >= lfBlocks[j].Round })
	if len(lfBlocks) > 0 {
		Logger.Info("bc-1 latest finalized Block", zap.Int64("lfb_round", lfBlocks[0].Round))
		return lfBlocks[0]
	}
	Logger.Info("bc-1 sharders returned no lfb.")
	return nil
}

/*StartProtocol -- Start protocol as a worker */
func StartProtocol() {

	ctx := common.GetRootContext()

	mc := GetMinerChain()

	mc.Sharders.OneTimeStatusMonitor(ctx)
	lfb := getLatestBlockFromSharders(ctx)
	var mr *Round
	if lfb != nil {
		sr := round.NewRound(lfb.Round)
		mr = mc.CreateRound(sr)
		mr, _ = mc.AddRound(mr).(*Round)
		mc.SetRandomSeed(sr, lfb.RoundRandomSeed)
		mc.SetLatestFinalizedBlock(ctx, lfb)

	} else {
		sr := round.NewRound(0)
		mr = mc.CreateRound(sr)
	}

	Logger.Info("starting the blockchain ...", zap.Int64("round", mr.GetRoundNumber()))
	mc.StartNextRound(ctx, mr)
}
