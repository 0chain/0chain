package sharder

import (
	"io/ioutil"
	"testing"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"github.com/stretchr/testify/require"
)

func TestStoreTransactions(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "txnsummarydb")
	require.NoError(t, err)

	txnStore, err := ememorystore.CreateDBWithMergeOperator(tmpDir, ememorystore.NewCounterMergeOperator("hash", "txns_count"))
	require.NoError(t, err)
	ememorystore.AddPool("txnsummarydb", txnStore)

	ememoryStore := ememorystore.GetStorageProvider()
	transaction.SetupTxnSummaryEntity(ememoryStore)

	txns := []datastore.Entity{
		&transaction.TransactionSummary{
			HashIDField: datastore.HashIDField{
				Hash: "hash1",
			},
			Round: 100,
		},
		&transaction.TransactionSummary{
			HashIDField: datastore.HashIDField{
				Hash: "hash2",
			},
			Round: 100,
		},
		&transaction.TransactionSummary{
			HashIDField: datastore.HashIDField{
				Hash: "hash3",
			},
			Round: 100,
		},
	}

	chain := Chain{}

	err = chain.storeTransactions(txns, 100)
	require.NoError(t, err)

	transactionSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	ctx := ememorystore.WithEntityConnection(common.GetRootContext(), transactionSummaryMetadata)
	defer ememorystore.Close(ctx)


	// Read from rocksdb and make sure those transactions are saved
	for _, txn := range txns {
		txnSummary, ok := txn.(*transaction.TransactionSummary)
		require.True(t, ok)
		txnFromDB := transactionSummaryMetadata.Instance().(*transaction.TransactionSummary)
		err := txnSummary.GetEntityMetadata().GetStore().Read(ctx, txn.GetKey(), txnFromDB)
		require.NoError(t, err)
		require.Equal(t, txnSummary.Hash, txnFromDB.Hash)
		require.Equal(t, txnSummary.Round, txnFromDB.Round)
	}

	// Check round txn count is updated
	rtcKey := transaction.BuildSummaryRoundKey(100)
	var rtc transaction.RoundTxnsCount
	err = transactionSummaryMetadata.GetStore().Read(ctx, rtcKey, &rtc)
	require.NoError(t, err)
	require.Equal(t, int64(3), rtc.TxnsCount)

	// Add one more txn
	newTxns := []datastore.Entity{
		&transaction.TransactionSummary{
			HashIDField: datastore.HashIDField{
				Hash: "hash4",
			},
			Round: 100,
		},
	}

	err = chain.storeTransactions(newTxns, 100)
	require.NoError(t, err)

	// Read from rocksdb and make sure this transaction is saved
	txnSummary, ok := newTxns[0].(*transaction.TransactionSummary)
	require.True(t, ok)
	txnFromDB := transactionSummaryMetadata.Instance().(*transaction.TransactionSummary)
	err = txnSummary.GetEntityMetadata().GetStore().Read(ctx, txnSummary.GetKey(), txnFromDB)
	require.NoError(t, err)
	require.Equal(t, txnSummary.Hash, txnFromDB.Hash)
	require.Equal(t, txnSummary.Round, txnFromDB.Round)



	// Check round txn count is updated
	rtcKey = transaction.BuildSummaryRoundKey(100)
	newRtc := transaction.RoundTxnsCount{}
	err = transactionSummaryMetadata.GetStore().Read(ctx, rtcKey, &newRtc)
	require.NoError(t, err)
	require.Equal(t, int64(4), newRtc.TxnsCount)

	// Test getTxnCountForRound
	count, err := chain.getTxnCountForRound(ctx, 100)
	require.NoError(t, err)
	require.Equal(t, 4, count)
}