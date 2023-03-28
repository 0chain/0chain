package transaction

import (
	"context"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// SetupWorkers - setup workers */
func SetupWorkers(ctx context.Context) {
	go CleanupWorker(ctx)
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
				logging.Logger.Info("transactions cleanup", zap.String("collection", collectionName), zap.Int("invalid_count", len(invalidTxns)), zap.Int64("collection_size", mstore.GetCollectionSize(cctx, transactionEntityMetadata, collectionName)))
				err = transactionEntityMetadata.GetStore().MultiDelete(cctx, transactionEntityMetadata, invalidTxns)
				if err != nil {
					logging.Logger.Error("Error in MultiDelete", zap.Error(err))
				} else {
					invalidTxns = invalidTxns[:0]
				}
			}
			if len(invalidHashes) > 0 {
				logging.Logger.Info("missing transactions cleanup", zap.String("collection", collectionName), zap.Int("missing_count", len(invalidHashes)))
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
			}
			return true, nil
		})
	if err != nil {
		logging.Logger.Error("error finding past transactions", zap.Error(err))
		//try to delete what we can, so no return here
	}
	txns = append(past, txns...)
	logging.Logger.Info("cleaning transactions", zap.String("collection", collectionName), zap.Int("missing_count", len(txns)))
	err = transactionEntityMetadata.GetStore().MultiDeleteFromCollection(cctx, transactionEntityMetadata, txns)
	if err != nil {
		logging.Logger.Error("Error in MultiDeleteFromCollection", zap.Error(err))
	}
}
