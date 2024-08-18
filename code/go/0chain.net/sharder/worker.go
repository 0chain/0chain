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
	go sc.UpdateMagicBlockWorker(ctx)
	go sc.RegisterSharderKeepWorker(ctx)
	go sc.SharderHealthCheck(ctx)

	go sc.TrackTransactionErrors(ctx)
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
	defer ememorystore.Close(bctx)
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
	defer ememorystore.Close(tctx)
	for _, txn := range b.Txns {
		_, err := sc.GetTransactionSummary(tctx, txn.Hash)
		if err != nil {
			return false
		}
	}
	return true
}

func (sc *Chain) RegisterSharderKeepWorker(ctx context.Context) {
	if !sc.ChainConfig.IsViewChangeEnabled() {
		return // don't send sharder_keep if view_change is false
	}

	// common register sharder keep constants
	const (
		repeat = 5 * time.Second // repeat every 5 seconds
	)

	var (
		ticker = time.NewTicker(repeat)
		phaseq = sc.PhaseEvents()
		pe     chain.PhaseEvent //
		latest time.Time        // last time phase updated by the node itself

		phaseRound int64 // starting round of latest accepted phase
	)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case tp := <-ticker.C:
			if tp.Sub(latest) < repeat || phaseq.Size() > 0 {
				continue // already have a fresh phase
			}
			go sc.GetPhaseFromSharders(ctx)
			continue
		default:
			pei, ok := phaseq.Pop()
			if !ok {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			pe = pei.Data.(chain.PhaseEvent)
			if !pe.Sharders {
				latest = time.Now()
			}
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

		logging.Logger.Debug("Start to register to sharder keep list")
		var txn, err = sc.RegisterSharderKeep()
		if err != nil {
			logging.Logger.Error("Register sharder keep failed",
				zap.Int64("phase start round", pe.Phase.StartRound),
				zap.Int64("phase current round", pe.Phase.CurrentRound),
				zap.Error(err))
			continue // repeat next time
		}

		// so, transaction sent, let's verify it

		if !sc.ConfirmTransaction(ctx, txn, 0) {
			logging.Logger.Debug("register_sharder_keep_worker -- failed "+
				"to confirm transaction", zap.Any("txn", txn))
			continue
		}

		logging.Logger.Info("register_sharder_keep_worker -- registered")
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
		logging.Logger.Panic("sharder health check - get global node failed", zap.Error(err))
		return
	}

	logging.Logger.Debug("sharder health check - start", zap.Any("period", gn.HealthCheckPeriod))
	HEALTH_CHECK_TIMER := gn.HealthCheckPeriod

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
			if err := sc.SendSmartContractTxn(txn, scData, minerUrls, mb.Sharders.N2NURLs()); err != nil {
				logging.Logger.Warn("sharder health check failed, try again")
			}

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
