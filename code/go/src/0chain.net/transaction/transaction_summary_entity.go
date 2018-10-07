package transaction

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*TransactionSummary - the summary of the transaction */
type TransactionSummary struct {
	datastore.VersionField
	datastore.CreationDateField
	datastore.HashIDField
	BlockHash string `json:"block_hash"`
}

var transactionSummaryEntityMetadata *datastore.EntityMetadataImpl

func TransactionSummaryProvider() datastore.Entity {
	t := &TransactionSummary{}
	t.Version = "1.0"
	t.CreationDate = common.Now()
	return t
}

func (t *TransactionSummary) GetEntityMetadata() datastore.EntityMetadata {
	return transactionSummaryEntityMetadata
}

func (t *TransactionSummary) GetKey() datastore.Key {
	return datastore.ToKey(t.Hash)
}

func (t *TransactionSummary) SetKey(key datastore.Key) {
	t.Hash = datastore.ToString(key)
}

/*Read - store read */
func (t *TransactionSummary) Read(ctx context.Context, key datastore.Key) error {
	return t.GetEntityMetadata().GetStore().Read(ctx, key, t)
}

/*Write - store read */
func (t *TransactionSummary) Write(ctx context.Context) error {
	return t.GetEntityMetadata().GetStore().Write(ctx, t)
}

/*Delete - store read */
func (t *TransactionSummary) Delete(ctx context.Context) error {
	return t.GetEntityMetadata().GetStore().Delete(ctx, t)
}

/*SetupTxnSummaryEntity - setup the txn summary entity */
func SetupTxnSummaryEntity(store datastore.Store) {
	transactionSummaryEntityMetadata = datastore.MetadataProvider()
	transactionSummaryEntityMetadata.Name = "txn_summary"
	transactionSummaryEntityMetadata.Provider = TransactionSummaryProvider
	transactionSummaryEntityMetadata.Store = store
	transactionSummaryEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("txn_summary", transactionSummaryEntityMetadata)
}
