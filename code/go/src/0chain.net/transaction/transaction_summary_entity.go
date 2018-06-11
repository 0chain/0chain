package transaction

import (
	"0chain.net/common"
	"0chain.net/datastore"
)

/*TransactionSummary - the summary of the transaction */
type TransactionSummary struct {
	datastore.VersionField
	datastore.CreationDateField
	datastore.NOIDField
	Hash       string        `json:"hash"`
	Block      string        `json:"block_hash"`
	ClientID   datastore.Key `json:"client_id"`
	ToClientID datastore.Key `json:"to_client_id"`
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

func SetupTxnSummaryEntity(store datastore.Store) {
	transactionSummaryEntityMetadata = datastore.MetadataProvider()
	transactionSummaryEntityMetadata.Name = "txn_summary"
	transactionSummaryEntityMetadata.MemoryDB = "txn_summarydb"
	transactionSummaryEntityMetadata.Provider = TransactionSummaryProvider
	transactionSummaryEntityMetadata.Store = store
	datastore.RegisterEntityMetadata("txn_summary", transactionSummaryEntityMetadata)
}
