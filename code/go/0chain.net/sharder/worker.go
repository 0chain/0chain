package sharder

import (
	"context"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/sharder/blockstore"

	"0chain.net/chaincore/httpclientutil"
	"0chain.net/smartcontract/minersc"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"

	"github.com/remeh/sizedwaitgroup"
	"github.com/spf13/viper"

	"go.uber.org/zap"

	. "0chain.net/core/logging"
)

const minerScSharderHealthCheck = "sharder_health_check"

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go sc.FinalizeRoundWorker(ctx, sc)  // 2) sequentially finalize the rounds
	go sc.FinalizedBlockWorker(ctx, sc) // 3) sequentially processes finalized blocks

	// Setup the deep and proximity scan
	go sc.HealthCheckSetup(ctx, DeepScan)
	go sc.HealthCheckSetup(ctx, ProximityScan)

	go sc.PruneStorageWorker(ctx, time.Minute*5, sc.getPruneCountRoundStorage(),
		sc.MagicBlockStorage)
	go sc.UpdateMagicBlockWorker(ctx)
	go sc.RegisterSharderKeepWorker(ctx)
	// Move old blocks to cloud
	if viper.GetBool("minio.enabled") {
		go sc.MinioWorker(ctx)
	}

	go sc.SharderHealthCheck(ctx)
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-sc.GetBlockChannel():
			sc.processBlock(ctx, b)
		}
	}
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

func (sc *Chain) hasBlockTransactions(ctx context.Context, b *block.Block) bool {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryEntityMetadata)
	defer persistencestore.Close(tctx)
	for _, txn := range b.Txns {
		_, err := sc.GetTransactionSummary(tctx, txn.Hash)
		if err != nil {
			return false
		}
	}
	return true
}

func (sc *Chain) RegisterSharderKeepWorker(ctx context.Context) {

	if !config.DevConfiguration.ViewChange {
		return // don't send sharder_keep if view_change is false
	}

	// common register sharder keep constants
	const (
		repeat = 5 * time.Second // repeat every 5 seconds
	)

	var (
		ticker = time.NewTicker(repeat)

		tickerq = ticker.C
		phaseq  = sc.PhaseEvents()
		doneq   = ctx.Done()

		pe     chain.PhaseEvent //
		latest time.Time        // last time phase updated by the node itself

		phaseRound int64 // starting round of latest accepted phase
	)

	defer ticker.Stop()

	for {
		select {
		case <-doneq:
			return
		case tp := <-tickerq:
			if tp.Sub(latest) < repeat || len(phaseq) > 0 {
				continue // already have a fresh phase
			}
			sc.GetPhaseFromSharders() // not in a goroutine
			continue
		case pe = <-phaseq:
			if !pe.Sharders {
				latest = time.Now()
			}
		}

		if pe.Phase.StartRound == phaseRound {
			continue // the phase already accepted
		}

		if pe.Phase.Phase != minersc.Contribute {
			phaseRound = pe.Phase.StartRound
			continue // we are interesting in contribute phase only on sharders
		}

		if sc.IsRegisteredSharderKeep(false) {
			phaseRound = pe.Phase.StartRound // already registered
			continue
		}

		var txn, err = sc.RegisterSharderKeep()
		if err != nil {
			Logger.Error("register_sharder_keep_worker", zap.Error(err))
			continue // repeat next time
		}

		// so, transaction sent, let's verify it

		if !sc.ConfirmTransaction(txn) {
			Logger.Debug("register_sharder_keep_worker -- failed "+
				"to confirm transaction", zap.Any("txn", txn))
			continue
		}

		Logger.Info("register_sharder_keep_worker -- registered")
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

func (sc *Chain) MinioWorker(ctx context.Context) {
	if !viper.GetBool("minio.enabled") {
		return
	}
	var iterInprogress = false
	var oldBlockRoundRange = viper.GetInt64("minio.old_block_round_range")
	var numWorkers = viper.GetInt("minio.num_workers")
	ticker := time.NewTicker(time.Duration(viper.GetInt64("minio.worker_frequency")) * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !iterInprogress {
				iterInprogress = true
				roundToProcess := sc.CurrentRound - oldBlockRoundRange
				fs := blockstore.GetStore()
				swg := sizedwaitgroup.New(numWorkers)
				for roundToProcess > 0 {
					hash, err := sc.GetBlockHash(ctx, roundToProcess)
					if err != nil {
						Logger.Error("Unable to get block hash from round number", zap.Any("round", roundToProcess))
						roundToProcess--
						continue
					}
					if fs.CloudObjectExists(hash) {
						Logger.Info("The data is already present on cloud, Terminating the worker...", zap.Any("round", roundToProcess))
						break
					} else {
						swg.Add()
						go sc.moveBlockToCloud(ctx, roundToProcess, hash, fs, &swg)
						roundToProcess--
					}
				}
				swg.Wait()
				iterInprogress = false
				Logger.Info("Moved old blocks to cloud successfully")
			}
		}
	}
}

func (sc *Chain) moveBlockToCloud(ctx context.Context, round int64, hash string, fs blockstore.BlockStore, swg *sizedwaitgroup.SizedWaitGroup) {
	err := fs.UploadToCloud(hash, round)
	if err != nil {
		Logger.Error("Error in uploading to cloud, The data is also missing from cloud", zap.Error(err), zap.Any("round", round))
	} else {
		Logger.Info("Block successfully uploaded to cloud", zap.Any("round", round))
		sc.TieringStats.TotalBlocksUploaded++
		sc.TieringStats.LastRoundUploaded = round
		sc.TieringStats.LastUploadTime = time.Now()
	}
	swg.Done()
}

func (sc *Chain) SharderHealthCheck(ctx context.Context) {
	const HEALTH_CHECK_TIMER = 60 * 5 // 5 Minute
	for {
		select {
		case <-ctx.Done():
			return
		default:
			selfNode := node.Self.Underlying()
			txn := httpclientutil.NewTransactionEntity(selfNode.GetKey(), sc.ID, selfNode.PublicKey)
			scData := &httpclientutil.SmartContractTxnData{}
			scData.Name = minerScSharderHealthCheck

			txn.ToClientID = minersc.ADDRESS
			txn.PublicKey = selfNode.PublicKey

			mb := sc.GetCurrentMagicBlock()
			var minerUrls = mb.Miners.N2NURLs()
			go httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
		}
		time.Sleep(HEALTH_CHECK_TIMER * time.Second)
	}
}
