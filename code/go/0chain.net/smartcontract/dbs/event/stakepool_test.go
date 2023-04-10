package event

import (
	"fmt"
	"testing"

	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm/clause"

	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/stretchr/testify/require"

	"0chain.net/core/common"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/currency"
	"github.com/stretchr/testify/assert"
)

func init() {
	viper.Set("logging.console", true)
	viper.Set("logging.level", "debug")
}

func TestStakePoolReward(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	previousSpus := []ProviderRewards{
		{
			ProviderID:   "provider_1",
			Rewards:      17,
			TotalRewards: 139,
		},
		{
			ProviderID:   "provider_2",
			Rewards:      23,
			TotalRewards: 269,
		},
	}
	res := edb.Get().Create(&previousSpus)
	require.NoError(t, res.Error)
	var oldPrs []ProviderRewards
	ret := edb.Get().Find(&oldPrs)
	require.NoError(t, ret.Error)

	spus := []dbs.StakePoolReward{
		{
			ProviderID: dbs.ProviderID{
				ID:   "provider_1",
				Type: spenum.Miner,
			},
			Reward:     168,
			RewardType: spenum.BlockRewardSharder,
		},
		{
			ProviderID: dbs.ProviderID{
				ID:   "provider_1",
				Type: spenum.Miner,
			},
			Reward:     80,
			RewardType: spenum.FeeRewardMiner,
		},
		{
			ProviderID: dbs.ProviderID{
				ID:   "provider_2",
				Type: spenum.Sharder,
			},
			Reward:     100,
			RewardType: spenum.BlockRewardSharder,
		},
		{
			ProviderID: dbs.ProviderID{
				ID:   "provider_2",
				Type: spenum.Sharder,
			},
			Reward:     21,
			RewardType: spenum.FeeRewardSharder,
		},
	}
	err := edb.rewardUpdate(spus, 9)
	require.NoError(t, err)

	var prs []ProviderRewards
	ret = edb.Get().Find(&prs)
	require.NoError(t, ret.Error)
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
					ProviderID:   mn.ID,
					Rewards:      mn.Stat.GeneratorRewards,
					TotalRewards: mn.Stat.GeneratorRewards,
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

	if err := db.addMiner([]Miner{mnMiner1, mnMiner2}); err != nil {
		t.Error(err)
	}

	if err := db.rewardProviders(map[string]currency.Coin{
		mnMiner1.ID: 20,
		mnMiner2.ID: 30,
	}, map[string]currency.Coin{
		mnMiner1.ID: 50,
		mnMiner2.ID: 60,
	}, 7); err != nil {
		t.Error(err)
	}

	assertMinerRewards(t, db, mnMiner1.ID, uint64(20+5), uint64(50+5), 7)
	assertMinerRewards(t, db, mnMiner2.ID, uint64(30+5), uint64(60+5), 7)
}

func TestEventDb_rewardProviderDelegates(t *testing.T) {
	t.Skip("requires bug fix, https://github.com/0chain/0chain/pull/2059")
	edb, clean := GetTestEventDB(t)
	defer clean()

	err := edb.addDelegatePools([]DelegatePool{
		{
			PoolID:       "pool 1",
			ProviderID:   "miner one",
			ProviderType: spenum.Miner,
			Reward:       5,
			TotalReward:  23,
		},
		{
			PoolID:       "pool 2",
			ProviderID:   "miner two",
			ProviderType: spenum.Miner,
			Reward:       4,
			TotalReward:  0,
		},
		{
			PoolID:       "pool 1",
			ProviderID:   "miner two",
			ProviderType: spenum.Miner,
			Reward:       3,
			TotalReward:  0,
		},
	})
	require.NoError(t, err)

	err = edb.rewardProviderDelegates(
		map[string]map[string]currency.Coin{
			"miner two": {"pool 2": 11},
			"mienr two": {"pool 1": 17},
			"miner one": {"pool 1": 20},
		}, 7)
	require.NoError(t, err)

	var dps []DelegatePool
	ret := edb.Get().Find(&dps)
	require.NoError(t, ret.Error)
	fmt.Println("dps", dps)

	requireDelegateRewards(t, edb, "pool 1", "miner one", 20+5, 23+20, 7)
	requireDelegateRewards(t, edb, "pool 2", "miner two", 11+4, 11+0, 7)
	requireDelegateRewards(t, edb, "pool 1", "miner two", 17+3, 17+0, 7)
}

func TestEventDb_blobberSpecificRevenue(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	err := edb.Store.Get().Model(&Blobber{}).Omit(clause.Associations).Create([]Blobber{
		{
			Provider: Provider{
				ID:                "B000",
			},
			BaseURL: "https://blobber.zero",
			TotalStorageIncome: 0,
			TotalReadIncome:   0,
			TotalSlashedStake: 0,
		},
		{
			Provider: Provider{
				ID:                "B001",
			},
			BaseURL: "https://blobber.one",
			TotalStorageIncome: 0,
			TotalReadIncome:   0,
			TotalSlashedStake: 0,
		},
		{
			Provider: Provider{
				ID:                "B002",
			},
			BaseURL: "https://blobber.two",
			TotalStorageIncome: 0,
			TotalReadIncome:   0,
			TotalSlashedStake: 0,
		},
		{
			Provider: Provider{
				ID:                "B003",
			},
			BaseURL: "https://blobber.three",
			TotalStorageIncome: 0,
			TotalReadIncome:   0,
			TotalSlashedStake: 0,
		},
	}).Error
	require.NoError(t, err)
	
	spus := []dbs.StakePoolReward{
		{
			// Shouldn't affect anybody
			ProviderID: dbs.ProviderID{
				ID: "B000",
				Type: spenum.Blobber,
			},
			Reward: 10,
			RewardType: spenum.BlockRewardBlobber,
		},
		{
			// Storage income : blobber one
			ProviderID: dbs.ProviderID{
				ID: "B001",
				Type: spenum.Blobber,
			},
			Reward: 20,
			RewardType: spenum.ChallengePassReward,
		},
		{
			// Read income : blobber two
			ProviderID: dbs.ProviderID{
				ID: "B002",
				Type: spenum.Blobber,
			},
			Reward: 30,
			RewardType: spenum.FileDownloadReward,
		},
		{
			// Slashed stake : blobber three
			ProviderID: dbs.ProviderID{
				ID: "B003",
				Type: spenum.Blobber,
			},
			Reward: 40,
			RewardType: spenum.ChallengeSlashPenalty,
		},
	}

	var (
		blobbersBefore []Blobber
		blobbersAfter  []Blobber
	)

	err = edb.Store.Get().Model(&Blobber{}).Omit(clause.Associations).Order("id ASC").Find(&blobbersBefore).Error
	require.NoError(t, err)

	err = edb.blobberSpecificRevenue(spus)
	require.NoError(t, err)

	err = edb.Store.Get().Model(&Blobber{}).Omit(clause.Associations).Order("id ASC").Find(&blobbersAfter).Error
	require.NoError(t, err)

	assert.Equal(t, blobbersBefore[0].TotalStorageIncome, blobbersAfter[0].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[0].TotalReadIncome, blobbersAfter[0].TotalReadIncome)
	assert.Equal(t, blobbersBefore[0].TotalSlashedStake, blobbersAfter[0].TotalSlashedStake)

	assert.Equal(t, blobbersBefore[1].TotalStorageIncome + 20, blobbersAfter[1].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[1].TotalReadIncome, blobbersAfter[1].TotalReadIncome)
	assert.Equal(t, blobbersBefore[1].TotalSlashedStake, blobbersAfter[1].TotalSlashedStake)

	assert.Equal(t, blobbersBefore[2].TotalStorageIncome, blobbersAfter[2].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[2].TotalReadIncome + 30, blobbersAfter[2].TotalReadIncome)
	assert.Equal(t, blobbersBefore[2].TotalSlashedStake, blobbersAfter[2].TotalSlashedStake)

	assert.Equal(t, blobbersBefore[3].TotalStorageIncome, blobbersAfter[3].TotalStorageIncome)
	assert.Equal(t, blobbersBefore[3].TotalReadIncome, blobbersAfter[3].TotalReadIncome)
	assert.Equal(t, blobbersBefore[3].TotalSlashedStake + 40, blobbersAfter[3].TotalSlashedStake)
}

func requireDelegateRewards(
	t *testing.T,
	eventDb *EventDb,
	poolId, providerID string,
	reward, totalReward uint64,
	lastUpdated int64,
) {
	dp, err := eventDb.GetDelegatePool(poolId, providerID)
	require.NoError(t, err)
	require.EqualValues(t, reward, uint64(dp.Reward))
	require.EqualValues(t, totalReward, uint64(dp.TotalReward))
	require.EqualValues(t, lastUpdated, dp.RoundPoolLastUpdated)
}

func assertMinerRewards(t *testing.T, eventDb *EventDb, minerId string, reward, totalReward uint64, lastUpdated int64) {
	miner, err := eventDb.GetMiner(minerId)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, totalReward, uint64(miner.Rewards.TotalRewards))
	assert.Equal(t, reward, uint64(miner.Rewards.Rewards))
	assert.Equal(t, miner.Rewards.RoundServiceChargeLastUpdated, lastUpdated)
}
