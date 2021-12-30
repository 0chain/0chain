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

func TestMiners(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")

	type Stat struct {
		// for miner (totals)
		GeneratorRewards state.Balance `json:"generator_rewards,omitempty"`
		GeneratorFees    state.Balance `json:"generator_fees,omitempty"`
		// for sharder (totals)
		SharderRewards state.Balance `json:"sharder_rewards,omitempty"`
		SharderFees    state.Balance `json:"sharder_fees,omitempty"`
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

	type MinerNode struct {
		*SimpleNode `json:"simple_miner"`
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
			TotalStaked:       state.Balance(mn.TotalStaked),
			Delete:            mn.Delete,
			DelegateWallet:    mn.DelegateWallet,
			ServiceCharge:     mn.ServiceCharge,
			NumberOfDelegates: mn.NumberOfDelegates,
			MinStake:          mn.MinStake,
			MaxStake:          mn.MaxStake,
			LastHealthCheck:   mn.LastHealthCheck,
			Rewards:           mn.Stat.GeneratorRewards,
			Fees:              mn.Stat.GeneratorFees,
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
		Type:        int(TypeStats),
		Tag:         int(TagAddMiner),
		Data:        string(data),
	}
	events := []Event{eventAddMn}
	eventDb.AddEvents(events)

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
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteMiner),
		Data:        string(data),
	}
	eventDb.AddEvents([]Event{eventAddOrOverwriteMn})

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
		Type:        int(TypeStats),
		Tag:         int(TagUpdateMiner),
		Data:        string(data),
	}
	eventDb.AddEvents([]Event{eventUpdateMn})

	miner, err = eventDb.GetMiner(mn.ID)
	require.NoError(t, err)
	require.EqualValues(t, miner.Path, update.Updates["path"])
	require.EqualValues(t, miner.ShortName, update.Updates["short_name"])

	// Miner - Delete Event
	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        int(TypeStats),
		Tag:         int(TagDeleteMiner),
		Data:        mn.ID,
	}
	eventDb.AddEvents([]Event{deleteEvent})

	miner, err = eventDb.GetMiner(mn.ID)
	require.Error(t, err)

}
