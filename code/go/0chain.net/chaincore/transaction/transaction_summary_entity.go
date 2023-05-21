package transaction

import (
	"context"
	"fmt"
	"path/filepath"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

/*TransactionSummary - the summary of the transaction */
type TransactionSummary struct {
	datastore.HashIDField // Keyspaced transaction hash - used as key
	Round int64 `json:"round"`
}

const TransactionKeyspace = "TRANSACTION"

var transactionSummaryEntityMetadata *datastore.EntityMetadataImpl

//TransactionSummaryProvider - factory method
func TransactionSummaryProvider() datastore.Entity {
	t := &TransactionSummary{}
	return t
}

//GetEntityMetadata - implement interface
func (t *TransactionSummary) GetEntityMetadata() datastore.EntityMetadata {
	return transactionSummaryEntityMetadata
}

// SetTransactionKey - set the entity hash to the keyspaced hash of the transaction hash
func BuildSummaryTransactionKey(hash string) datastore.Key {
	return datastore.ToKey(
		fmt.Sprintf(
			"%s:%s",
			TransactionKeyspace,
			hash,
		),
	)
}

//GetKey - implement interface
func (t *TransactionSummary) GetKey() datastore.Key {
	return datastore.ToKey(t.Hash)
}

//SetKey - implement interface
func (t *TransactionSummary) SetKey(key datastore.Key) {
	t.Hash = datastore.ToString(key)
}

/*Read - store read */
func (t *TransactionSummary) Read(ctx context.Context, key datastore.Key) error {
	return t.GetEntityMetadata().GetStore().Read(ctx, key, t)
}

/*GetScore - score for write*/
func (t *TransactionSummary) GetScore() (int64, error) {
	return t.Round, nil
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

// SetupRoundSummaryDB - setup the round summary db
// SummaryDB has 2 keyspaces, TRANSACTION and ROUND specified by the prefix of the key (Hash):
// 1. Maps transaction hash to round number
// 2. Maps round number (hashed) to number of transactions in that round
func SetupTxnSummaryDB(workdir string) {
	datadir := filepath.Join(workdir, "data/rocksdb/txnsummary")

	db, err := ememorystore.CreateDB(datadir)
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("txnsummarydb", db)
}