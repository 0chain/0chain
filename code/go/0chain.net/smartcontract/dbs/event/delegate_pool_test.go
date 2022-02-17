package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
)

func TestDelegatePoolsEvent(t *testing.T) {
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

	dp := DelegatePool{
		PoolID:       "ool_id",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",

		Balance: 29,
	}

	err = eventDb.addOrOverwriteDelegatePool(dp)
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	var count int64
	eventDb.Get().Table("delegate_pool").Count(&count)
	require.Equal(t, int64(1), count, "Delegate pool not getting inserted")

	err = eventDb.drop()
	require.NoError(t, err)
}
