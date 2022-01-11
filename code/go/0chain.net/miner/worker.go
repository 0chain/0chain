package miner

import (
	"context"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/logging"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/minersc"
)

const minerScMinerHealthCheck = "miner_health_check"

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.RoundWorker(ctx)              //we are going to start this after we are ready with the round
	go mc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx)      // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks

	go mc.PruneStorageWorker(ctx, time.Minute*5, mc.getPruneCountRoundStorage(), mc.MagicBlockStorage, mc.roundDkg)
	go mc.UpdateMagicBlockWorker(ctx)
	go mc.MinerHealthCheck(ctx)
	go mc.NotarizationProcessWorker(ctx)
	go mc.BlockVerifyWorkers(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	var protocol Protocol = mc

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.blockMessageChannel:
			if !mc.isStarted() {
				break
			}
			go func(bmsg *BlockMessage) {
				ts := time.Now()
				if bmsg.Sender != nil {
					logging.Logger.Debug("message",
						zap.Any("msg", GetMessageLookup(bmsg.Type)),
						zap.Any("sender_index", bmsg.Sender.SetIndex),
						zap.Any("id", bmsg.Sender.GetKey()))
				} else {
					logging.Logger.Debug("message", zap.Any("msg", GetMessageLookup(bmsg.Type)))
				}
				switch bmsg.Type {
				case MessageVRFShare:
					protocol.HandleVRFShare(ctx, bmsg)
				case MessageVerify:
					protocol.HandleVerifyBlockMessage(ctx, bmsg)
				case MessageVerificationTicket:
					protocol.HandleVerificationTicketMessage(ctx, bmsg)
				case MessageNotarization:
					protocol.HandleNotarizationMessage(ctx, bmsg)
				case MessageNotarizedBlock:
					protocol.HandleNotarizedBlockMessage(ctx, bmsg)
				}
				if bmsg.Sender != nil {
					logging.Logger.Debug("message (done)",
						zap.Any("msg", GetMessageLookup(bmsg.Type)),
						zap.Any("sender_index", bmsg.Sender.SetIndex),
						zap.Any("id", bmsg.Sender.GetKey()),
						zap.Any("duration", time.Since(ts)))
				} else {
					logging.Logger.Debug("message (done)",
						zap.Any("msg", GetMessageLookup(bmsg.Type)),
						zap.Any("duration", time.Since(ts)))
				}
			}(msg)
		}
	}
}

func roundTimeoutProcess(ctx context.Context, proto Protocol, rn int64) {
	var cancel func()
	ctx, cancel = context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	rc := make(chan struct{})
	ts := time.Now()
	go func() {
		proto.HandleRoundTimeout(ctx, rn)
		close(rc)
	}()

	select {
	case <-ctx.Done():
		logging.Logger.Error("protocol.HandleRoundTimeout timeout",
			zap.Error(ctx.Err()),
			zap.Int64("round", rn))
	case <-rc:
		logging.Logger.Info("protocol.HandleRoundTimeout finished",
			zap.Int64("round", rn),
			zap.Any("duration", time.Since(ts)))
	}
}

//RoundWorker - a worker that monitors the round progress
func (mc *Chain) RoundWorker(ctx context.Context) {

	var (
		timer             = time.NewTimer(4 * time.Second)
		cround            = mc.GetCurrentRound()
		protocol Protocol = mc
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !mc.isStarted() {
				break
			}

			if cround == mc.GetCurrentRound() {
				r := mc.GetMinerRound(cround)

				if r != nil {
					if r.IsFinalized() || r.IsFinalizing() {
						// check next round
						nr := mc.GetRound(cround + 1)
						if nr != nil {
							roundTimeoutProcess(ctx, protocol, cround+1)
						}
					} else {
						logging.Logger.Info("round timeout",
							zap.Any("round", r.Number),
							zap.Any("current round", cround),
							zap.Int("VRF_shares", len(r.GetVRFShares())),
							zap.Int("proposedBlocks", len(r.GetProposedBlocks())),
							zap.Int("notarizedBlocks", len(r.GetNotarizedBlocks())))
						roundTimeoutProcess(ctx, protocol, cround)
					}
				} else {
					// set current round to latest finalized block
					lfbr := mc.GetLatestFinalizedBlock().Round
					mc.SetCurrentRound(lfbr)
					logging.Logger.Debug("Round timeout, nil miner round, set current round to lfb round",
						zap.Int64("nil round", cround),
						zap.Int64("lfb round", lfbr))
				}
			} else {
				cround = mc.GetCurrentRound()
				mc.ResetRoundTimeoutCount()
			}
		}
		var next = mc.GetNextRoundTimeoutTime(ctx)
		logging.Logger.Info("got_timeout", zap.Int("next", next))
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
