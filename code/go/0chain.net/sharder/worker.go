package sharder

import (
	"context"
	"sort"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/sharder/blockstore"

	"0chain.net/chaincore/httpclientutil"
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
	go sc.RegisterSharderKeepWorker(ctx)
	// Move old blocks to cloud
	if viper.GetBool("minio.enabled") {
		go sc.MinioWorker(ctx)
	}

	go sc.SharderHealthCheck(ctx)
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
	if err == nil && sc.isValidRound(r) == true {
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

// isPhaseContibute
func (sc *Chain) isPhaseContibute(ctx context.Context) (is bool) {

	if sc.IsActiveInChain() {
		var lfb = sc.GetLatestFinalizedBlock()
		if lfb == nil {
			Logger.Error("is_phase_contibute -- can't get lfb")
			return
		}

		var cstate = chain.CreateTxnMPT(lfb.ClientState)
		if cstate == nil {
			Logger.Error("is_phase_contibute -- can't get phase node",
				zap.String("error", "missing client state of LFB"))
			return
		}
		var seri, err = cstate.GetNodeValue(
			util.Path(encryption.Hash(minersc.PhaseKey)),
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
			zap.Int("phase", int(phaseNode.Phase)),
			zap.Bool("is_contribute", phaseNode.Phase == minersc.Contribute))
		return phaseNode.Phase == minersc.Contribute
	}

	// not active in chain, use REST API call

	var mbs = sc.GetLatestFinalizedMagicBlockFromSharder(ctx)
	if len(mbs) == 0 {
		Logger.Error("is_phase_contibute -- no LFMB from sharders")
		return false
	}
	if len(mbs) > 1 {
		sort.Slice(mbs, func(i, j int) bool {
			return mbs[i].StartingRound > mbs[j].StartingRound
		})
	}

	var (
		magicBlock = mbs[0]                        // the latest one
		sharders   = magicBlock.Sharders.N2NURLs() // sharders
		pn         = new(minersc.PhaseNode)        //
		err        error                           //
	)
	err = httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, "/getPhase", nil,
		sharders, pn, 1)
	if err != nil {
		Logger.Error("is_phase_contibute -- requesting phase from sharders"+
			" using REST API call", zap.Error(err))
		return false
	}

	return pn.Phase == minersc.Contribute
}

func (sc *Chain) RegisterSharderKeepWorker(ctx context.Context) {

	if !config.DevConfiguration.ViewChange {
		return // don't send sharder_keep if view_change is false
	}

	var (
		timerCheck = time.NewTicker(5 * time.Second)
		doneq      = ctx.Done()
	)
	defer timerCheck.Stop()

	for {
		select {
		case <-doneq:
			return
		case <-timerCheck.C:

			if !sc.IsActiveInChain() &&
				!sc.IsRegisteredSharderKeep() &&
				sc.isPhaseContibute(ctx) {

				txn, err := sc.RegisterSharderKeep()
				if err != nil {
					Logger.Error("register_sharder_keep_worker", zap.Error(err))
				} else {
					if txn == nil || sc.ConfirmTransaction(txn) {
						Logger.Info("register_sharder_keep_worker -- " +
							"registered")
					} else {
						Logger.Debug("register_sharder_keep_worker -- failed "+
							"to confirm transaction", zap.Any("txn", txn))
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

func (sc *Chain) MinioWorker(ctx context.Context) {
	var iterInprogress = false
	var oldBlockRoundRange = viper.GetInt64("minio.old_block_round_range")
	var numWorkers = viper.GetInt("minio.num_workers")
	var roundProcessed = int64(0)
	ticker := time.NewTicker(time.Duration(viper.GetInt64("minio.worker_frequency")) * time.Second)
	for true {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !iterInprogress {
				iterInprogress = true
				roundDifference := sc.CurrentRound - roundProcessed
				if roundDifference > oldBlockRoundRange {
					roundsTillProcess := roundProcessed + (roundDifference - oldBlockRoundRange)
					swg := sizedwaitgroup.New(numWorkers)
					for roundProcessed <= roundsTillProcess {
						swg.Add()
						roundProcessed++
						go sc.moveBlockToCloud(ctx, roundProcessed, &swg)
					}
					swg.Wait()
				}
				iterInprogress = false
				Logger.Info("Moved old blocks to cloud successfully")
			}
		}
	}
}

func (sc *Chain) moveBlockToCloud(ctx context.Context, round int64, swg *sizedwaitgroup.SizedWaitGroup) {
	hash, err := sc.GetBlockHash(ctx, round)
	if err != nil {
		Logger.Error("Unable to get block hash from round number", zap.Any("round", round))
		swg.Done()
		return
	}

	fs := blockstore.GetStore()
	err = fs.UploadToCloud(hash, round)
	if err != nil {
		if fs.CloudObjectExists(hash) {
			Logger.Error("Error in uploading to cloud, The data is already present on cloud", zap.Error(err), zap.Any("round", round))
		} else {
			Logger.Error("Error in uploading to cloud, The data is also missing from cloud", zap.Error(err), zap.Any("round", round))
		}
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
