package event

import (
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
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
	access := config.DbAccess{
		Enabled:         true,
		Name:            os.Getenv("POSTGRES_DB"),
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	t.Skip("only for local debugging, requires local postgresql")
	eventDb, err := NewEventDb(access, config.DbSettings{})
	if err != nil {
		return
	}
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer eventDb.Drop()
	require.NoError(t, err)

	tr := Transaction{
		ImmutableModel: model.ImmutableModel{ID: 1},
		Hash:           "something_0",
		ClientId:       "someClientID",
		BlockHash:      "blockHash",
	}
	SetUpTransactionData(t, eventDb)
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
		gotTrs, err := eventDb.GetTransactionByClientId("someClientID", common.Pagination{Limit: 10, IsDescending: true})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 10)

		gotTrs, err = eventDb.GetTransactionByClientId("someClient", common.Pagination{Limit: 10, IsDescending: true})
		require.NoError(t, err)
		require.Equal(t, len(gotTrs), 0, "No Transaction should be returned")

		gotTrs, err = eventDb.GetTransactionByClientId("someClientID", common.Pagination{Limit: 5, IsDescending: true})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 5)

		gotTrs, err = eventDb.GetTransactionByClientId("someClientID", common.Pagination{Offset: 5, Limit: 5, IsDescending: true})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 5, 5)

	})

	t.Run("GetTransactionByBlockHash", func(t *testing.T) {
		gotTrs, err := eventDb.GetTransactionByBlockHash("blockHash", common.Pagination{Limit: 10, IsDescending: true})
		require.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 10)

		gotTrs, err = eventDb.GetTransactionByBlockHash("someHash", common.Pagination{Limit: 10, IsDescending: true})
		assert.NoError(t, err)
		require.Equal(t, len(gotTrs), 0, "No Transaction should be returned")

		gotTrs, err = eventDb.GetTransactionByBlockHash("blockHash", common.Pagination{Limit: 5, IsDescending: true})
		assert.NoError(t, err)
		compareTransactions(t, gotTrs, 0, 5)

		gotTrs, err = eventDb.GetTransactionByBlockHash("blockHash", common.Pagination{Offset: 5, Limit: 5, IsDescending: true})
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
			BlockHash: "blockHash",
		}
		err := eventDb.addTransactions([]Transaction{tr})
		require.NoError(t, err, "Error while inserting Transaction to event Database")
	}
}
