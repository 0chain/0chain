package event

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestGetSharderWithDelegatePools(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	createSharders(t, edb, 2)

	err := edb.addDelegatePools([]DelegatePool{
		{

			PoolID:       "pool_id",
			ProviderType: spenum.Sharder,
			ProviderID:   "1",
			DelegateID:   "delegate_id",

			Balance: 30,
		},
		{
			PoolID:       "pool_id_2",
			ProviderType: spenum.Sharder,
			ProviderID:   "1",
			DelegateID:   "delegate_id_2",

			Balance: 30,
		},
	})
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	p, err := edb.GetDelegatePool("pool_id", "1")
	require.NoError(t, err, "Error while retrieving DelegatePool from event Database")
	require.Equal(t, p.PoolID, "pool_id")
	require.Equal(t, p.ProviderType, spenum.Sharder)
	require.Equal(t, p.ProviderID, "1")

	s, dps, err := edb.GetSharderWithDelegatePools("1")

	require.NoError(t, err, "Error while getting sharder with delegate pools")
	require.Equal(t, s.ID, "1")
	require.Equal(t, 2, len(dps))
	require.Equal(t, "1", dps[0].ProviderID) // failed, the provider id is ""
}

func TestGetSharderWithDelegatePoolsNoPools(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	createSharders(t, edb, 2)

	s, dps, err := edb.GetSharderWithDelegatePools("1")

	require.NoError(t, err, "Error while getting sharder with delegate pools")
	require.Nil(t, dps, "there should be no delegate pools")
	require.Equal(t, s.ID, "1")
}

func TestSharders(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")

	type Stat struct {
		GeneratorRewards currency.Coin `json:"generator_rewards,omitempty"`
		GeneratorFees    currency.Coin `json:"generator_fees,omitempty"`
		SharderRewards   currency.Coin `json:"sharder_rewards,omitempty"`
		SharderFees      currency.Coin `json:"sharder_fees,omitempty"`
	}

	type NodeType int

	type SimpleNode struct {
		ID          string `json:"id" validate:"hexadecimal,len=64"`
		N2NHost     string `json:"n2n_host"`
		Host        string `json:"host"`
		Port        int    `json:"port"`
		Path        string `json:"path"`
		PublicKey   string `json:"public_key"`
		ShortName   string `json:"short_name"`
		BuildTag    string `json:"build_tag"`
		TotalStaked int64  `json:"total_stake"`
		Delete      bool   `json:"delete"`

		// settings and statistic

		// DelegateWallet grabs node rewards (excluding stake rewards) and
		// controls the node setting. If the DelegateWallet hasn't been provided,
		// then node ID used (for genesis nodes, for example).
		DelegateWallet string `json:"delegate_wallet" validate:"omitempty,hexadecimal,len=64"` // ID
		// ServiceChange is % that miner node grabs where it's generator.
		ServiceCharge float64 `json:"service_charge"` // %
		// NumberOfDelegates is max allowed number of delegate pools.
		NumberOfDelegates int `json:"number_of_delegates"`

		// Stat contains node statistic.
		Stat Stat `json:"stat"`

		// NodeType used for delegate pools statistic.
		NodeType NodeType `json:"node_type,omitempty"`

		// LastHealthCheck used to check for active node
		LastHealthCheck common.Timestamp `json:"last_health_check"`
	}

	type SharderNode struct {
		*SimpleNode
	}

	convertSn := func(sn SharderNode) Sharder {
		return Sharder{

			N2NHost:   sn.N2NHost,
			Host:      sn.Host,
			Port:      sn.Port,
			Path:      sn.Path,
			PublicKey: sn.PublicKey,
			ShortName: sn.ShortName,
			BuildTag:  sn.BuildTag,
			Delete:    sn.Delete,
			Provider: Provider{
				ID:             sn.ID,
				TotalStake:     currency.Coin(sn.TotalStaked),
				DelegateWallet: sn.DelegateWallet,
				ServiceCharge:  sn.ServiceCharge,
				NumDelegates:   sn.NumberOfDelegates,
				Rewards: ProviderRewards{
					ProviderID: sn.ID,
					Rewards:    sn.Stat.GeneratorRewards,
				},
				LastHealthCheck: sn.LastHealthCheck,
			},

			Fees:      sn.Stat.GeneratorFees,
			Longitude: 0,
			Latitude:  0,
		}
	}

	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            "zchain_user",
		Password:        "zchian",
		Host:            "localhost",
		Port:            "5432",
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}

	eventDb, err := NewEventDb(access, config.DbSettings{})
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	// Sharder - Add Event
	sn := SharderNode{
		&SimpleNode{
			ID:                "sharder one",
			N2NHost:           "n2n one host",
			Host:              "sharder one host",
			Port:              1999,
			Path:              "path sharder one",
			PublicKey:         "pub key",
			ShortName:         "mo",
			BuildTag:          "build tag",
			TotalStaked:       51,
			Delete:            false,
			DelegateWallet:    "delegate wallet",
			ServiceCharge:     10.6,
			NumberOfDelegates: 6,
			Stat: Stat{
				GeneratorRewards: 5,
				GeneratorFees:    3,
				SharderRewards:   10,
				SharderFees:      5,
			},
			NodeType:        NodeType(1),
			LastHealthCheck: common.Timestamp(51),
		},
	}

	snSharder := convertSn(sn)
	data, err := json.Marshal(&snSharder)
	require.NoError(t, err)

	eventAddSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash",
		Type:        TypeStats,
		Tag:         TagAddSharder,
		Data:        string(data),
	}
	events := []Event{eventAddSn}
	eventDb.ProcessEvents(context.TODO(), events, 100, "hash", 10, CommitNow())

	sharder, err := eventDb.GetSharder(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, sharder.Path, sn.Path)

	// Sharder - Overwrite event
	sn.SimpleNode.Path = "path sharder one - overwrite"

	snSharder2 := convertSn(sn)
	data, err = json.Marshal(&snSharder2)
	require.NoError(t, err)

	eventAddOrOverwriteSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash2",
		Type:        TypeStats,
		Tag:         TagAddSharder,
		Data:        string(data),
	}
	eventDb.ProcessEvents(context.TODO(), []Event{eventAddOrOverwriteSn}, 100, "hash", 10, CommitNow())

	sharder, err = eventDb.GetSharder(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, sharder.Path, sn.Path)

	// Sharder - Update event
	update := dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"path":       "new path",
			"short_name": "new short name",
		},
	}
	data, err = json.Marshal(&update)
	require.NoError(t, err)

	eventUpdateSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash3",
		Type:        TypeStats,
		Tag:         TagUpdateSharder,
		Data:        string(data),
	}
	eventDb.ProcessEvents(context.TODO(), []Event{eventUpdateSn}, 100, "hash", 10, CommitNow())

	sharder, err = eventDb.GetSharder(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, sharder.Path, update.Updates["path"])
	require.EqualValues(t, sharder.ShortName, update.Updates["short_name"])

	// Sharder - Delete Event
	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        TypeStats,
		Tag:         TagDeleteSharder,
		Data:        sn.ID,
	}
	eventDb.ProcessEvents(context.TODO(), []Event{deleteEvent}, 100, "hash", 10, CommitNow())

	sharder, err = eventDb.GetSharder(sn.ID)
	require.Error(t, err)

}

func TestSharderFilter(t *testing.T) {
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
	defer func() {
		err = eventDb.Drop()
		assert.NoError(t, err, "error while dropping database")
	}()
	assert.NoError(t, err, "error while migrating database")
	createSharders(t, eventDb, 10)
	t.Run("sharders which are active", func(t *testing.T) {
		sharders, err := eventDb.GetShardersWithFilterAndPagination(SharderQuery{Active: null.BoolFrom(true)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "There should be no error")
		for _, sharder := range sharders {
			assert.Equal(t, true, sharder.Active, "all sharder should be active")
		}
		assert.Equal(t, 5, len(sharders), "only active sharders should be returned")
	})
	t.Run("sharders which are not active", func(t *testing.T) {
		sharders, err := eventDb.GetShardersWithFilterAndPagination(SharderQuery{Active: null.BoolFrom(false)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "There should be no error")
		for _, sharder := range sharders {
			assert.Equal(t, false, sharder.Active, "all sharder should be inactive")
		}
		assert.Equal(t, 5, len(sharders), "only inactive sharders should be returned")
	})
}

func TestGetSharderLocations(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
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
	eventDb, err := NewEventDb(access, config.DbSettings{})
	if err != nil {
		t.Skip("only for local debugging, requires local postgresql")
	}
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	defer func() {
		err = eventDb.Drop()
		assert.NoError(t, err, "error while dropping database")
	}()
	assert.NoError(t, err, "error while migrating database")
	createShardersWithLocation(t, eventDb, 12)
	t.Run("sharder locations without any filters", func(t *testing.T) {
		locations, err := eventDb.GetSharderGeolocations(SharderQuery{}, common2.Pagination{})
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 12, len(locations), "all sharders should be returned")
		for _, location := range locations {
			id, err := strconv.ParseInt(location.SharderID, 10, 0)
			assert.NoError(t, err, "sharder id should be parsed to integer")
			assert.Equal(t, location.Longitude, float64(100+id), "longitude should match")
			assert.Equal(t, location.Latitude, float64(100-id), "longitude should match")
		}
	})
	t.Run("locations for sharders which are active", func(t *testing.T) {
		locations, err := eventDb.GetSharderGeolocations(SharderQuery{Active: null.BoolFrom(true)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 6, len(locations), "locations of only active sharders should be returned")
		for _, location := range locations {
			id, err := strconv.ParseInt(location.SharderID, 10, 0)
			assert.NoError(t, err, "sharder id should be parsed to integer")
			assert.Equal(t, location.Longitude, float64(100+id), "longitude should match")
			assert.Equal(t, location.Latitude, float64(100-id), "longitude should match")
		}
	})
	t.Run("locations for sharders which are inactive", func(t *testing.T) {
		locations, err := eventDb.GetSharderGeolocations(SharderQuery{Active: null.BoolFrom(false)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 6, len(locations), "locations of only active sharders should be returned")
		for _, location := range locations {
			id, err := strconv.ParseInt(location.SharderID, 10, 0)
			assert.NoError(t, err, "sharder id should be parsed to integer")
			assert.Equal(t, location.Longitude, float64(100+id), "longitude should match")
			assert.Equal(t, location.Latitude, float64(100-id), "longitude should match")
		}
	})
}

func createSharders(t *testing.T, eventDb *EventDb, count int) {
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%d", i)
		s := Sharder{Active: i%2 == 0, Provider: Provider{ID: id}}
		err := eventDb.addSharders([]Sharder{s})
		assert.NoError(t, err, "There should be no error")

		require.NoError(t, eventDb.Get().Create(&ProviderRewards{
			ProviderID: id,
		}).Error)
	}
}

func createShardersWithLocation(t *testing.T, eventDb *EventDb, count int) {
	for i := 0; i < count; i++ {
		s := Sharder{Active: i%2 == 0, Provider: Provider{ID: fmt.Sprintf("%d", i)}, Longitude: float64(100 + i), Latitude: float64(100 - i)}
		err := eventDb.addSharders([]Sharder{s})
		assert.NoError(t, err, "There should be no error")
	}
}
