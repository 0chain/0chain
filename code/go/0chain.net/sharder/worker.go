package sharder

import (
	"context"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/minersc"

	"github.com/0chain/common/core/logging"
)

const minerScSharderHealthCheck = "sharder_health_check"

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go sc.FinalizeRoundWorker(ctx)      // 2) sequentially finalize the rounds
	go sc.FinalizedBlockWorker(ctx, sc) // 3) sequentially processes finalized blocks

	go sc.SyncLFBStateWorker(ctx)

	// Setup the deep and proximity scan
	go sc.HealthCheckSetup(ctx, DeepScan)
	go sc.HealthCheckSetup(ctx, ProximityScan)

	go sc.PruneStorageWorker(ctx, time.Minute*5, sc.getPruneCountRoundStorage(),
		sc.MagicBlockStorage)
	go sc.RegisterSharderKeepWorker(ctx)
	go sc.SharderHealthCheck(ctx)

	go sc.TrackTransactionErrors(ctx)
	go sc.MBDiscoveryWorker(ctx)
}

func (sc *Chain) hasRoundSummary(ctx context.Context, rNum int64) (*round.Round, bool) {
	r, err := sc.GetRoundFromStore(ctx, rNum)
	if err == nil && sc.isValidRound(r) {
		return r, true
	}
	return nil, false
}

func (sc *Chain) hasBlockSummary(ctx context.Context, bHash string) (*block.BlockSummary, bool) {
	bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx, bSummaryEntityMetadata)
	bs, err := sc.GetBlockSummary(bctx, bHash)
	if err == nil {
		return bs, true
	}
	return nil, false
}

func (sc *Chain) hasBlock(bHash string, rNum int64) (*block.Block, bool) {
	b, err := sc.GetBlockFromStore(bHash, rNum)
	if err == nil {
		return b, true
	}
	return nil, false
}

func (sc *Chain) hasBlockTransactions(ctx context.Context, b *block.Block) bool { //nolint
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := ememorystore.WithEntityConnection(ctx, txnSummaryEntityMetadata)
	defer ememorystore.Close(tctx, txnSummaryEntityMetadata)
	for _, txn := range b.Txns {
		_, err := sc.GetTransactionSummary(tctx, txn.Hash)
		if err != nil {
			return false
		}
	}
	return true
}

func (sc *Chain) RegisterSharderKeepWorker(ctx context.Context) {
	var (
		phaseq = sc.PhaseEvents()
		pe     chain.PhaseEvent //

		phaseRound int64 // starting round of latest accepted phase
	)

	for {
		select {
		case <-ctx.Done():
		default:
			if !sc.ChainConfig.IsViewChangeEnabled() {
				// don't send sharder_keep if view_change is false
				time.Sleep(time.Second)
				continue
			}

			pei, ok := phaseq.Pop()
			if !ok {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			pe = pei.Data.(chain.PhaseEvent)
		}

		if pe.Phase.StartRound < phaseRound {
			continue
		}

		if pe.Phase.Phase != minersc.Contribute {
			phaseRound = pe.Phase.StartRound
			continue // we are interesting in contribute phase only on sharders
		}

		if sc.IsRegisteredSharderKeep(context.Background(), false) {
			phaseRound = pe.Phase.StartRound // already registered
			continue
		}

		logging.Logger.Debug("[mvc] register_sharder_keep_worker - start to register to sharder keep list")
		var txn, err = sc.RegisterSharderKeep()
		if err != nil {
			logging.Logger.Error("[mvc] register_sharder_keep_worker - register sharder keep failed",
				zap.Int64("phase start round", pe.Phase.StartRound),
				zap.Int64("phase current round", pe.Phase.CurrentRound),
				zap.Error(err))
			continue // repeat next time
		}

		if !sc.ConfirmTransaction(ctx, txn, 30) {
			logging.Logger.Debug("[mvc] register_sharder_keep_worker - register sharder keep txn failed",
				zap.Any("txn", txn))
			continue
		}

		logging.Logger.Info("[mvc] register_sharder_keep_worker - register success")
		phaseRound = pe.Phase.StartRound // accepted
	}
}

func (sc *Chain) getPruneCountRoundStorage() func(storage round.RoundStorage) int {
	viper.SetDefault("server_chain.round_magic_block_storage.prune_below_count", chain.DefaultCountPruneRoundStorage)
	pruneBelowCountMB := viper.GetInt("server_chain.round_magic_block_storage.prune_below_count")
	return func(storage round.RoundStorage) int {
		switch storage {
		case sc.MagicBlockStorage:
			return pruneBelowCountMB
		default:
			return chain.DefaultCountPruneRoundStorage
		}
	}
}

func (sc *Chain) SharderHealthCheck(ctx context.Context) {
	gn, err := minersc.GetGlobalNode(sc.GetQueryStateContext())
	if err != nil {
		logging.Logger.Error("sharder health check - get global node failed, retrying later", zap.Error(err))
		return
	}

	gnb := gn.MustBase()
	logging.Logger.Debug("sharder health check - start", zap.Any("period", gnb.HealthCheckPeriod))
	HEALTH_CHECK_TIMER := gnb.HealthCheckPeriod

	for {
		select {
		case <-ctx.Done():
			return
		default:
			selfNode := node.Self.Underlying()
			txn := httpclientutil.NewSmartContractTxn(selfNode.GetKey(), sc.ID, selfNode.PublicKey, minersc.ADDRESS)
			scData := &httpclientutil.SmartContractTxnData{}
			scData.Name = minerScSharderHealthCheck

			mb := sc.GetCurrentMagicBlock()
			var minerUrls = mb.Miners.N2NURLs()
			go func() {
				if err := sc.SendSmartContractTxn(txn, scData, minerUrls, mb.Sharders.N2NURLs()); err != nil {
					logging.Logger.Warn("sharder health check failed, try again")
					return
				}

				sc.ConfirmTransaction(ctx, txn, 30)
			}()

		}
		time.Sleep(HEALTH_CHECK_TIMER)
	}
}

func (sc *Chain) TrackTransactionErrors(ctx context.Context) {
	var (
		timerDuration     = 1 * time.Hour
		timer             = time.NewTimer(timerDuration)
		edb               = sc.GetQueryStateContext().GetEventDB()
		permanentInterval = edb.Settings().PermanentPartitionChangePeriod
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(timerDuration)

			currentRound := sc.GetCurrentRound()
			err := edb.UpdateTransactionErrors(currentRound / permanentInterval)
			if err != nil {
				logging.Logger.Error("TrackTransactionErrors: ", zap.Error(err))
			}
		}
	}
}

// MBDiscoveryWorker periodically checks if the sharder is behind on magic blocks
// and discovers newer MBs from miners. This handles the case where a view change
// completes on miners but the sharder doesn't adopt the new MB, causing the chain
// to stall because the sharder can't validate blocks signed under the new DKG.
func (sc *Chain) MBDiscoveryWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastLFBRound int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lfb := sc.GetLatestFinalizedBlock()
			if lfb == nil {
				continue
			}

			// Only trigger discovery when the chain is stuck (LFB hasn't advanced)
			if lfb.Round != lastLFBRound {
				lastLFBRound = lfb.Round
				continue
			}

			// Chain is stuck — first check if we have a stored MB that's newer
			// than the active one. This happens when the sharder loads an LFB
			// whose round is before the latest MB's starting round, causing it
			// to downgrade the active MB during startup.
			latestStoredMB := sc.GetLatestMagicBlock()
			if latestStoredMB == nil {
				continue
			}

			lfmb := sc.GetLatestFinalizedMagicBlock(context.Background())
			activeMB := lfmb.MagicBlock
			if activeMB != nil && latestStoredMB.MagicBlockNumber > activeMB.MagicBlockNumber {
				logging.Logger.Info("mb_discovery_worker - activating stored MB that's newer than active MB",
					zap.Int64("active_mb_number", activeMB.MagicBlockNumber),
					zap.Int64("active_mb_sr", activeMB.StartingRound),
					zap.Int64("stored_mb_number", latestStoredMB.MagicBlockNumber),
					zap.Int64("stored_mb_sr", latestStoredMB.StartingRound),
					zap.Int64("lfb_round", lfb.Round))

				syntheticBlock := block.NewBlock("", latestStoredMB.StartingRound)
				syntheticBlock.MagicBlock = latestStoredMB
				syntheticBlock.MagicBlockNumber = latestStoredMB.MagicBlockNumber
				sc.SetLatestFinalizedMagicBlock(syntheticBlock)
				continue
			}

			logging.Logger.Info("mb_discovery_worker - chain stuck, checking for newer MBs from miners",
				zap.Int64("lfb_round", lfb.Round),
				zap.Int64("current_mb_number", latestStoredMB.MagicBlockNumber),
				zap.Int64("current_mb_sr", latestStoredMB.StartingRound))

			sc.discoverNewerMBsFromSharders(ctx, latestStoredMB.MagicBlockNumber, lfb.Round)
		}
	}
}
