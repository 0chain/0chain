package event

import (
	"encoding/json"
	"golang.org/x/net/context"
	"testing"
	"time"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestBlobbers(t *testing.T) {
	t.Skip("only for local debugging, requires local postgresql")

	type StorageNodeGeolocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	type Terms struct {
		ReadPrice               state.Balance `json:"read_price"`
		WritePrice              state.Balance `json:"write_price"`
		MinLockDemand           float64       `json:"min_lock_demand"`
		MaxOfferDuration        time.Duration `json:"max_offer_duration"`
		ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
	}
	type stakePoolSettings struct {
		DelegateWallet string        `json:"delegate_wallet"`
		MinStake       state.Balance `json:"min_stake"`
		MaxStake       state.Balance `json:"max_stake"`
		NumDelegates   int           `json:"num_delegates"`
		ServiceCharge  float64       `json:"service_charge"`
	}
	type StorageNode struct {
		ID              string                 `json:"id"`
		BaseURL         string                 `json:"url"`
		Geolocation     StorageNodeGeolocation `json:"geolocation"`
		Terms           Terms                  `json:"terms"`    // terms
		Capacity        int64                  `json:"capacity"` // total blobber capacity
		Used            int64                  `json:"used"`     // allocated capacity
		LastHealthCheck common.Timestamp       `json:"last_health_check"`
		PublicKey       string                 `json:"-"`
		// StakePoolSettings used initially to create and setup stake pool.
		StakePoolSettings stakePoolSettings `json:"stake_pool_settings"`
	}
	convertSn := func(sn StorageNode) Blobber {
		return Blobber{
			BlobberID:               sn.ID,
			BaseURL:                 sn.BaseURL,
			Latitude:                sn.Geolocation.Latitude,
			Longitude:               sn.Geolocation.Longitude,
			ReadPrice:               int64(sn.Terms.ReadPrice),
			WritePrice:              int64(sn.Terms.WritePrice),
			MinLockDemand:           sn.Terms.MinLockDemand,
			MaxOfferDuration:        sn.Terms.MaxOfferDuration.String(),
			ChallengeCompletionTime: sn.Terms.ChallengeCompletionTime.String(),
			Capacity:                sn.Capacity,
			Used:                    sn.Used,
			LastHealthCheck:         int64(sn.LastHealthCheck),
			DelegateWallet:          sn.StakePoolSettings.DelegateWallet,
			MinStake:                int64(sn.StakePoolSettings.MaxStake),
			MaxStake:                int64(sn.StakePoolSettings.MaxStake),
			NumDelegates:            sn.StakePoolSettings.NumDelegates,
			ServiceCharge:           sn.StakePoolSettings.ServiceCharge,
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

	sn := StorageNode{
		ID:      "blobber one",
		BaseURL: "base url",
		Geolocation: StorageNodeGeolocation{
			Longitude: 17,
			Latitude:  23,
		},
		Terms: Terms{
			ReadPrice:               state.Balance(29),
			WritePrice:              state.Balance(31),
			MinLockDemand:           37.0,
			MaxOfferDuration:        39 * time.Minute,
			ChallengeCompletionTime: 41 * time.Minute,
		},
		Capacity:        43,
		Used:            47,
		LastHealthCheck: common.Timestamp(51),
		PublicKey:       "public key",
		StakePoolSettings: stakePoolSettings{
			DelegateWallet: "delegate wallet",
			MinStake:       state.Balance(53),
			MaxStake:       state.Balance(57),
			NumDelegates:   59,
			ServiceCharge:  61.0,
		},
	}
	SnBlobber := convertSn(sn)
	data, err := json.Marshal(&SnBlobber)
	require.NoError(t, err)

	require.NoError(t, err)
	eventAddSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash",
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteBlobber),
		Data:        string(data),
	}
	events := []Event{eventAddSn}
	eventDb.AddEvents(context.TODO(), events)

	blobber, err := eventDb.GetBlobber(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, blobber.BaseURL, sn.BaseURL)

	update := dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"base_url":   "new base url",
			"latitude":   67.0,
			"longitude":  83.0,
			"read_price": 87,
			"capacity":   89,
		},
	}
	data, err = json.Marshal(&update)
	require.NoError(t, err)

	eventUpdateSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash2",
		Type:        int(TypeStats),
		Tag:         int(TagUpdateBlobber),
		Data:        string(data),
	}
	eventDb.AddEvents(context.TODO(), []Event{eventUpdateSn})

	blobber, err = eventDb.GetBlobber(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, blobber.BaseURL, update.Updates["base_url"])

	sn2 := StorageNode{
		ID:      "blobber one",
		BaseURL: "another base url",
		Geolocation: StorageNodeGeolocation{
			Longitude: 91,
			Latitude:  93,
		},
		Terms: Terms{
			ReadPrice:               state.Balance(97),
			WritePrice:              state.Balance(101),
			MinLockDemand:           103.0,
			MaxOfferDuration:        107 * time.Minute,
			ChallengeCompletionTime: 113 * time.Minute,
		},
		Capacity:        119,
		Used:            127,
		LastHealthCheck: common.Timestamp(131),
		PublicKey:       "public key",
		StakePoolSettings: stakePoolSettings{
			DelegateWallet: "delegate wallet",
			MinStake:       state.Balance(137),
			MaxStake:       state.Balance(139),
			NumDelegates:   143,
			ServiceCharge:  149.0,
		},
	}
	SnBlobber2 := convertSn(sn2)
	data, err = json.Marshal(&SnBlobber2)
	require.NoError(t, err)
	eventOverwrite := Event{
		BlockNumber: 2,
		TxHash:      "tx hash3",
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteBlobber),
		Data:        string(data),
	}
	eventDb.AddEvents(context.TODO(), []Event{eventOverwrite})
	overWrittenBlobber, err := eventDb.GetBlobber(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, sn2.BaseURL, overWrittenBlobber.BaseURL)

	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        int(TypeStats),
		Tag:         int(TagDeleteBlobber),
		Data:        blobber.BlobberID,
	}
	eventDb.AddEvents(context.TODO(), []Event{deleteEvent})

	blobber, err = eventDb.GetBlobber(sn.ID)
	require.Error(t, err)
}
