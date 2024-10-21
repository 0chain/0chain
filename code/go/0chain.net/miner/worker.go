package miner

import (
	"context"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/minersc"
	"github.com/0chain/common/core/logging"
)

const minerScMinerHealthCheck = "miner_health_check"

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.RoundWorker(ctx)              //we are going to start this after we are ready with the round
	go mc.MessageWorker(ctx)            // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx)      // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks
	go mc.BlockWorker(ctx)              // 4) sync blocks when stuck

	go mc.SyncLFBStateWorker(ctx)

	go mc.PruneStorageWorker(ctx, time.Minute*5, mc.getPruneCountRoundStorage(), mc.MagicBlockStorage, mc.GetRoundDkg())
	//TODO uncomment it, atm it breaks executing faucet pour somehow
	go mc.MinerHealthCheck(ctx)
	go mc.NotarizationProcessWorker(ctx)
	go mc.BlockVerifyWorkers(ctx)
}

/*MessageWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) MessageWorker(ctx context.Context) {
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
						zap.Int("sender_index", bmsg.Sender.SetIndex),
						zap.String("id", bmsg.Sender.GetKey()))
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
						zap.Int("sender_index", bmsg.Sender.SetIndex),
						zap.String("id", bmsg.Sender.GetKey()),
						zap.Duration("duration", time.Since(ts)))
				} else {
					logging.Logger.Debug("message (done)",
						zap.Any("msg", GetMessageLookup(bmsg.Type)),
						zap.Duration("duration", time.Since(ts)))
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
			zap.Duration("duration", time.Since(ts)))
	}
}

// RoundWorker - a worker that monitors the round progress
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
		case nr := <-mc.GetNotifyMoveToNextRoundC():
			roundNum := nr.GetRoundNumber()
			logging.Logger.Debug("notify move to next round",
				zap.Int64("round", roundNum),
				zap.Int64("next round", roundNum+1))
			if mr := mc.GetMinerRound(nr.GetRoundNumber()); mr != nil {
				mc.ProgressOnNotarization(mr)
			}
		case <-timer.C:
			if !mc.isStarted() {
				break
			}

			if cround == mc.GetCurrentRound() {
				r := mc.GetMinerRound(cround)

				if r != nil {
					if r.IsFinalized() || r.IsFinalizing() {
						logging.Logger.Info("round worker: round is finalized or finalizing, check next round",
							zap.Int64("round", cround))

						// check next round
						nr := mc.GetRound(cround + 1)
						if nr != nil {
							roundTimeoutProcess(ctx, protocol, cround+1)
						} else {
							logging.Logger.Info("round worker: next round is nil", zap.Int64("next round", cround+1))
						}
					} else {
						logging.Logger.Info("round worker: round timeout",
							zap.Int64("round", r.Number),
							zap.Int64("current round", cround),
							zap.Int("VRF_shares", len(r.GetVRFShares())),
							zap.Int("proposedBlocks", len(r.GetProposedBlocks())),
							zap.Int("verificationTickets", len(r.verificationTickets)),
							zap.Int("notarizedBlocks", len(r.GetNotarizedBlocks())))
						roundTimeoutProcess(ctx, protocol, cround)
					}

					lfb := mc.GetLatestFinalizedBlock()
					lfbTk := mc.GetLatestLFBTicket(ctx)
					if lfb.Round < lfbTk.Round {
						logging.Logger.Info("round worker: LFB < latest lfb ticket round, notify block sync",
							zap.Int64("lfb round", lfb.Round),
							zap.Int64("lfb ticket round", lfbTk.Round),
							zap.Int64("current round", cround))
						mc.NotifyBlockSync()
					}
				} else {
					// set current round to latest finalized block
					// lfbr := mc.GetLatestFinalizedBlock().Round
					// mc.SetCurrentRound(lfbr)
					// logging.Logger.Debug("round worker: Round timeout, nil miner round, set current round to lfb round",
					// 	zap.Int64("nil round", cround),
					// 	zap.Int64("lfb round", lfbr))
					logging.Logger.Warn("round worker: Round timeout, nil miner round", zap.Int64("nil round", cround))
				}
			} else {
				cround = mc.GetCurrentRound()
				mc.ResetRoundTimeoutCount()
			}
		}
		var next = mc.GetNextRoundTimeoutTime(ctx)
		logging.Logger.Info("round worker: got_timeout", zap.Int("next", next))
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
		case mc.GetRoundDkg():
			return pruneBelowCountDKG
		case mc.MagicBlockStorage:
			return pruneBelowCountMB
		default:
			return chain.DefaultCountPruneRoundStorage
		}
	}
}

func (mc *Chain) MinerHealthCheck(ctx context.Context) {
	gn, err := minersc.GetGlobalNode(mc.GetQueryStateContext())
	if err != nil {
		logging.Logger.Panic("miner health check - get global node failed", zap.Error(err))
		return
	}

	gnb := gn.MustBase()
	logging.Logger.Debug("miner health check - start", zap.Any("period", gnb.HealthCheckPeriod))
	HEALTH_CHECK_TIMER := gnb.HealthCheckPeriod

	for {
		select {
		case <-ctx.Done():
			return
		default:
			selfNode := node.Self.Underlying()
			txn := httpclientutil.NewSmartContractTxn(selfNode.GetKey(), mc.ID, selfNode.PublicKey, minersc.ADDRESS)
			scData := &httpclientutil.SmartContractTxnData{}
			scData.Name = minerScMinerHealthCheck

			mb := mc.GetCurrentMagicBlock()
			var minerUrls = mb.Miners.N2NURLs()
			go func() {
				if err := mc.SendSmartContractTxn(txn, scData, minerUrls, mb.Sharders.N2NURLs()); err != nil {
					logging.Logger.Warn("miner health check -  send smart contract failed",
						zap.Int("urls len", len(minerUrls)),
						zap.Error(err))
					return
				}

				mc.ConfirmTransaction(ctx, txn, 30)
			}()
		}
		time.Sleep(HEALTH_CHECK_TIMER)
	}
}
