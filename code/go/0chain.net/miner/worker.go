package miner

import (
	"context"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/smartcontract/minersc"
	"github.com/spf13/viper"

	"0chain.net/core/logging"
	. "0chain.net/core/logging"

	"go.uber.org/zap"
)

const minerScMinerHealthCheck = "miner_health_check"

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.RoundWorker(ctx)              //we are going to start this after we are ready with the round
	go mc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx, mc)  // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks

	go mc.PruneStorageWorker(ctx, time.Minute*5, mc.getPruneCountRoundStorage(), mc.MagicBlockStorage, mc.roundDkg)
	go mc.UpdateMagicBlockWorker(ctx)
	go mc.MinerHealthCheck(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	var protocol Protocol = mc

	for true {
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.GetBlockMessageChannel():
			if !mc.isStarted() {
				break
			}
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

	var (
		timer             = time.NewTimer(4 * time.Second)
		cround            = mc.GetCurrentRound()
		protocol Protocol = mc
	)

	for true {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !mc.isStarted() {
				break
			}
			if cround == mc.GetCurrentRound() {
				round := mc.GetMinerRound(cround)

				if round != nil {
					logging.Logger.Info("Round timeout",
						zap.Any("Number", round.Number),
						zap.Int("VRF_shares", len(round.GetVRFShares())),
						zap.Int("proposedBlocks", len(round.GetProposedBlocks())),
						zap.Int("notarizedBlocks", len(round.GetNotarizedBlocks())))
					protocol.HandleRoundTimeout(ctx, cround)
				}
			} else {
				cround = mc.GetCurrentRound()
				mc.ResetRoundTimeoutCount()
			}
		}
		var next = mc.GetNextRoundTimeoutTime(ctx)
		Logger.Info("got_timeout", zap.Int("next", next))
		timer = time.NewTimer(time.Duration(next) * time.Millisecond)
	}
}

func (mc *Chain) RestartRoundEventWorker(ctx context.Context) {

	// restart round events subscribers

	var (
		subs = make(map[chan struct{}]struct{})

		subq   = mc.subRestartRoundEventChannel
		unsubq = mc.unsubRestartRoundEventChannel
		rrq    = mc.restartRoundEventChannel

		doneq = ctx.Done()
	)

	defer close(mc.restartRoundEventWorkerIsDoneChannel)

	for {
		select {
		case <-doneq:
			return
		case ch := <-subq:
			subs[ch] = struct{}{}
		case ch := <-unsubq:
			delete(subs, ch)
		case <-rrq:
			for ch := range subs {
				select {
				case ch <- struct{}{}: // trigger for the subscriber
				default: // non-blocking
				}
			}
		}
	}
}

func (mc *Chain) getPruneCountRoundStorage() func(storage round.RoundStorage) int {
	viper.SetDefault("server_chain.round_magic_block_storage.prune_below_count", chain.DefaultCountPruneRoundStorage)
	viper.SetDefault("server_chain.round_dkg_storage.prune_below_count", chain.DefaultCountPruneRoundStorage)
	pruneBelowCountMB := viper.GetInt("server_chain.round_magic_block_storage.prune_below_count")
	pruneBelowCountDKG := viper.GetInt("server_chain.round_dkg_storage.prune_below_count")
	return func(storage round.RoundStorage) int {
		switch storage {
		case mc.roundDkg:
			return pruneBelowCountDKG
		case mc.MagicBlockStorage:
			return pruneBelowCountMB
		default:
			return chain.DefaultCountPruneRoundStorage
		}
	}
}

func (mc *Chain) MinerHealthCheck(ctx context.Context) {
	const HEALTH_CHECK_TIMER = 60 * 5 // 5 Minute
	for {
		select {
		case <-ctx.Done():
			return
		default:
			selfNode := node.Self.Underlying()
			txn := httpclientutil.NewTransactionEntity(selfNode.GetKey(), mc.ID, selfNode.PublicKey)
			scData := &httpclientutil.SmartContractTxnData{}
			scData.Name = minerScMinerHealthCheck

			txn.ToClientID = minersc.ADDRESS
			txn.PublicKey = selfNode.PublicKey

			mb := mc.GetCurrentMagicBlock()
			var minerUrls = mb.Miners.N2NURLs()
			go httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
		}
		time.Sleep(HEALTH_CHECK_TIMER * time.Second)
	}
}
