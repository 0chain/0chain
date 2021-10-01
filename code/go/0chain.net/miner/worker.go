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
	go mc.BlockSyncWorker(ctx)

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
		case msg := <-mc.blockMessageChannel:
			if !mc.isStarted() {
				break
			}
			go func(bmsg *BlockMessage) {
				if bmsg.Sender != nil {
					logging.Logger.Debug("message", zap.Any("msg", GetMessageLookup(bmsg.Type)), zap.Any("sender_index", bmsg.Sender.SetIndex), zap.Any("id", bmsg.Sender.GetKey()))
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
					logging.Logger.Debug("message (done)", zap.Any("msg", GetMessageLookup(bmsg.Type)), zap.Any("sender_index", bmsg.Sender.SetIndex), zap.Any("id", bmsg.Sender.GetKey()))
				} else {
					logging.Logger.Debug("message (done)", zap.Any("msg", GetMessageLookup(bmsg.Type)))
				}
			}(msg)
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
				r := mc.GetMinerRound(cround)

				if r != nil {
					logging.Logger.Info("Round timeout",
						zap.Any("round", r.Number),
						zap.Any("current round", cround),
						zap.Int("VRF_shares", len(r.GetVRFShares())),
						zap.Int("proposedBlocks", len(r.GetProposedBlocks())),
						zap.Int("notarizedBlocks", len(r.GetNotarizedBlocks())))
					func(ctx context.Context) {
						cctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
						defer cancel()
						rc := make(chan struct{})
						ts := time.Now()
						go func() {
							protocol.HandleRoundTimeout(cctx, cround)
							close(rc)
						}()

						select {
						case <-cctx.Done():
							logging.Logger.Error("protocol.HandleRoundTimeout timeout",
								zap.Error(cctx.Err()),
								zap.Int64("round", cround))
						case <-rc:
							logging.Logger.Info("protocol.HandleRoundTimeout finished",
								zap.Int64("round", cround),
								zap.Any("duration", time.Since(ts)))
						}
					}(ctx)
				} else {
					// set current round to latet finalized block
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

// BlockSyncWorker pull and sync blocks from remote
func (mc *Chain) BlockSyncWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-mc.GetSyncBlocksChan():
			func() {
				tm := time.Now()
				b, _ := mc.GetBlock(ctx, req.Hash)
				if b == nil {
					logging.Logger.Warn("[block sync worker] - start block does not exist locally",
						zap.String("block", req.Hash),
						zap.Int64("round", req.Round))
					return
				}
				syncNum := req.Num
				// use half of the PruneStateBelowCount, usually it would be 50,
				maxSyncNum := int64(mc.PruneStateBelowCount / 2)
				if syncNum > maxSyncNum {
					syncNum = maxSyncNum
				}

				blocks := mc.SyncBlocks(ctx, b, syncNum, req.SaveToDB)
				logging.Logger.Debug("[block sync worker] - blocks synced",
					zap.Int64("start round", b.Round-1),
					zap.Int64("req sync num", syncNum),
					zap.Int("synced num", len(blocks)),
					zap.Bool("save to db", req.SaveToDB),
					zap.Any("duration", time.Since(tm)))
			}()
		}
	}
}
