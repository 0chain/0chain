package miner

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

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

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx, mc)  // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks
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
			if msg.Sender != nil {
				Logger.Debug("message", zap.Any("msg", GetMessageLookup(msg.Type)), zap.Any("sender_index", msg.Sender.SetIndex), zap.Any("id", msg.Sender.GetKey()))
			} else {
				Logger.Debug("message", zap.Any("msg", GetMessageLookup(msg.Type)))
			}
			switch msg.Type {
			case MessageVRFShare:
				roundTimeout.Stop()
				protocol.HandleVRFShare(ctx, msg)
			case MessageVerify:
				roundTimeout.Stop()
				protocol.HandleVerifyBlockMessage(ctx, msg)
			case MessageVerificationTicket:
				roundTimeout.Stop()
				protocol.HandleVerificationTicketMessage(ctx, msg)
			case MessageNotarization:
				roundTimeout.Stop()
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

/*StartProtocol -- Start protocol as a worker */
func StartProtocol() {
	ctx := common.GetRootContext()
	mc := GetMinerChain()
	lfb := getLatestBlockFromSharders(ctx)
	sr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
	if lfb != nil {
		mc.SetLatestFinalizedBlock(ctx, lfb)
		sr.Number = lfb.Round + 1
		sr.RandomSeed = rand.New(rand.NewSource(lfb.RoundRandomSeed)).Int63()
	} else {
		sr.Number = 1
	}
	msr := mc.CreateRound(sr)

	Logger.Info("starting the blockchain ...")
	mc.StartRound(ctx, msr)
}
