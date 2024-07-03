package event

import (
	"os"
	"testing"
	"time"

	"0chain.net/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryDataForMiner(t *testing.T) {

	access := config.DbAccess{
		Enabled:         true,
		Name:            os.Getenv("POSTGRES_DB"),
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 100 * time.Second,
	}
	eventDb, err := NewInMemoryEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)
	config.Configuration().ChainConfig = &TestConfig{conf: &TestConfigData{DbsSettings: config.DbSettings{
		AggregatePeriod:       10,
		PartitionKeepCount:    10,
		PartitionChangePeriod: 100,
	}}}
	assert.NoError(t, err, "error while migrating database")
	_ = createMiners(t, eventDb, 10)

	t.Run("TestQueryDataForMiner", func(t *testing.T) {
		miners, err := eventDb.GetQueryData("id,host,port", &Miner{})
		assert.NoError(t, err, "error while fetching data")
		assert.NotNil(t, miners, "miners should not be nil")
		assert.Len(t, miners, 10, "miners should have 10 records")
	})

}
