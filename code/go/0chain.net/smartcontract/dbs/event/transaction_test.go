package event

import (
	"fmt"
	"testing"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagAddTransactions(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	round := int64(7)

	transactionsEvents := []Event{
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagAddTransactions,
			Index:       "one",
			Data:        Transaction{Hash: "one", Fee: 3},
		},
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagAddTransactions,
			Index:       "one",
			Data:        Transaction{Hash: "one", Fee: 3},
		},
		{
			BlockNumber: round,
			TxHash:      "2",
			Type:        TypeStats,
			Tag:         TagAddTransactions,
			Index:       "two",
			Data:        Transaction{Hash: "two", Fee: 5},
		},
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagAddTransactions,
			Index:       "two",
			Data:        Transaction{Hash: "two", Fee: 7},
		},
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagAddTransactions,
			Index:       "three",
			Data:        Transaction{Hash: "three", Fee: 11},
		},
	}
	events, err := mergeEvents(round, "", transactionsEvents)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Len(t, events[0].Data, 3, "the five events should have been merged into three")

	require.NoError(t, edb.addStat(events[0]))

	var txs []Transaction
	edb.Get().Find(&txs)
	require.Len(t, txs, 3)
}

func TestFindTransactionByHash(t *testing.T) {
	eventDb, clean := GetTestEventDB(t)
	defer clean()

	// Add two blocks
	eventDb.addOrUpdateBlock(Block{Hash: "blockHash", Round: 7, IsFinalised: true})
	eventDb.addOrUpdateBlock(Block{Hash: "blockHash_unf", Round: 8, IsFinalised: false})

	tr := Transaction{
		ImmutableModel: model.ImmutableModel{ID: 1},
		Hash:           "something_0",
		ClientId:       "someClientID",
		ToClientId:     "someToClientID",
		BlockHash:      "blockHash",
	}
	SetUpTransactionData(t, eventDb)

	err := eventDb.addTransactions([]Transaction{
		// Differnt id
		{
			Hash: 	"something_0_difId",
			ClientId: "someClientID2",
			ToClientId: "someToClientID2",
			BlockHash: "blockHash",
			Round: 7,
		},
		{
			Hash: 	"something_1_difId",
			ClientId: "someClientID2",
			ToClientId: "someToClientID2",
			BlockHash: "blockHash",
			Round: 7,
		},
		{
			Hash: 	"something_2_difId",
			ClientId: "someClientID2",
			ToClientId: "someToClientID2",
			BlockHash: "blockHash",
			Round: 7,
		},
		{
			Hash: 	"something_3_difId",
			ClientId: "someClientID2",
			ToClientId: "someToClientID2",
			BlockHash: "blockHash",
			Round: 7,
		},
		{
			Hash: 	"something_4_difId",
			ClientId: "someClientID2",
			ToClientId: "someToClientID2",
			BlockHash: "blockHash",
			Round: 7,
		},
		// Unfinalized block
		{
			Hash: 	"something_0_unf",
			ClientId: "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash_unf",
			Round: 7,
		},
		{
			Hash: 	"something_1_unf",
			ClientId: "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash_unf",
			Round: 7,
		},
		{
			Hash: 	"something_2_unf",
			ClientId: "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash_unf",
			Round: 7,
		},
		{
			Hash: 	"something_3_unf",
			ClientId: "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash_unf",
			Round: 7,
		},
		{
			Hash: 	"something_4_unf",
			ClientId: "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash_unf",
			Round: 7,
		},
	})
	require.NoError(t, err, "Error while inserting Transaction to event Database")

	t.Run("GetTransactions", func(t *testing.T) {
		gotTrs, err := eventDb.GetTransactions(common.Pagination{Limit: 30, IsDescending: false})
		require.NoError(t, err)
		require.Len(t, gotTrs, 15, "All transactions of finalized blocks should be returned")
		for _, tr := range gotTrs {
			require.Equal(t, "blockHash", tr.BlockHash, "All transactions should be from finalized blocks")
		}
	})

	t.Run("GetTransactionByHash", func(t *testing.T) {
		gotTr, err := eventDb.GetTransactionByHash("something_0")

		// To ignore createdAt and updatedAt
		tr.ImmutableModel.ID = gotTr.ID
		tr.CreatedAt = gotTr.CreatedAt
		require.Equal(t, tr, gotTr, "Transaction not getting inserted")
		gotTr, err = eventDb.GetTransactionByHash("some")
		require.Error(t, err, "issue while getting the transaction by hash")
	})

	t.Run("GetTransactionByClientId", func(t *testing.T) {
		gotTrs, err := eventDb.GetTransactionByClientId("someClientID", common.Pagination{Limit: 20, IsDescending: false})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 10)

		gotTrs, err = eventDb.GetTransactionByClientId("wrongID", common.Pagination{Limit: 20, IsDescending: false})
		require.NoError(t, err)
		require.Equal(t, len(gotTrs), 0, "No Transaction should be returned")

		gotTrs, err = eventDb.GetTransactionByClientId("someClientID", common.Pagination{Limit: 5, IsDescending: false})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 5)

		gotTrs, err = eventDb.GetTransactionByClientId("someClientID", common.Pagination{Offset: 5, Limit: 20, IsDescending: false})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 5, 5)

	})

	t.Run("GetTransactionByToClientId", func(t *testing.T) {
		gotTrs, err := eventDb.GetTransactionByToClientId("someToClientID", common.Pagination{Limit: 20, IsDescending: false})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 10)

		gotTrs, err = eventDb.GetTransactionByToClientId("wrongID", common.Pagination{Limit: 20, IsDescending: false})
		require.NoError(t, err)
		require.Equal(t, len(gotTrs), 0, "No Transaction should be returned")

		gotTrs, err = eventDb.GetTransactionByToClientId("someToClientID", common.Pagination{Limit: 5, IsDescending: false})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 5)

		gotTrs, err = eventDb.GetTransactionByToClientId("someToClientID", common.Pagination{Offset: 5, Limit: 20, IsDescending: false})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 5, 5)
	})

	t.Run("GetTransactionByBlockHash", func(t *testing.T) {
		gotTrs, err := eventDb.GetTransactionByBlockHash("blockHash", common.Pagination{Limit: 20, IsDescending: false})
		require.NoError(t, err)
		require.Len(t, gotTrs, 15, "All transactions of blockHash should be returned")
		for _, tr := range gotTrs {
			require.Equal(t, "blockHash", tr.BlockHash, "All transactions should be from blockHash")
		}

		gotTrs, err = eventDb.GetTransactionByBlockHash("wrongHash", common.Pagination{Limit: 10, IsDescending: false})
		assert.NoError(t, err)
		require.Equal(t, len(gotTrs), 0, "No Transaction should be returned")

		gotTrs, err = eventDb.GetTransactionByBlockHash("blockHash", common.Pagination{Limit: 5, IsDescending: false})
		assert.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 5)

		gotTrs, err = eventDb.GetTransactionByBlockHash("blockHash", common.Pagination{Offset: 5, Limit: 5, IsDescending: false})
		assert.NoError(t, err)
		compareTransactions(t, gotTrs, 5, 5)
	})

}

func compareTransactions(t *testing.T, gotTr []Transaction, offset, limit int) {
	require.Equal(t, limit, len(gotTr), "Not all transactions were returned")
	i := 0
	for i = offset; i < limit; i++ {
		tr := Transaction{
			Hash:      fmt.Sprintf("something_%d", i),
			ClientId:  "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash",
		}
		tr.ImmutableModel.ID = gotTr[i].ID
		tr.CreatedAt = gotTr[i].CreatedAt
		require.Equal(t, tr, gotTr[i], "Transaction not matching")
	}
}

func SetUpTransactionData(t *testing.T, eventDb *EventDb) {
	for i := 0; i < 10; i++ {
		tr := Transaction{
			Hash:      fmt.Sprintf("something_%d", i),
			ClientId:  "someClientID",
			ToClientId: "someToClientID",
			BlockHash: "blockHash",
		}
		err := eventDb.addTransactions([]Transaction{tr})
		require.NoError(t, err, "Error while inserting Transaction to event Database")
	}
}
