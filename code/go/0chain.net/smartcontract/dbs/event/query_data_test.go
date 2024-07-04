package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryDataForMiner(t *testing.T) {

	eventDb, clean := GetTestEventDB(t)
	defer clean()

	err := eventDb.AutoMigrate()
	require.NoError(t, err)
	// config.Configuration().ChainConfig = &TestConfig{conf: &TestConfigData{DbsSettings: config.DbSettings{
	// 	AggregatePeriod:       10,
	// 	PartitionKeepCount:    10,
	// 	PartitionChangePeriod: 100,
	// }}}
	// assert.NoError(t, err, "error while migrating database")
	_ = createMiners(t, eventDb, 10)

	t.Run("TestQueryDataForMiner", func(t *testing.T) {
		miners, err := eventDb.GetQueryData("id,host,port", &Miner{})
		assert.NoError(t, err, "error while fetching data")
		assert.NotNil(t, miners, "miners should not be nil")
		assert.Len(t, miners, 10, "miners should have 10 records")
	})

}
