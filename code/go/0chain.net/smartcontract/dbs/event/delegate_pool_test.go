package event

import (
	"testing"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/stretchr/testify/require"
)

func TestDelegatePoolsEvent(t *testing.T) {
	t.Parallel()
	db, clean := GetTestEventDB(t)
	defer clean()

	dp := DelegatePool{
		PoolID:       "pool_id",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",

		Balance: 29,
	}

	err := db.addDelegatePools([]DelegatePool{dp})
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	dps, err := db.GetDelegatePools("provider_id")
	require.NoError(t, err)
	require.Len(t, dps, 1)
	require.Equal(t, dps[0].PoolID, dp.PoolID)
	require.Equal(t, dps[0].Balance, dp.Balance)
}

// Same as TestDelegatePoolsEvent but with different balance to see if running tests parallel
// on updating the same table item would affect the result.
func TestDelegatePoolsEvent2(t *testing.T) {
	t.Parallel()
	db, clean := GetTestEventDB(t)
	defer clean()

	dp := DelegatePool{
		PoolID:       "pool_id",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",

		Balance: 30,
	}

	err := db.addDelegatePools([]DelegatePool{dp})
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	dps, err := db.GetDelegatePools("provider_id")
	require.NoError(t, err)
	require.Len(t, dps, 1)
	require.Equal(t, dps[0].PoolID, dp.PoolID)
	require.Equal(t, dps[0].Balance, dp.Balance)
}

// Same as TestDelegatePoolsEvent2
func TestDelegatePoolsEvent3(t *testing.T) {
	t.Parallel()
	db, clean := GetTestEventDB(t)
	defer clean()
	dp := DelegatePool{
		PoolID:       "pool_id",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",

		Balance: 40,
	}

	err := db.addDelegatePools([]DelegatePool{dp})
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	dps, err := db.GetDelegatePools("provider_id")
	require.NoError(t, err)
	require.Len(t, dps, 1)
	require.Equal(t, dps[0].PoolID, dp.PoolID)
	require.Equal(t, dps[0].Balance, dp.Balance)
}

func TestTagStakePoolReward(t *testing.T) {
	t.Parallel()
	db, clean := GetTestEventDB(t)
	defer clean()

	bl := Blobber{
		Provider: Provider{ID: "provider_id",
			Rewards: ProviderRewards{
				ProviderID:   "provider_id",
				Rewards:      11,
				TotalRewards: 23,
			}},
	}
	res := db.Store.Get().Create(&bl)
	if res.Error != nil {
		t.Errorf("Error while inserting blobber %v", bl)
		return
	}

	dp := DelegatePool{
		PoolID:       "pool_id_1",
		ProviderType: spenum.Blobber,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",
		TotalReward:  29,
		Balance:      3,
	}

	dp2 := DelegatePool{
		PoolID:       "pool_id_2",
		ProviderType: spenum.Blobber,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",
		Balance:      5,
	}

	err := db.addDelegatePools([]DelegatePool{dp})
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")
	err = db.addDelegatePools([]DelegatePool{dp2})
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	var before int64
	db.Get().Table("delegate_pools").Count(&before)
	var spr dbs.StakePoolReward
	spr.ProviderId = "provider_id"
	spr.ProviderType = spenum.Blobber
	spr.Reward = 17
	spr.DelegateRewards = map[string]currency.Coin{
		"pool_id_1": 15,
		"pool_id_2": 2,
	}
	require.NoError(t, db.addStat(Event{
		Type:  TypeStats,
		Tag:   TagStakePoolReward,
		Index: spr.ProviderId,
		Data:  []dbs.StakePoolReward{spr},
	}))

	var after int64
	db.Get().Table("delegate_pools").Count(&after)
	dps, err := db.GetDelegatePools("provider_id")
	require.NoError(t, err)
	require.Len(t, dps, 2)
}
