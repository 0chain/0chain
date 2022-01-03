package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
	"time"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestSharders(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")

	type Stat struct {
		GeneratorRewards state.Balance `json:"generator_rewards,omitempty"`
		GeneratorFees    state.Balance `json:"generator_fees,omitempty"`
		SharderRewards   state.Balance `json:"sharder_rewards,omitempty"`
		SharderFees      state.Balance `json:"sharder_fees,omitempty"`
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
		MinStake state.Balance `json:"min_stake"`
		// MaxStake allowed by node.
		MaxStake state.Balance `json:"max_stake"`

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
			SharderID:         sn.ID,
			N2NHost:           sn.N2NHost,
			Host:              sn.Host,
			Port:              sn.Port,
			Path:              sn.Path,
			PublicKey:         sn.PublicKey,
			ShortName:         sn.ShortName,
			BuildTag:          sn.BuildTag,
			TotalStaked:       state.Balance(sn.TotalStaked),
			Delete:            sn.Delete,
			DelegateWallet:    sn.DelegateWallet,
			ServiceCharge:     sn.ServiceCharge,
			NumberOfDelegates: sn.NumberOfDelegates,
			MinStake:          sn.MinStake,
			MaxStake:          sn.MaxStake,
			LastHealthCheck:   sn.LastHealthCheck,
			Rewards:           sn.Stat.GeneratorRewards,
			Fees:              sn.Stat.GeneratorFees,
			Longitude:         0,
			Latitude:          0,
		}
	}

	access := dbs.DbAccess{
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

	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.drop()
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

	snSharder := convertSn(sn)
	data, err := json.Marshal(&snSharder)
	require.NoError(t, err)

	eventAddSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash",
		Type:        int(TypeStats),
		Tag:         int(TagAddSharder),
		Data:        string(data),
	}
	events := []Event{eventAddSn}
	eventDb.AddEvents(events)

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
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteSharder),
		Data:        string(data),
	}
	eventDb.AddEvents([]Event{eventAddOrOverwriteSn})

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
		Type:        int(TypeStats),
		Tag:         int(TagUpdateSharder),
		Data:        string(data),
	}
	eventDb.AddEvents([]Event{eventUpdateSn})

	sharder, err = eventDb.GetSharder(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, sharder.Path, update.Updates["path"])
	require.EqualValues(t, sharder.ShortName, update.Updates["short_name"])

	// Sharder - Delete Event
	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        int(TypeStats),
		Tag:         int(TagDeleteSharder),
		Data:        sn.ID,
	}
	eventDb.AddEvents([]Event{deleteEvent})

	sharder, err = eventDb.GetSharder(sn.ID)
	require.Error(t, err)

}
