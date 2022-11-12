package event

import (
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAddTransaction(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	tr := Transaction{}
	err = eventDb.addTransactions([]Transaction{tr})
	require.NoError(t, err, "Error while inserting Transaction to event Database")
	var count int64
	eventDb.Get().Table("transactions").Count(&count)
	require.Equal(t, int64(1), count, "Transaction not getting inserted")
	err = eventDb.Drop()
	require.NoError(t, err)
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		return
	}
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer eventDb.Drop()
	require.NoError(t, err)

	tr := Transaction{
		Model:     gorm.Model{ID: 1},
		Hash:      "something_0",
		ClientId:  "someClientID",
		BlockHash: "blockHash",
	}
	SetUpTransactionData(t, eventDb)
	t.Run("GetTransactionByHash", func(t *testing.T) {
		gotTr, err := eventDb.GetTransactionByHash("something_0")

		// To ignore createdAt and updatedAt
		tr.Model.ID = gotTr.ID
		tr.CreatedAt = gotTr.CreatedAt
		tr.UpdatedAt = gotTr.UpdatedAt
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
		tr.Model.ID = gotTr[i].ID
		tr.CreatedAt = gotTr[i].CreatedAt
		tr.UpdatedAt = gotTr[i].UpdatedAt
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
