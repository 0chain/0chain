package sharder

import (
	"context"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/sharder/blockstore"

	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"

	"github.com/remeh/sizedwaitgroup"
	"github.com/spf13/viper"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go sc.FinalizeRoundWorker(ctx, sc)  // 2) sequentially finalize the rounds
	go sc.FinalizedBlockWorker(ctx, sc) // 3) sequentially processes finalized blocks

	// Setup the deep and proximity scan
	go sc.HealthCheckSetup(ctx, DeepScan)
	go sc.HealthCheckSetup(ctx, ProximityScan)

	go sc.PruneStorageWorker(ctx, time.Minute*5, sc.getPruneCountRoundStorage(), sc.MagicBlockStorage)
	go sc.RegisterSharderKeepWorker(ctx)
	// Move old blocks to cloud
	if viper.GetBool("minio.enabled") {
		go sc.MoveOldBlocksToCloud(ctx)
	}
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {
	for true {
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
	if err == nil {
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

func (sc *Chain) hasTransactions(ctx context.Context, bs *block.BlockSummary) bool {
	if bs == nil {
		return false
	}
	count, err := sc.getTxnCountForRound(ctx, bs.Round)
	if err != nil {
		return false
	}
	return count == bs.NumTxns
}

func sleepOrDone(ctx context.Context, sleep time.Duration) (done bool) {
	var tm = time.NewTimer(sleep)
	defer tm.Stop()
	select {
	case <-ctx.Done():
		done = true
	case <-tm.C:
	}
	return
}

func (sc *Chain) isPhaseContibute() (is bool) {
	var (
		cstate    = sc.GetLatestFinalizedBlock().ClientState
		seri, err = cstate.GetNodeValue(
			util.Path(encryption.Hash(minersc.PhaseKey)),
		)
	)
	if err != nil {
		Logger.Error("is_phase_contibute -- can't get phase node",
			zap.Error(err))
		return
	}
	var phaseNode = new(minersc.PhaseNode)
	if err = phaseNode.Decode(seri.Encode()); err != nil {
		Logger.Error("is_phase_contibute -- can't decode phase node",
			zap.Error(err))
		return
	}
	Logger.Debug("is_phase_contibute",
		zap.Int("phase", phaseNode.Phase),
		zap.Bool("is_contribute", phaseNode.Phase == minersc.Contribute))
	return phaseNode.Phase == minersc.Contribute
}

func (sc *Chain) RegisterSharderKeepWorker(ctx context.Context) {
	timerCheck := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timerCheck.C:
			if !sc.ActiveInChain() || !sc.IsRegisteredSharderKeep() {
				for !sc.IsRegisteredSharderKeep() {

					for !sc.isPhaseContibute() {
						if sleepOrDone(ctx, time.Second) {
							return
						}
					}

					txn, err := sc.RegisterSharderKeep()
					if err != nil {
						Logger.Error("register_sharder_keep_worker", zap.Error(err))
					} else {
						if txn == nil || sc.ConfirmTransaction(txn) {
							Logger.Info("register_sharder_keep_worker -- registered")
						} else {
							Logger.Debug("register_sharder_keep_worker -- failed to confirm transaction", zap.Any("txn", txn))
						}
					}

					if sleepOrDone(ctx, time.Second) {
						return
					}
				}
			}
		}
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

func (sc *Chain) MoveOldBlocksToCloud(ctx context.Context) {
	var iterInprogress = false
	var oldBlockRoundRange = viper.GetInt64("minio.old_block_round_range")
	var numWorkers = viper.GetInt("minio.num_workers")
	var roundToProcess = int64(0)
	ticker := time.NewTicker(time.Duration(viper.GetInt64("minio.worker_frequency")) * time.Second)
	for true {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !iterInprogress {
				iterInprogress = true

				// Get total rounds to process
				roundsToProcess := sc.CurrentRound - (oldBlockRoundRange + roundToProcess)
				fs := blockstore.GetStore()
				if roundsToProcess > 0 {
					//Create sized wait group to do concurrent uploads
					swg := sizedwaitgroup.New(numWorkers)
					for roundToProcess <= roundsToProcess {
						// Get block hash for the round to process
						hash, err := sc.GetBlockHash(ctx, roundToProcess)
						if err != nil {
							Logger.Error("Unable to get block hash from round number", zap.Any("round", roundToProcess))
							roundToProcess++
							continue
						}

						swg.Add()
						go func(hash string, round int64) {
							err = fs.UploadToCloud(hash, round)
							if err != nil {
								Logger.Error("Error in uploading to cloud", zap.Error(err))
							}
							swg.Done()
						}(hash, roundToProcess)
						roundToProcess++
					}
					swg.Wait()
				}
				iterInprogress = false
				Logger.Info("Moved old blocks to cloud successfully")
			}
		}
	}
}
