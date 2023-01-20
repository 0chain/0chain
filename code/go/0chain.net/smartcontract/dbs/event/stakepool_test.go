package event

import (
	"testing"

	"0chain.net/core/common"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/currency"
	"github.com/stretchr/testify/assert"
)

func init() {
	viper.Set("logging.console", true)
	viper.Set("logging.level", "debug")
}

func TestEventDb_rewardProviders(t *testing.T) {
	db, clean := GetTestEventDB(t)
	defer clean()

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

			N2NHost:   mn.N2NHost,
			Host:      mn.Host,
			Port:      mn.Port,
			Path:      mn.Path,
			PublicKey: mn.PublicKey,
			ShortName: mn.ShortName,
			BuildTag:  mn.BuildTag,
			Delete:    mn.Delete,
			Provider: Provider{
				ID:             mn.ID,
				TotalStake:     currency.Coin(mn.TotalStaked),
				DelegateWallet: mn.DelegateWallet,
				ServiceCharge:  mn.ServiceCharge,
				NumDelegates:   mn.NumberOfDelegates,
				MinStake:       mn.MinStake,
				MaxStake:       mn.MaxStake,
				Rewards: ProviderRewards{
					ProviderID: mn.ID,
					Rewards:    mn.Stat.GeneratorRewards,
				},
				LastHealthCheck: mn.LastHealthCheck,
			},

			Fees:      mn.Stat.GeneratorFees,
			Longitude: 0,
			Latitude:  0,
		}
	}
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
	// Miner - Add Event
	mn2 := MinerNode{
		&SimpleNode{
			ID:                "miner two",
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

	mnMiner1 := convertMn(mn)
	mnMiner2 := convertMn(mn2)

	if err := db.addOrOverwriteMiner([]Miner{mnMiner1, mnMiner2}); err != nil {
		t.Error(err)
	}

	if err := db.rewardProviders([]ProviderRewards{{
		ProviderID:   mnMiner1.ID,
		Rewards:      20,
		TotalRewards: 40,
	}, {
		ProviderID:   mnMiner2.ID,
		Rewards:      30,
		TotalRewards: 50,
	}}); err != nil {
		t.Error(err)
	}

	assertMinerRewards(t, db, mnMiner1.ID, int64(20+5), int64(40))
	assertMinerRewards(t, db, mnMiner2.ID, int64(30+5), int64(50))
}

func assertMinerRewards(t *testing.T, eventDb *EventDb, minerId string, reward, totalReward int64) {
	miner, err := eventDb.GetMiner(minerId)
	if err != nil {
		t.Error(err)
	}

	rt1, err := miner.Rewards.TotalRewards.Int64()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, int64(totalReward), rt1)

	r1, err := miner.Rewards.Rewards.Int64()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, int64(reward), r1)
}
