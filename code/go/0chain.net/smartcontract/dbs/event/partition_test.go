package event

import (
	"testing"

	"0chain.net/core/config"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
)

func init() {
	viper.Set("logging.console", true)
	viper.Set("logging.level", "debug")
}

const req = `SELECT
    child.relname       AS child
FROM pg_inherits
    JOIN pg_class parent            ON pg_inherits.inhparent = parent.oid
    JOIN pg_class child             ON pg_inherits.inhrelid   = child.oid
    JOIN pg_namespace nmsp_parent   ON nmsp_parent.oid  = parent.relnamespace
    JOIN pg_namespace nmsp_child    ON nmsp_child.oid   = child.relnamespace
WHERE parent.relname='blobber_aggregates'`

func TestPartitionCreate(t *testing.T) {
	logging.InitLogging("development", "")

	config.Configuration().ChainConfig = &TestConfig{conf: &TestConfigData{DbsSettings: config.DbSettings{
		AggregatePeriod:              10,
		PartitionKeepCount:           10,
		PartitionChangePeriod:        100,
		RollingPartitionChangePeriod: 10,
		RollingPartitionKeepCount:    1,
	}}}

	db, f := GetTestEventDB(t)
	defer f()
	err := db.addRollingPartition(11, "blobber_aggregates")
	require.NoError(t, err)
	err = db.addRollingPartition(101, "blobber_aggregates")
	require.NoError(t, err)
	err = db.addRollingPartition(201, "blobber_aggregates")
	require.NoError(t, err)

	var partitions []string
	db.Store.Get().Raw(req).Scan(&partitions)
	require.Equal(t, 13, len(partitions))

	err = db.dropRollingPartition(201, "blobber_aggregates")
	require.NoError(t, err)

	db.Store.Get().Raw(req).Scan(&partitions)
	require.Equal(t, 13, len(partitions))
}
