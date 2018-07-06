package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/ememorystore"
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

/*GetBlockSummary - given a block hash, get the block summary */
func GetBlockSummary(ctx context.Context, hash string) (*block.BlockSummary, error) {
	blockSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	blockSummary := blockSummaryEntityMetadata.Instance().(*block.BlockSummary)
	err := blockSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(hash), blockSummary)
	if err != nil {
		return nil, err
	}
	return blockSummary, nil
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
