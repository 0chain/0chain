package sharder

import (
	"context"

	"0chain.net/ememorystore"
	"0chain.net/persistencestore"
	"0chain.net/transaction"

	"0chain.net/datastore"
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
	ts, err := GetTransactionSummary(ctx, hash)
	if err != nil {
		return nil, err
	}
	bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	bs, err := GetBlockSummary(bctx, ts.BlockHash)
	if err != nil {
		return nil, err
	}
	confirmation := datastore.GetEntityMetadata("txn_confirmation").Instance().(*transaction.Confirmation)
	confirmation.Hash = hash
	confirmation.BlockHash = ts.BlockHash
	confirmation.Round = bs.Round
	confirmation.RoundRandomSeed = bs.RoundRandomSeed
	confirmation.CreationDate = bs.CreationDate
	b, err := GetSharderChain().GetBlockBySummary(ctx, bs)
	if err != nil {
		return nil, err
	}
	mt := b.GetMerkleTree()
	confirmation.MerkleTreeRoot = mt.GetRoot()
	confirmation.MerkleTreePath = mt.GetPath(confirmation)
	return confirmation, nil
}

/*StoreTransactions - persists given list of transactions*/
func (sc *Chain) StoreTransactions(ctx context.Context, txns []datastore.Entity) error {
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryMetadata)
	defer persistencestore.Close(tctx)
	err := txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, txns)
	if err != nil {
		return err
	}
	return nil
}
