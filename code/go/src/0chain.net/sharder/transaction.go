package sharder

import (
	"context"
	"time"

	"0chain.net/block"
	"go.uber.org/zap"

	"0chain.net/datastore"
	"0chain.net/ememorystore"
	. "0chain.net/logging"
	"0chain.net/persistencestore"
	"0chain.net/transaction"
)

/*GetTransactionSummary - given a transaction hash, get the transaction summary */
func GetTransactionSummary(ctx context.Context, hash string) (*transaction.TransactionSummary, error) {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	txnSummary := txnSummaryEntityMetadata.Instance().(*transaction.TransactionSummary)
	err := txnSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(hash), txnSummary)
	if err != nil {
		return nil, err
	}
	return txnSummary, nil
}

/*GetTransactionConfirmation - given a transaction return the confirmation of it's presence in the block chain */
func GetTransactionConfirmation(ctx context.Context, hash string) (*transaction.Confirmation, error) {
	var ts *transaction.TransactionSummary
	t, err := GetSharderChain().BlockTxnCache.Get(hash)
	if err != nil {
		ts, err = GetTransactionSummary(ctx, hash)
		if err != nil {
			return nil, err
		}
	} else {
		ts = t.(*transaction.TransactionSummary)
	}
	confirmation := datastore.GetEntityMetadata("txn_confirmation").Instance().(*transaction.Confirmation)
	confirmation.Hash = hash
	confirmation.BlockHash = ts.BlockHash

	var b *block.Block
	bc, err := GetSharderChain().BlockCache.Get(ts.BlockHash)
	if err != nil {
		bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
		bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
		defer ememorystore.Close(bctx)
		bs, err := GetBlockSummary(bctx, ts.BlockHash)
		if err != nil {
			return nil, err
		}
		confirmation.Round = bs.Round
		confirmation.RoundRandomSeed = bs.RoundRandomSeed
		confirmation.CreationDate = bs.CreationDate
		b, err = GetSharderChain().GetBlockBySummary(ctx, bs)
		if err != nil {
			return confirmation, nil
		}
	} else {
		b = bc.(*block.Block)
		confirmation.Round = b.Round
		confirmation.RoundRandomSeed = b.RoundRandomSeed
		confirmation.CreationDate = b.CreationDate
	}
	txn := b.GetTransaction(hash)
	confirmation.Transaction = txn
	mt := b.GetMerkleTree()
	confirmation.MerkleTreeRoot = mt.GetRoot()
	confirmation.MerkleTreePath = mt.GetPath(confirmation)
	rmt := b.GetReceiptsMerkleTree()
	confirmation.ReceiptMerkleTreeRoot = rmt.GetRoot()
	confirmation.ReceiptMerkleTreePath = rmt.GetPath(transaction.NewTransactionReceipt(txn))
	return confirmation, nil
}

/*StoreTransactions - persists given list of transactions*/
func (sc *Chain) StoreTransactions(ctx context.Context, b *block.Block) error {
	var sTxns = make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		txnSummary := txn.GetSummary()
		txnSummary.BlockHash = b.Hash
		sTxns[idx] = txnSummary
	}
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryMetadata)
	defer persistencestore.Close(tctx)
	for numTrials := 1; numTrials <= 10; numTrials++ {
		err := txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, sTxns)
		if err != nil {
			Logger.Error("save transactions error", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
			if err.Error() == "gocql: no host available in the pool" {
				// long gc pauses can result in this error and so waiting longer to retry
				time.Sleep(100 * time.Millisecond)
			} else {
				time.Sleep(10 * time.Millisecond)
			}
		} else {
			Logger.Info("transactions saved successfully", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Int("block_size", len(b.Txns)))
			break
		}
	}
	return nil
}
