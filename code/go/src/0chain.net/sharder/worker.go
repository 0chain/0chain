package sharder

import (
	"context"
	"time"

	"0chain.net/chain"
	"0chain.net/state"
	"0chain.net/util"

	"0chain.net/block"

	"0chain.net/blockstore"

	"0chain.net/node"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers() {
	ClearWorkerState()
	ctx := common.GetRootContext()
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)                 // 1) receives incoming blocks from the network
	go sc.BlockFinalizationWorker(ctx, sc) // 2) sequentially runs finalization logic
	go sc.BlockStorageWorker(ctx)          // 3) persists the blocks
}

/*ClearWorkerState - clears the worker state */
func ClearWorkerState() {
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

func (sc *Chain) processBlock(ctx context.Context, b *block.Block) {
	eb, err := sc.GetBlock(ctx, b.Hash)
	if eb != nil {
		if err == nil {
			Logger.Debug("block already received", zap.Any("round", b.Round), zap.Any("block", b.Hash))
			return
		} else {
			Logger.Error("get block", zap.Any("block", b.Hash), zap.Error(err))
		}
	}
	if err := sc.VerifyNotarization(ctx, b, b.VerificationTickets); err != nil {
		Logger.Error("notarization verification failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	if err := b.Validate(ctx); err != nil {
		Logger.Error("block validation", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return
	}
	sc.AddBlock(b)
	er := sc.GetRound(b.Round)
	if er != nil {
		if sc.BlocksToSharder == chain.FINALIZED {
			nb := er.GetNotarizedBlocks()
			if len(nb) > 0 {
				Logger.Error("*** different blocks for the same round ***", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("existing_block", nb[0].Hash))
			}
		}
	} else {
		er = datastore.GetEntityMetadata("round").Instance().(*round.Round)
		er.Number = b.Round
		er.RandomSeed = b.RoundRandomSeed
		sc.AddRound(er)
	}
	Logger.Info("received block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.String("prev_state", util.ToHex(b.ClientState.GetRoot())))
	err = sc.ComputeState(ctx, b)
	if err != nil {
		if config.DevConfiguration.State {
			Logger.Error("error computing the state (TODO sync state)", zap.Error(err))
		}
	}
	if b.Round == 1 {
		val, err := b.ClientState.GetNodeValue(util.Path(sc.OwnerID))
		if err != nil {
			panic(err)
		} else {
			state := sc.ClientStateDeserializer.Deserialize(val).(*state.State)
			Logger.Info("initial tokens", zap.Any("state", state))
		}
	}
	er.AddNotarizedBlock(b)
	pr := sc.GetRound(er.Number - 1)
	if pr != nil {
		go sc.FinalizeRound(ctx, pr, sc)
	}
}

/*BlockStorageWorker - a background worker that processes a block to store it in suitable formats */
func (sc *Chain) BlockStorageWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case r := <-sc.GetRoundChannel():
			b, err := sc.GetBlockFromHash(ctx, r.BlockHash, r.Number)
			if err != nil {
				Logger.Error("failed to get block", zap.String("blockhash", r.BlockHash), zap.Error(err))
			} else {
				self := node.GetSelfNode(ctx)
				if sc.CanStoreBlock(r, self.Node) {
					var sTxns = make([]datastore.Entity, len(b.Txns))
					for idx, txn := range b.Txns {
						txnSummary := txn.GetSummary()
						txnSummary.BlockHash = b.Hash
						sTxns[idx] = txnSummary
					}
					err = sc.StoreTransactions(ctx, sTxns)
					if err != nil {
						Logger.Error("save transactions error", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
						err = sc.retryPersistingTransactions(ctx, sTxns, b)
						if err != nil {
							Logger.Error("save transactions error", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
							continue
						} else {
							Logger.Info("transactions saved successfully", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Int("block_size", len(b.Txns)))
						}
					} else {
						Logger.Info("transactions saved successfully", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Int("block_size", len(b.Txns)))
					}
					err = sc.StoreBlock(ctx, b)
					if err != nil {
						Logger.Error("db error (save block)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
					}
				} else {
					err = blockstore.GetStore().DeleteBlock(b)
					if err != nil {
						Logger.Error("failed to delete block from file system", zap.String("blockhash", b.Hash), zap.Error(err))
					}
				}
			}
		}
	}
}

func (sc *Chain) retryPersistingTransactions(ctx context.Context, sTxns []datastore.Entity, b *block.Block) error {
	var err error
	for numTrials := 1; numTrials <= 10; numTrials++ {
		Logger.Info("retrying to save transactions to db", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("trail", numTrials), zap.Error(err))
		err = sc.StoreTransactions(ctx, sTxns)
		if err == nil {
			return nil
		}
		if err.Error() == "gocql: no host available in the pool" {
			// long gc pauses can result in this error and so waiting longer to retry
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return err
}
