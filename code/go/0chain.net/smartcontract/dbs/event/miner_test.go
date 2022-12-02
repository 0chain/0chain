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
	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/logging"
	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestMiners(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")

	type Stat struct {
		// for miner (totals)
		GeneratorRewards currency.Coin `json:"generator_rewards,omitempty"`
		GeneratorFees    currency.Coin `json:"generator_fees,omitempty"`
		// for sharder (totals)
		SharderRewards currency.Coin `json:"sharder_rewards,omitempty"`
		SharderFees    currency.Coin `json:"sharder_fees,omitempty"`
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
		// MinStake allowed by node.
		MinStake currency.Coin `json:"min_stake"`
		// MaxStake allowed by node.
		MaxStake currency.Coin `json:"max_stake"`

		// Stat contains node statistic.
		Stat Stat `json:"stat"`

		// NodeType used for delegate pools statistic.
		NodeType NodeType `json:"node_type,omitempty"`

		// LastHealthCheck used to check for active node
		LastHealthCheck common.Timestamp `json:"last_health_check"`
	}

	type MinerNode struct {
		*SimpleNode
	}

	convertMn := func(mn MinerNode) Miner {
		return Miner{
			MinerID:           mn.ID,
			N2NHost:           mn.N2NHost,
			Host:              mn.Host,
			Port:              mn.Port,
			Path:              mn.Path,
			PublicKey:         mn.PublicKey,
			ShortName:         mn.ShortName,
			BuildTag:          mn.BuildTag,
			TotalStaked:       currency.Coin(mn.TotalStaked),
			Delete:            mn.Delete,
			DelegateWallet:    mn.DelegateWallet,
			ServiceCharge:     mn.ServiceCharge,
			NumberOfDelegates: mn.NumberOfDelegates,
			MinStake:          mn.MinStake,
			MaxStake:          mn.MaxStake,
			LastHealthCheck:   mn.LastHealthCheck,
			Rewards: ProviderRewards{
				ProviderID: mn.ID,
				Rewards:    mn.Stat.GeneratorRewards,
			},
			Fees:      mn.Stat.GeneratorFees,
			Longitude: 0,
			Latitude:  0,
		}
	}

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
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	// Miner - Add Event
	mn := MinerNode{
		&SimpleNode{
			ID:                "miner one",
			N2NHost:           "n2n one host",
			Host:              "miner one host",
			Port:              1999,
			Path:              "path miner one",
			PublicKey:         "pub key",
			ShortName:         "mo",
			BuildTag:          "build tag",
			TotalStaked:       51,
			Delete:            false,
			DelegateWallet:    "delegate wallet",
			ServiceCharge:     10.6,
			NumberOfDelegates: 6,
			MinStake:          15,
			MaxStake:          100,
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

	mnMiner := convertMn(mn)
	data, err := json.Marshal(&mnMiner)
	require.NoError(t, err)

	eventAddMn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash",
		Type:        TypeStats,
		Tag:         TagAddOrOverwriteMiner,
		Data:        string(data),
	}
	events := []Event{eventAddMn}
	eventDb.ProcessEvents(context.TODO(), events, 100, "hash", 10)
	time.Sleep(100 * time.Millisecond)
	miner, err := eventDb.GetMiner(mn.ID)
	require.NoError(t, err)
	require.EqualValues(t, miner.Path, mn.Path)

	// Miner - Overwrite event
	mn.SimpleNode.Path = "path miner one - overwrite"

	mnMiner2 := convertMn(mn)
	data, err = json.Marshal(&mnMiner2)
	require.NoError(t, err)

	eventAddOrOverwriteMn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash2",
		Type:        TypeStats,
		Tag:         TagAddOrOverwriteMiner,
		Data:        string(data),
	}
	eventDb.ProcessEvents(context.TODO(), []Event{eventAddOrOverwriteMn}, 100, "hash", 10)

	miner, err = eventDb.GetMiner(mn.ID)
	require.NoError(t, err)
	require.EqualValues(t, miner.Path, mn.Path)

	// Miner - Update event
	update := dbs.DbUpdates{
		Id: mn.ID,
		Updates: map[string]interface{}{
			"path":       "new path",
			"short_name": "new short name",
		},
	}
	data, err = json.Marshal(&update)
	require.NoError(t, err)

	eventUpdateMn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash3",
		Type:        TypeStats,
		Tag:         TagUpdateMiner,
		Data:        string(data),
	}
	eventDb.ProcessEvents(context.TODO(), []Event{eventUpdateMn}, 100, "bhash", 10)

	miner, err = eventDb.GetMiner(mn.ID)
	require.NoError(t, err)
	require.EqualValues(t, miner.Path, update.Updates["path"])
	require.EqualValues(t, miner.ShortName, update.Updates["short_name"])

	// Miner - Delete Event
	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        TypeStats,
		Tag:         TagDeleteMiner,
		Data:        mn.ID,
	}
	eventDb.ProcessEvents(context.TODO(), []Event{deleteEvent}, 100, "bhash", 10)

	miner, err = eventDb.GetMiner(mn.ID)
	assert.Error(t, err)

}

func TestGetMiners(t *testing.T) {
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
	assert.NoError(t, err, "error while migrating database")
	createMiners(t, eventDb, 10)

	t.Run("Inactive miners should be returned", func(t *testing.T) {
		miners, err := eventDb.GetMinersWithFiltersAndPagination(MinerQuery{Active: null.BoolFrom(false)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "Error should not be returned")
		for _, miner := range miners {
			assert.Equal(t, false, miner.Active, "Miner is active")
		}
		assert.Equal(t, 5, len(miners), "All miners were not returned")
	})
	t.Run("Active miners should be returned", func(t *testing.T) {
		miners, err := eventDb.GetMinersWithFiltersAndPagination(MinerQuery{Active: null.BoolFrom(true)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "Error should not be returned")
		for _, miner := range miners {
			assert.Equal(t, true, miner.Active, "Miner is not active")
		}
		assert.Equal(t, 5, len(miners), "All miners were not returned")
	})
}

func TestGetMinerLocations(t *testing.T) {
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
	createMinersWithLocation(t, eventDb, 12)
	t.Run("miner locations without any filters", func(t *testing.T) {
		locations, err := eventDb.GetMinerGeolocations(MinerQuery{}, common2.Pagination{})
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 12, len(locations), "all miners should be returned")
		for _, location := range locations {
			id, err := strconv.ParseInt(location.MinerID, 10, 0)
			assert.NoError(t, err, "miner id should be parsed to integer")
			assert.Equal(t, location.Longitude, float64(100+id), "longitude should match")
			assert.Equal(t, location.Latitude, float64(100-id), "longitude should match")
		}
	})
	t.Run("locations for miners which are active", func(t *testing.T) {
		locations, err := eventDb.GetMinerGeolocations(MinerQuery{Active: null.BoolFrom(true)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 6, len(locations), "locations of only active miners should be returned")
		for _, location := range locations {
			id, err := strconv.ParseInt(location.MinerID, 10, 0)
			assert.NoError(t, err, "miner id should be parsed to integer")
			assert.Equal(t, location.Longitude, float64(100+id), "longitude should match")
			assert.Equal(t, location.Latitude, float64(100-id), "longitude should match")
		}
	})
	t.Run("locations for miners which are inactive", func(t *testing.T) {
		locations, err := eventDb.GetMinerGeolocations(MinerQuery{Active: null.BoolFrom(false)}, common2.Pagination{Limit: 10})
		assert.NoError(t, err, "There should be no error")
		assert.Equal(t, 6, len(locations), "locations of only active miners should be returned")
		for _, location := range locations {
			id, err := strconv.ParseInt(location.MinerID, 10, 0)
			assert.NoError(t, err, "miner id should be parsed to integer")
			assert.Equal(t, location.Longitude, float64(100+id), "longitude should match")
			assert.Equal(t, location.Latitude, float64(100-id), "longitude should match")
		}
	})
}

func createMiners(t *testing.T, eventDb *EventDb, count int) {
	for i := 0; i < count; i++ {
		m := Miner{
			MinerID:           fmt.Sprintf("bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d%v", i),
			N2NHost:           "198.18.0.73",
			Host:              "198.18.0.73",
			Port:              7073,
			PublicKey:         "aa182e7f1aa1cfcb6cad1e2cbf707db43dbc0afe3437d7d6c657e79cca732122f02a8106891a78b3ebaa2a37ebd148b7ef48f5c0b1b3311094b7f15a1bd7de12",
			ShortName:         "localhost.m2",
			BuildTag:          "d4b6b52f17b87d7c090d5cac29c6bfbf1051c820",
			Delete:            false,
			DelegateWallet:    "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8",
			ServiceCharge:     0.1,
			NumberOfDelegates: 10,
			MinStake:          0,
			MaxStake:          1000000000000,
			LastHealthCheck:   1644881505,
			Rewards: ProviderRewards{
				ProviderID: fmt.Sprintf("bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d%v", i),
				Rewards:    9725520000000,
			},
			Active: i%2 == 0,
		}
		err := eventDb.addOrOverwriteMiner([]Miner{m})
		assert.NoError(t, err, "inserting miners failed")
	}
}

func createMinersWithLocation(t *testing.T, eventDb *EventDb, count int) {
	for i := 0; i < count; i++ {
		s := Miner{Active: i%2 == 0, MinerID: fmt.Sprintf("%d", i), Longitude: float64(100 + i), Latitude: float64(100 - i)}
		err := eventDb.addOrOverwriteMiner([]Miner{s})
		assert.NoError(t, err, "There should be no error")
	}
}

func compareMiners(t *testing.T, miners []Miner, offset, limit int) {
	for i := offset; i < offset+limit; i++ {
		want := Miner{
			MinerID:           fmt.Sprintf("bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d%v", i),
			N2NHost:           "198.18.0.73",
			Host:              "198.18.0.73",
			Port:              7073,
			PublicKey:         "aa182e7f1aa1cfcb6cad1e2cbf707db43dbc0afe3437d7d6c657e79cca732122f02a8106891a78b3ebaa2a37ebd148b7ef48f5c0b1b3311094b7f15a1bd7de12",
			ShortName:         "localhost.m2",
			BuildTag:          "d4b6b52f17b87d7c090d5cac29c6bfbf1051c820",
			Delete:            false,
			DelegateWallet:    "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8",
			ServiceCharge:     0.1,
			NumberOfDelegates: 10,
			MinStake:          0,
			MaxStake:          1000000000000,
			LastHealthCheck:   1644881505,
			Rewards: ProviderRewards{
				ProviderID: fmt.Sprintf("bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d%v", i),
				Rewards:    9725520000000,
			},
			Active: i%2 == 0,
		}
		want.CreatedAt = miners[i].CreatedAt
		want.ID = miners[i].ID
		want.UpdatedAt = miners[i].UpdatedAt
		assert.Equal(t, want, miners[i], "Miners did not match")
	}
}

func BenchmarkReturnValueMiner(t *testing.B) {
	for i := 0; i < t.N; i++ {
		mi := ReturnValue()
		assert.Equal(t, "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d", mi.MinerID)
		t.Log("")
	}
}

func BenchmarkReturnPointerValueMiner(t *testing.B) {
	for i := 0; i < t.N; i++ {
		mi := ReturnPointer()
		assert.Equal(t, "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d", mi.MinerID)
		t.Log("")
	}
}

func (edb *EventDb) GetMinerPointer(id string) (*Miner, error) {
	miner := &Miner{}
	return miner, edb.Store.Get().
		Model(&Miner{}).
		Where(&Miner{MinerID: id}).
		First(miner).Error
}

func ReturnValue() Miner {
	return Miner{
		MinerID:           "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d",
		N2NHost:           "198.18.0.73",
		Host:              "198.18.0.73",
		Port:              7073,
		PublicKey:         "aa182e7f1aa1cfcb6cad1e2cbf707db43dbc0afe3437d7d6c657e79cca732122f02a8106891a78b3ebaa2a37ebd148b7ef48f5c0b1b3311094b7f15a1bd7de12",
		ShortName:         "localhost.m2",
		BuildTag:          "d4b6b52f17b87d7c090d5cac29c6bfbf1051c820",
		Delete:            false,
		DelegateWallet:    "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8",
		ServiceCharge:     0.1,
		NumberOfDelegates: 10,
		MinStake:          0,
		MaxStake:          1000000000000,
		LastHealthCheck:   1644881505,
		Rewards: ProviderRewards{
			ProviderID: "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d",
			Rewards:    9725520000000,
		},
		Active: true}
}

func ReturnPointer() *Miner {
	return &Miner{
		MinerID:           "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d",
		N2NHost:           "198.18.0.73",
		Host:              "198.18.0.73",
		Port:              7073,
		PublicKey:         "aa182e7f1aa1cfcb6cad1e2cbf707db43dbc0afe3437d7d6c657e79cca732122f02a8106891a78b3ebaa2a37ebd148b7ef48f5c0b1b3311094b7f15a1bd7de12",
		ShortName:         "localhost.m2",
		BuildTag:          "d4b6b52f17b87d7c090d5cac29c6bfbf1051c820",
		Delete:            false,
		DelegateWallet:    "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8",
		ServiceCharge:     0.1,
		NumberOfDelegates: 10,
		MinStake:          0,
		MaxStake:          1000000000000,
		LastHealthCheck:   1644881505,
		Rewards: ProviderRewards{
			ProviderID: "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d",
			Rewards:    9725520000000,
		},
		Active: true}
}
