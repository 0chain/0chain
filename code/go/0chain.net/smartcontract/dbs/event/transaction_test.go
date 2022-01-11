package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAddTransaction(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := dbs.DbAccess{
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
	err = eventDb.addTransaction(tr)
	require.NoError(t, err, "Error while inserting Transaction to event Database")
	var count int64
	eventDb.Get().Table("transactions").Count(&count)
	require.Equal(t, int64(1), count, "Transaction not getting inserted")
	err = eventDb.drop()
	require.NoError(t, err)
}

func TestFindTransaction(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
	access := dbs.DbAccess{
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
	defer eventDb.drop()
	require.NoError(t, err)

	tr := Transaction{
		Model: gorm.Model{ID: 1},
		Hash:  "something",
	}
	err = eventDb.addTransaction(tr)
	require.NoError(t, err, "Error while inserting Transaction to event Database")
	gotTr, err := eventDb.GetTransactionByHash("something")

	// To ignore createdAt and updatedAt
	tr.CreatedAt = gotTr.CreatedAt
	tr.UpdatedAt = gotTr.UpdatedAt
	require.Equal(t, tr, gotTr, "Transaction not getting inserted")

	gotTr, err = eventDb.GetTransactionByHash("some")
	require.Error(t, err, "issue while getting the transaction by hash")
}
