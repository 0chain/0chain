package transaction

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"go.uber.org/zap"
)

//SetupWorkers - setup workers */
func SetupWorkers(ctx context.Context) {
	go CleanupWorker(ctx)
}

/*CleanupWorker - a worker to delete transactiosn that are no longer valid */
func CleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
	mstore, ok := transactionEntityMetadata.GetStore().(*memorystore.Store)
	if !ok {
		return
	}
	var invalidTxns = make([]datastore.Entity, 0, 1024)
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*Transaction)
	collectionName := txn.GetCollectionName()

	var handler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		txn, ok := qe.(*Transaction)
		if !ok {
			qe.Delete(ctx)
			return true
		}
		if !common.Within(int64(txn.CreationDate), GetTxnTimeout()-1) {
			invalidTxns = append(invalidTxns, txn)
			return true
		}
		return false
	}

	for true {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mstore.IterateCollectionAsc(cctx, transactionEntityMetadata, collectionName, handler)
			if len(invalidTxns) > 0 {
				Logger.Info("transactions cleanup", zap.String("collection", collectionName), zap.Int("invalid_count", len(invalidTxns)))
				transactionEntityMetadata.GetStore().MultiDelete(cctx, transactionEntityMetadata, invalidTxns)
				invalidTxns = invalidTxns[:0]
			}
		}
	}
}
