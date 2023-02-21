package event

import (
	"testing"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/stretchr/testify/require"

	"0chain.net/smartcontract/dbs"
)

func TestInsertDelegateReward(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	const round = 17
	minerIds := createMiners(t, edb, 2)
	err := edb.addDelegatePools([]DelegatePool{
		{
			PoolID:       "pool_id",
			ProviderType: spenum.Miner,
			ProviderID:   minerIds[1],
			DelegateID:   "delegate_id",

			Balance: 30,
		},
		{
			PoolID:       "pool_id_2",
			ProviderType: spenum.Miner,
			ProviderID:   minerIds[1],
			DelegateID:   "delegate_id_2",

			Balance: 30,
		},
	})
	require.NoError(t, err)
	s, dps, err := edb.GetMinerWithDelegatePools(minerIds[1])
	s = s
	dps = dps

	spr1 := dbs.StakePoolReward{
		Provider: dbs.Provider{
			ProviderId:   minerIds[0],
			ProviderType: spenum.Miner,
		},
		Reward:     123,
		RewardType: spenum.BlockRewardBlobber,
		DelegateRewards: map[string]currency.Coin{
			"pool_id":   700,
			"pool_id_2": 1100,
		},
	}
	spr2 := dbs.StakePoolReward{
		Provider: dbs.Provider{
			ProviderId:   minerIds[1],
			ProviderType: spenum.Miner,
		},
		Reward:     234,
		RewardType: spenum.BlockRewardBlobber,
		DelegateRewards: map[string]currency.Coin{
			"pool_id": 1900,
		},
	}

	inserts := []dbs.StakePoolReward{
		spr1, spr2,
	}

	err = edb.insertDelegateReward(inserts, round)
	require.NoError(t, err)

	rewards, err := edb.GetDelegateRewards(common.Pagination{
		Offset:       0,
		Limit:        10,
		IsDescending: true,
	}, "", round-1, round+1)
	require.NoError(t, err)
	require.Len(t, rewards, 3)

}
