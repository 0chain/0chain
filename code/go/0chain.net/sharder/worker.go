package sharder

import (
	"context"
	"sync"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"
	"0chain.net/core/viper"
	"0chain.net/sharder/blockstore"
	"0chain.net/smartcontract/minersc"

	"0chain.net/core/logging"
)

const minerScSharderHealthCheck = "sharder_health_check"

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go sc.FinalizeRoundWorker(ctx)      // 2) sequentially finalize the rounds
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
	const stuckDuration = 3 * time.Second
	var (
		endRound int64
		syncing  bool

		syncBlocksTimer  = time.NewTimer(7 * time.Second)
		aheadN           = int64(config.GetLFBTicketAhead())
		maxRequestBlocks = aheadN

		// triggered after sync process is started
		stuckCheckTimer = time.NewTimer(10 * time.Second)
	)

	for {
		select {
		case <-ctx.Done():
			logging.Logger.Error("BlockWorker exit", zap.Error(ctx.Err()))
			return
		case <-stuckCheckTimer.C:
			logging.Logger.Debug("process block, detected stuck, trigger sync",
				zap.Int64("round", sc.GetCurrentRound()),
				zap.Int64("lfb", sc.GetLatestFinalizedBlock().Round))
			stuckCheckTimer.Reset(stuckDuration)
			// trigger sync
			syncBlocksTimer.Reset(0)
		case <-syncBlocksTimer.C:
			// reset sync timer to 1 minute
			syncBlocksTimer.Reset(time.Minute)

			var (
				lfbTk = sc.GetLatestLFBTicket(ctx)
				lfb   = sc.GetLatestFinalizedBlock()
			)

			cr := sc.GetCurrentRound()
			if cr < lfb.Round {
				sc.SetCurrentRound(lfb.Round)
				cr = lfb.Round
			}

			if lfb.Round+aheadN <= cr {
				logging.Logger.Debug("process block, synced to lfb+ahead, start to force finalize rounds",
					zap.Int64("current round", cr),
					zap.Int64("lfb", lfb.Round),
					zap.Int64("lfb+ahead", lfb.Round+aheadN))

				for rn := lfb.Round + 1; rn <= cr; rn++ {
					if r := sc.GetRound(rn); r != nil {
						sc.FinalizeRound(sc.GetRound(rn))
					}
				}
				continue
			}

			endRound = lfbTk.Round + aheadN

			if endRound <= cr {
				continue
			}

			logging.Logger.Debug("process block, sync triggered",
				zap.Int64("lfb", lfb.Round),
				zap.Int64("lfb ticket", lfbTk.Round),
				zap.Int64("current round", cr),
				zap.Int64("end round", endRound))

			// trunc to send maxRequestBlocks each time
			reqNum := endRound - cr
			if reqNum > maxRequestBlocks {
				reqNum = maxRequestBlocks
			}

			endRound = cr + reqNum
			syncing = true

			logging.Logger.Debug("process block, sync blocks",
				zap.Int64("start round", cr+1),
				zap.Int64("end round", cr+reqNum+1))
			go sc.requestBlocks(ctx, cr, reqNum)
		case b := <-sc.blockChannel:
			stuckCheckTimer.Reset(stuckDuration)
			cr := sc.GetCurrentRound()
			lfb := sc.GetLatestFinalizedBlock()
			if b.Round > lfb.Round+aheadN {

				// trigger sync process to pull the latest blocks when
				// current round is > lfb.Round + aheadN to break the stuck if any.
				if !syncing {
					syncBlocksTimer.Reset(0)
				}

				// avoid the skipping logs when syncing blocks
				if b.Round <= lfb.Round+2*aheadN {
					logging.Logger.Debug("process block skip",
						zap.Int64("block round", b.Round),
						zap.Int64("current round", cr),
						zap.Int64("lfb", lfb.Round),
						zap.Bool("syncing", syncing))
				}

				continue
			}

			if err := sc.processBlock(ctx, b); err != nil {
				logging.Logger.Error("process block failed",
					zap.Error(err),
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.String("prev block", b.PrevHash))
				if err != ErrNoPreviousBlock {
					continue
				}

				// fetch the previous block
				pb, _ := sc.GetNotarizedBlock(ctx, b.PrevHash, b.Round-1)
				if pb != nil {
					if err := sc.processBlock(ctx, pb); err != nil {
						logging.Logger.Error("process block, handle previous block failed",
							zap.Int64("round", pb.Round),
							zap.String("block", pb.Hash),
							zap.Error(err))
						continue
					}

					// process this block again
					if err := sc.processBlock(ctx, b); err != nil {
						logging.Logger.Error("process block, failed after getting previous block",
							zap.Int64("round", b.Round),
							zap.String("block", b.Hash),
							zap.Error(err))
						continue
					}
				}
			}

			lfbTk := sc.GetLatestLFBTicket(ctx)
			lfb = sc.GetLatestFinalizedBlock()
			logging.Logger.Debug("process block successfully",
				zap.Int64("round", b.Round),
				zap.Int64("lfb round", lfb.Round),
				zap.Int64("lfb ticket round", lfbTk.Round))

			if b.Round >= lfb.Round+aheadN || b.Round >= endRound {
				syncing = false
				if b.Round < lfbTk.Round {
					logging.Logger.Debug("process block, hit end, trigger sync",
						zap.Int64("round", b.Round),
						zap.Int64("end round", endRound),
						zap.Int64("current round", cr))
					syncBlocksTimer.Reset(0)
				}
			}
		}
	}
}

func (sc *Chain) requestBlocks(ctx context.Context, startRound, reqNum int64) {
	blocks := make([]*block.Block, reqNum)
	wg := sync.WaitGroup{}
	for i := int64(0); i < reqNum; i++ {
		wg.Add(1)
		go func(idx int64) {
			defer wg.Done()
			r := startRound + idx + 1
			var cancel func()
			cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			b, err := sc.GetNotarizedBlockFromSharders(cctx, "", r)
			if err != nil {
				// fetch from miners
				b, err = sc.GetNotarizedBlock(cctx, "", r)
				if err != nil {
					logging.Logger.Error("request block failed",
						zap.Int64("round", r),
						zap.Error(err))
					return
				}
			}

			blocks[idx] = b
		}(i)
	}
	wg.Wait()

	for _, b := range blocks {
		if b == nil {
			// return if block is not acquired, break here as we will redo the sync process from the missed one later
			return
		}

		if err := sc.PushToBlockProcessor(b); err != nil {
			logging.Logger.Debug("requested block, but failed to pushed to process channel",
				zap.Int64("round", b.Round), zap.Error(err))
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

func (sc *Chain) hasBlockTransactions(ctx context.Context, b *block.Block) bool { //nolint
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
			if tp.Sub(latest) < repeat || len(phaseq) > 0 {
				continue // already have a fresh phase
			}
			go sc.GetPhaseFromSharders(ctx)
			continue
		case pe = <-phaseq:
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

		if !sc.ConfirmTransaction(ctx, txn) {
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

func (sc *Chain) MinioWorker(ctx context.Context) {
	if !viper.GetBool("minio.enabled") {
		return
	}
	var oldBlockRoundRange = viper.GetInt64("minio.old_block_round_range")
	var numWorkers = viper.GetInt("minio.num_workers")
	ticker := time.NewTicker(time.Duration(viper.GetInt64("minio.worker_frequency")) * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			roundToProcess := sc.GetCurrentRound() - oldBlockRoundRange
			fs := blockstore.GetStore()
			swg := sizedwaitgroup.New(numWorkers)
			for roundToProcess > 0 {
				hash, err := sc.GetBlockHash(ctx, roundToProcess)
				if err != nil {
					logging.Logger.Error("Unable to get block hash from round number", zap.Any("round", roundToProcess))
					roundToProcess--
					continue
				}
				if fs.CloudObjectExists(hash) {
					logging.Logger.Info("The data is already present on cloud, Terminating the worker...", zap.Any("round", roundToProcess))
					break
				} else {
					swg.Add()
					go sc.moveBlockToCloud(ctx, roundToProcess, hash, fs, &swg)
					roundToProcess--
				}
			}
			swg.Wait()
			logging.Logger.Info("Moved old blocks to cloud successfully")
		}
	}
}

func (sc *Chain) moveBlockToCloud(ctx context.Context, round int64, hash string, fs blockstore.BlockStore, swg *sizedwaitgroup.SizedWaitGroup) {
	err := fs.UploadToCloud(hash, round)
	if err != nil {
		logging.Logger.Error("Error in uploading to cloud, The data is also missing from cloud", zap.Error(err), zap.Any("round", round))
	} else {
		logging.Logger.Info("Block successfully uploaded to cloud", zap.Any("round", round))
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
			if err := httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls); err != nil {
				logging.Logger.Warn("sharder health check failed, try again")
			}

		}
		time.Sleep(HEALTH_CHECK_TIMER * time.Second)
	}
}
