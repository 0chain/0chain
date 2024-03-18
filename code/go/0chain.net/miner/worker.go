package miner

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/util/taskqueue"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/minersc"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

const minerScMinerHealthCheck = "miner_health_check"

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers(ctx context.Context) {
	mc := GetMinerChain()
	go mc.RoundWorker(ctx)              //we are going to start this after we are ready with the round
	go mc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go mc.FinalizeRoundWorker(ctx)      // 2) sequentially finalize the rounds
	go mc.FinalizedBlockWorker(ctx, mc) // 3) sequentially processes finalized blocks
	go mc.ticketVerifyWorker(ctx)

	go mc.SyncLFBStateWorker(ctx)

	go mc.PruneStorageWorker(ctx, time.Minute*5, mc.getPruneCountRoundStorage(), mc.MagicBlockStorage, mc.roundDkg)
	go mc.UpdateMagicBlockWorker(ctx)
	//TODO uncomment it, atm it breaks executing faucet pour somehow
	go mc.MinerHealthCheck(ctx)
	go mc.NotarizationProcessWorker(ctx)
	go mc.BlockVerifyWorkers(ctx)
	go mc.SyncAllMissingNodesWorker(ctx)
}

func (mc *Chain) startMessageWorker(ctx context.Context) {
	var protocol Protocol = mc
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.blockMessageChannel:
			if !mc.isStarted() {
				break
			}

			switch msg.Type {
			case MessageVerificationTicket:
				protocol.HandleVerificationTicketMessage(ctx, msg)
				continue
			case MessageVerify:
				protocol.HandleVerifyBlockMessage(ctx, msg)
				continue
			default:
			}
			// if msg.Type == MessageVerificationTicket {
			// 	protocol.HandleVerificationTicketMessage(ctx, msg)
			// 	continue
			// }

			_ = taskqueue.Execute(taskqueue.Common, func() error {
				func(bmsg *BlockMessage) {
					ts := time.Now()
					if bmsg.Sender != nil {
						logging.Logger.Debug("message",
							zap.Any("msg", GetMessageLookup(bmsg.Type)),
							zap.Int("sender_index", bmsg.Sender.SetIndex),
							zap.String("id", bmsg.Sender.GetKey()))
					} else {
						logging.Logger.Debug("message", zap.Any("msg", GetMessageLookup(bmsg.Type)))
					}

					// taskqueue.Execute(taskqueue.Common, func() error {
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
					// return nil
					// })

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
				return nil
			})
		}
	}
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	// start 10 workers to process the incoming messages
	for i := 0; i < 10; i++ {
		mc.startMessageWorker(ctx)
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
							zap.Int64("round", r.Number),
							zap.Int64("current round", cround),
							zap.Int("VRF_shares", len(r.GetVRFShares())),
							zap.Int("proposedBlocks", len(r.GetProposedBlocks())),
							zap.Int("verificationTickets", len(r.verificationTickets)),
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
	gn, err := minersc.GetGlobalNode(mc.GetQueryStateContext())
	if err != nil {
		logging.Logger.Panic("miner health check - get global node failed", zap.Error(err))
		return
	}

	logging.Logger.Debug("miner health check - start", zap.Any("period", gn.HealthCheckPeriod))
	HEALTH_CHECK_TIMER := gn.HealthCheckPeriod

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
				}
			}()
		}
		time.Sleep(HEALTH_CHECK_TIMER)
	}
}

func (mc *Chain) SyncAllMissingNodesWorker(ctx context.Context) {
	// start in a second, repeat every 30 minutes
	tk := time.NewTicker(time.Second)
	for {
		select {
		case <-tk.C:
			mc.syncAllMissingNodes(ctx)
			// do all missing nodes check and sync every 30 minutes
			// TODO: move the interval to a config file
			tk.Reset(30 * time.Minute)
		case <-ctx.Done():
			logging.Logger.Debug("Sync all missing nodes worker exit!")
			return
		}
	}
}

func (mc *Chain) syncAllMissingNodes(ctx context.Context) {
	// get LFB first
	var (
		lfb = mc.GetLatestFinalizedBlock()
		tk  = time.NewTicker(time.Second)
	)

	for {
		if lfb == nil || lfb.ClientState == nil {
			time.Sleep(10 * time.Second)
			lfb = mc.GetLatestFinalizedBlock()
			continue
		}

		logging.Logger.Debug("sync all missing nodes - start from LFB", zap.Int64("round", lfb.Round))
		break
	}

	var (
		missingNodes []util.Key
	)

	// get all missing nodes from LFB
	for {
		logging.Logger.Debug("sync all missing nodes - loading all missing nodes...")
		var err error
		start := time.Now()
		missingNodes, err = lfb.ClientState.GetAllMissingNodes()
		if err != nil {
			logging.Logger.Error("sync all missing nodes - get all missing nodes failed", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}

		// Record the number of missing nodes and the time it took to acquire them
		mc.MissingNodesStat.Counter.Inc(int64(len(missingNodes)))
		mc.MissingNodesStat.Timer.UpdateSince(start)
		node.Self.Underlying().Info.SetStateMissingNodes(int64(len(missingNodes)))

		logging.Logger.Debug("sync all missing nodes - finish load all missing nodes",
			zap.Int("num", len(missingNodes)))

		mns := make([]string, 0, len(missingNodes))
		for _, n := range missingNodes {
			mns = append(mns, util.ToHex(n))
		}
		mn := strings.Join(mns, "\n")
		err = os.WriteFile("/tmp/missing_nodes.txt", []byte(mn), 0644)
		if err != nil {
			logging.Logger.Error("sync all missing nodes - write missing nodes to file failed", zap.Error(err))
		} else {
			logging.Logger.Debug("sync all missing nodes - write missing nodes to file")
		}
		break
	}

	var (
		batchSize = 100
		batchs    = len(missingNodes) / batchSize
		start     = time.Now()
	)

	for idx := 1; idx <= batchs; idx++ {
		<-tk.C
		// pull missing nodes
		start := (idx - 1) * batchSize
		end := idx * batchSize
		wc := make(chan struct{}, 1)
		mc.SyncMissingNodes(lfb.Round, missingNodes[start:end], wc)
		<-wc
		logging.Logger.Debug("sync all missing nodes - pull missing nodes",
			zap.Int("num", batchSize),
			zap.Int("remaining", len(missingNodes)-end))

		node.Self.Underlying().Info.SetStateMissingNodes(int64(len(missingNodes) - end))
		tk.Reset(2 * time.Second)
	}

	mc.MissingNodesStat.SyncTimer.UpdateSince(start)

	mod := len(missingNodes) % batchSize
	if mod > 0 {
		wc := make(chan struct{}, 1)
		mc.SyncMissingNodes(lfb.Round, missingNodes[batchs*batchSize:], wc)
		<-wc
		logging.Logger.Debug("sync all missing nodes - pull missing nodes",
			zap.Int("num", mod))
	}

	logging.Logger.Debug("sync all missing nodes - done")
}
