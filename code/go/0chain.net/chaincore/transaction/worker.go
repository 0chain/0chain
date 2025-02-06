package transaction

import (
	"context"
	"time"

	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

var invalidFutureTxnsCache = cache.NewLRUCache[string, struct{}](100)

// SetupWorkers - setup workers */
func SetupWorkers(ctx context.Context) {
	go CleanupWorker(ctx)
}

func IsInvalidFutureTxn(hash string) bool {
	_, err := invalidFutureTxnsCache.Get(hash)
	return err == nil // see txn in the cache, means it's invalid
}

func RemoveInvalidTxnsFromCache(hashes []string) {
	for _, hash := range hashes {
		invalidFutureTxnsCache.Remove(hash)
	}
}

/*CleanupWorker - a worker to delete transactiosn that are no longer valid */
func CleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
	defer memorystore.Close(cctx)
	mstore, ok := transactionEntityMetadata.GetStore().(*memorystore.Store)
	if !ok {
		return
	}
	var (
		invalidHashes = make([]datastore.Entity, 0, 1024)
		invalidTxns   = make([]datastore.Entity, 0, 1024)
	)
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*Transaction)
	collectionName := txn.GetCollectionName()

	var handler = func(ctx context.Context, qe datastore.CollectionEntity) (bool, error) {
		txn, ok := qe.(*Transaction)
		if !ok {
			err := qe.Delete(ctx)
			if err != nil {
				logging.Logger.Error("Error in deleting txn in redis", zap.Error(err))
			}
		}
		if !common.Within(int64(txn.CreationDate), TXN_TIME_TOLERANCE-1) {
			invalidTxns = append(invalidTxns, txn)
		}
		err := transactionEntityMetadata.GetStore().Read(ctx, txn.Hash, txn)
		cerr, ok := err.(*common.Error)
		if ok && cerr.Code == datastore.EntityNotFound {
			invalidHashes = append(invalidHashes, txn)
		}
		return true, nil
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := mstore.IterateCollectionAsc(cctx, transactionEntityMetadata, collectionName, handler)
			if err != nil {
				logging.Logger.Error("Error in IterateCollectionAsc", zap.Error(err))
			}
			if len(invalidTxns) > 0 {
				invalidTxnHashes := make([]string, len(invalidTxns))
				for i, t := range invalidTxns {
					invalidTxnHashes[i] = t.(*Transaction).Hash
				}

				logging.Logger.Info("transactions cleanup",
					zap.String("collection", collectionName),
					zap.Int("invalid_count", len(invalidTxns)),
					zap.Strings("txns", invalidTxnHashes),
					zap.Int64("collection_size", mstore.GetCollectionSize(cctx, transactionEntityMetadata, collectionName)))
				err = transactionEntityMetadata.GetStore().MultiDelete(cctx, transactionEntityMetadata, invalidTxns)
				if err != nil {
					logging.Logger.Error("Error in MultiDelete", zap.Error(err))
				} else {
					invalidTxns = invalidTxns[:0]
				}
			}
			if len(invalidHashes) > 0 {
				txnHashes := make([]string, len(invalidHashes))
				for i, t := range invalidHashes {
					txnHashes[i] = t.(*Transaction).Hash
				}
				logging.Logger.Info("missing transactions cleanup",
					zap.String("collection", collectionName),
					zap.Int("missing_count", len(invalidHashes)),
					zap.Strings("txns", txnHashes))
				err = transactionEntityMetadata.GetStore().MultiDeleteFromCollection(cctx, transactionEntityMetadata, invalidHashes)
				if err != nil {
					logging.Logger.Error("Error in MultiDeleteFromCollection", zap.Error(err))
				} else {
					invalidHashes = invalidHashes[:0]
				}
			}
		}
	}
}

func RemoveFromPool(ctx context.Context, txns []datastore.Entity) {
	cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
	defer memorystore.Close(cctx)

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*Transaction)
	collectionName := txn.GetCollectionName()

	logging.Logger.Debug("cleaning past transactions")
	clientMaxNonce := make(map[string]int64)
	for _, e := range txns {
		blockTx, ok := e.(*Transaction)
		if !ok {
			logging.Logger.Error("generate block (invalid entity)", zap.Any("entity", e))
			continue
		}
		nonce := clientMaxNonce[blockTx.ClientID]
		if blockTx.Nonce > nonce {
			clientMaxNonce[blockTx.ClientID] = blockTx.Nonce
		}
	}

	var past []datastore.Entity
	var pastTxnHash []string
	err := transactionEntityMetadata.GetStore().IterateCollection(cctx, transactionEntityMetadata, collectionName,
		func(ctx context.Context, qe datastore.CollectionEntity) (bool, error) {
			current, ok := qe.(*Transaction)
			if !ok {
				logging.Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
				return true, nil
			}

			maxNonce := clientMaxNonce[current.ClientID]
			if current.Nonce <= maxNonce {
				past = append(past, current)
				pastTxnHash = append(pastTxnHash, current.Hash)
			}
			return true, nil
		})
	if err != nil {
		logging.Logger.Error("error finding past transactions", zap.Error(err))
		//try to delete what we can, so no return here
	}
	txns = append(past, txns...)
	txnHashes := make([]string, len(txns))
	for i, t := range txns {
		txnHashes[i] = t.(*Transaction).Hash
	}
	logging.Logger.Info("cleaning transactions",
		zap.String("collection", collectionName),
		zap.Int("missing_count", len(txns)),
		zap.Any("txns", txnHashes),
		zap.Any("past txns", pastTxnHash))
	err = transactionEntityMetadata.GetStore().MultiDeleteFromCollection(cctx, transactionEntityMetadata, txns)
	if err != nil {
		logging.Logger.Error("Error in MultiDeleteFromCollection", zap.Error(err))
	} else {
		RemoveInvalidTxnsFromCache(txnHashes)
	}
}

func CollectInvalidFutureTxns(ctx context.Context, creationDate common.Timestamp, nonce int64, clientID string) ([]datastore.Entity, error) {
	cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
	defer memorystore.Close(cctx)

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*Transaction)
	collectionName := txn.GetCollectionName()

	var (
		futureTxns []datastore.Entity
		txnHashes  []string
	)

	err := transactionEntityMetadata.GetStore().IterateCollection(cctx, transactionEntityMetadata,
		collectionName, func(ctx context.Context, qe datastore.CollectionEntity) (bool, error) {
			txn, ok := qe.(*Transaction)
			if !ok {
				logging.Logger.Error("remove future txns (invalid entity)", zap.Any("entity", qe))
				return true, nil
			}

			if (txn.CreationDate >= creationDate || (nonce > 0 && txn.Nonce >= nonce)) && txn.ClientID == clientID {
				futureTxns = append(futureTxns, txn)
				txnHashes = append(txnHashes, txn.Hash)
			}
			return true, nil
		})
	if err != nil {
		return nil, err
	}

	if len(futureTxns) == 0 {
		return nil, nil
	}

	for _, hash := range txnHashes {
		invalidFutureTxnsCache.Add(hash, struct{}{})
	}

	logging.Logger.Info("[mvc] find invalid future transactions", zap.Any("txns", txnHashes))
	return futureTxns, nil
}

func GetOldNonceTxns(ctx context.Context, clientID string, nonce int64) ([]datastore.Entity, error) {
	logging.Logger.Debug("[mvc] remove old nonce txns", zap.String("clientID", clientID), zap.Int64("nonce", nonce))
	cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
	defer memorystore.Close(cctx)

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*Transaction)
	collectionName := txn.GetCollectionName()

	var (
		oldTxns   []datastore.Entity
		txnHashes []string
	)

	err := transactionEntityMetadata.GetStore().IterateCollection(cctx, transactionEntityMetadata,
		collectionName, func(ctx context.Context, qe datastore.CollectionEntity) (bool, error) {
			txn, ok := qe.(*Transaction)
			if !ok {
				logging.Logger.Error("remove future txns (invalid entity)", zap.Any("entity", qe))
				return true, nil
			}

			if txn.Nonce <= nonce && txn.ClientID == clientID {
				oldTxns = append(oldTxns, txn)
				txnHashes = append(txnHashes, txn.Hash)
			}
			return true, nil
		})
	if err != nil {
		return nil, err
	}

	if len(oldTxns) == 0 {
		logging.Logger.Debug("[mvc] see no old txns", zap.String("clientID", clientID), zap.Int64("nonce", nonce))
		return nil, nil
	}

	logging.Logger.Info("[mvc] clean txns, old transactions", zap.Any("txns", txnHashes))
	return oldTxns, nil
	// return transactionEntityMetadata.GetStore().MultiDeleteFromCollection(cctx, transactionEntityMetadata, oldTxns)
}
