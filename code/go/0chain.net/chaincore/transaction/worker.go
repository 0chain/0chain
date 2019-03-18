package transaction

import (
	"context"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"go.uber.org/zap"
)

//SetupWorkers - setup workers */
func SetupWorkers(ctx context.Context) {
	go CleanupWorker(ctx)
	go ConnectionCountWorker(ctx)
}

func ConnectionCountWorker(ctx context.Context) {
	for true {
		time.Sleep(time.Second * 3)
		openConnections, idleConnections := memorystore.GetConnectionCount(transactionEntityMetadata)
		Logger.Info("connection count for redis_txns", zap.Any("redis active connections", openConnections), zap.Any("redis idle connections", idleConnections))
	}
}

/*CleanupWorker - a worker to delete transactiosn that are no longer valid */
func CleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
	mstore, ok := transactionEntityMetadata.GetStore().(*memorystore.Store)
	if !ok {
		return
	}
	var invalidHashes = make([]datastore.Entity, 0, 1024)
	var invalidTxns = make([]datastore.Entity, 0, 1024)
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*Transaction)
	collectionName := txn.GetCollectionName()

	var handler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		txn, ok := qe.(*Transaction)
		if !ok {
			err := qe.Delete(ctx)
			if err != nil {
				Logger.Error("Error in deleting txn in redis", zap.Error(err))
			}
			return true
		}
		if !common.Within(int64(txn.CreationDate), TXN_TIME_TOLERANCE-1) {
			invalidTxns = append(invalidTxns, txn)
			return true
		}
		err := transactionEntityMetadata.GetStore().Read(ctx, txn.Hash, txn)
		cerr, ok := err.(*common.Error)
		if ok && cerr.Code == datastore.EntityNotFound {
			invalidHashes = append(invalidHashes, txn)
			return true
		}
		return false
	}

	for true {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := mstore.IterateCollectionAsc(cctx, transactionEntityMetadata, collectionName, handler)
			if err != nil {
				Logger.Error("Error in IterateCollectionAsc", zap.Error(err))
			}
			if len(invalidTxns) > 0 {
				Logger.Info("transactions cleanup", zap.String("collection", collectionName), zap.Int("invalid_count", len(invalidTxns)))
				err = transactionEntityMetadata.GetStore().MultiDelete(cctx, transactionEntityMetadata, invalidTxns)
				if err != nil {
					Logger.Error("Error in MultiDelete", zap.Error(err))
				} else {
					invalidTxns = invalidTxns[:0]
				}
			}
			if len(invalidHashes) > 0 {
				Logger.Info("missing transactions cleanup", zap.String("collection", collectionName), zap.Int("missing_count", len(invalidHashes)))
				err = transactionEntityMetadata.GetStore().MultiDeleteFromCollection(cctx, transactionEntityMetadata, invalidHashes)
				if err != nil {
					Logger.Error("Error in MultiDeleteFromCollection", zap.Error(err))
				} else {
					invalidHashes = invalidHashes[:0]
				}
			}
		}
	}
}
