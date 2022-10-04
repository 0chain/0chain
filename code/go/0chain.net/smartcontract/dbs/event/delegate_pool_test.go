package event

import (
	"encoding/json"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
)

func TestDelegatePoolsEvent(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
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
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	dp := DelegatePool{
		PoolID:       "pool_id",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",

		Balance: 29,
	}

	err = eventDb.addOrOverwriteDelegatePools(dp)
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	dps, err := eventDb.GetDelegatePools("provider_id", int(spenum.Blobber))
	require.NoError(t, err)
	require.Len(t, dps, 1)

	err = eventDb.Drop()
	require.NoError(t, err)
}

func TestTagStakePoolReward(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")
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
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	bl := Blobber{
		BlobberID:          "provider_id",
		Reward:             11,
		TotalServiceCharge: 23,
	}
	res := eventDb.Store.Get().Create(&bl)
	if res.Error != nil {
		t.Errorf("Error while inserting blobber %v", bl)
		return
	}

	dp := DelegatePool{
		PoolID:       "pool_id_1",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",
		TotalReward:  29,
		Balance:      3,
	}

	dp2 := DelegatePool{
		PoolID:       "pool_id_2",
		ProviderType: 2,
		ProviderID:   "provider_id",
		DelegateID:   "delegate_id",

		Balance: 5,
	}

	err = eventDb.addOrOverwriteDelegatePools(dp)
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")
	err = eventDb.addOrOverwriteDelegatePools(dp2)
	require.NoError(t, err, "Error while inserting DelegatePool to event Database")

	var before int64
	eventDb.Get().Table("delegate_pools").Count(&before)

	var spr dbs.StakePoolReward
	spr.ProviderId = "provider_id"
	spr.ProviderType = int(spenum.Blobber)
	spr.Reward = 17
	spr.DelegateRewards = map[string]int64{
		"pool_id_1": 15,
		"pool_id_2": 2,
	}
	data, err := json.Marshal(&spr)
	eventDb.addStat(Event{
		Type:  int(TypeStats),
		Tag:   int(TagStakePoolReward),
		Index: spr.ProviderId,
		Data:  string(data),
	})

	var after int64
	eventDb.Get().Table("delegate_pools").Count(&after)

	dps, err := eventDb.GetDelegatePools("provider_id", int(spenum.Blobber))
	require.NoError(t, err)
	require.Len(t, dps, 2)
}
