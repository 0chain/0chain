package event

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/currency"

	"golang.org/x/net/context"

	"go.uber.org/zap"

	"github.com/0chain/common/core/logging"

	"0chain.net/core/common"

	"github.com/stretchr/testify/require"

	"0chain.net/smartcontract/dbs"
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
		ReadPrice        currency.Coin `json:"read_price"`
		WritePrice       currency.Coin `json:"write_price"`
		MinLockDemand    float64       `json:"min_lock_demand"`
		MaxOfferDuration time.Duration `json:"max_offer_duration"`
	}
	type stakePoolSettings struct {
		DelegateWallet string        `json:"delegate_wallet"`
		MinStake       currency.Coin `json:"min_stake"`
		MaxStake       currency.Coin `json:"max_stake"`
		NumDelegates   int           `json:"num_delegates"`
		ServiceCharge  float64       `json:"service_charge"`
	}
	type StorageNode struct {
		ID              string                 `json:"id"`
		BaseURL         string                 `json:"url"`
		Geolocation     StorageNodeGeolocation `json:"geolocation"`
		Terms           Terms                  `json:"terms"`     // terms
		Capacity        int64                  `json:"capacity"`  // total blobber capacity
		Allocated       int64                  `json:"allocated"` // allocated capacity
		LastHealthCheck common.Timestamp       `json:"last_health_check"`
		PublicKey       string                 `json:"-"`
		SavedData       int64                  `json:"saved_data"`
		// StakePoolSettings used initially to create and setup stake pool.
		StakePoolSettings stakePoolSettings `json:"stake_pool_settings"`
	}
	convertSn := func(sn StorageNode) Blobber {
		return Blobber{
			BlobberID:        sn.ID,
			BaseURL:          sn.BaseURL,
			Latitude:         sn.Geolocation.Latitude,
			Longitude:        sn.Geolocation.Longitude,
			ReadPrice:        sn.Terms.ReadPrice,
			WritePrice:       sn.Terms.WritePrice,
			MinLockDemand:    sn.Terms.MinLockDemand,
			MaxOfferDuration: sn.Terms.MaxOfferDuration.Nanoseconds(),
			Capacity:         sn.Capacity,
			Allocated:        sn.Allocated,
			LastHealthCheck:  int64(sn.LastHealthCheck),
			Provider: &Provider{
				DelegateWallet: sn.StakePoolSettings.DelegateWallet,
				MinStake:       sn.StakePoolSettings.MaxStake,
				MaxStake:       sn.StakePoolSettings.MaxStake,
				NumDelegates:   sn.StakePoolSettings.NumDelegates,
				ServiceCharge:  sn.StakePoolSettings.ServiceCharge,
			},
			SavedData: sn.SavedData,
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
	eventDb, err := NewEventDb(access)
	require.NoError(t, err)
	defer eventDb.Close()
	err = eventDb.Drop()
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
			ReadPrice:        currency.Coin(29),
			WritePrice:       currency.Coin(31),
			MinLockDemand:    37.0,
			MaxOfferDuration: 39 * time.Minute,
		},
		Capacity:        43,
		Allocated:       47,
		LastHealthCheck: common.Timestamp(51),
		PublicKey:       "public key",
		StakePoolSettings: stakePoolSettings{
			DelegateWallet: "delegate wallet",
			MinStake:       currency.Coin(53),
			MaxStake:       currency.Coin(57),
			NumDelegates:   59,
			ServiceCharge:  61.0,
		},
		SavedData: 10,
	}
	SnBlobber := convertSn(sn)
	data, err := json.Marshal(&SnBlobber)
	require.NoError(t, err)

	eventAddSn := Event{
		BlockNumber: 2,
		TxHash:      "tx hash",
		Type:        TypeStats,
		Tag:         TagAddBlobber,
		Data:        string(data),
	}
	events := []Event{eventAddSn}
	eventDb.ProcessEvents(context.TODO(), events, 100, "hash", 10)

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
		Type:        TypeStats,
		Tag:         TagUpdateBlobber,
		Data:        string(data),
	}
	eventDb.ProcessEvents(context.TODO(), []Event{eventUpdateSn}, 100, "hash", 10)

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
			ReadPrice:        currency.Coin(97),
			WritePrice:       currency.Coin(101),
			MinLockDemand:    103.0,
			MaxOfferDuration: 107 * time.Minute,
		},
		Capacity:        119,
		Allocated:       127,
		LastHealthCheck: common.Timestamp(131),
		PublicKey:       "public key",
		StakePoolSettings: stakePoolSettings{
			DelegateWallet: "delegate wallet",
			MinStake:       currency.Coin(137),
			MaxStake:       currency.Coin(139),
			NumDelegates:   143,
			ServiceCharge:  149.0,
		},
		SavedData: 10,
	}
	SnBlobber2 := convertSn(sn2)
	data, err = json.Marshal(&SnBlobber2)
	require.NoError(t, err)
	eventOverwrite := Event{
		BlockNumber: 2,
		TxHash:      "tx hash3",
		Type:        TypeStats,
		Tag:         TagUpdateBlobber,
		Data:        string(data),
	}
	eventDb.ProcessEvents(context.TODO(), []Event{eventOverwrite}, 100, "hash", 10)
	overWrittenBlobber, err := eventDb.GetBlobber(sn.ID)
	require.NoError(t, err)
	require.EqualValues(t, sn2.BaseURL, overWrittenBlobber.BaseURL)

	deleteEvent := Event{
		BlockNumber: 3,
		TxHash:      "tx hash4",
		Type:        TypeStats,
		Tag:         TagDeleteBlobber,
		Data:        blobber.BlobberID,
	}
	eventDb.ProcessEvents(context.TODO(), []Event{deleteEvent}, 100, "hash", 10)

	blobber, err = eventDb.GetBlobber(sn.ID)
	require.Error(t, err)
}

func TestBlobberIds(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		return
	}
	defer eventDb.Close()

	err = eventDb.AutoMigrate()
	require.NoError(t, err)
	defer eventDb.Drop()

	setUpBlobbers(t, eventDb)

	blobberIDs, err := eventDb.GetAllBlobberId()
	require.NoError(t, err)
	require.Equal(t, 10, len(blobberIDs), "All blobber id's were not found")

}

func TestBlobberLatLong(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		t.Skip("only for local debugging, requires local postgresql")
		return
	}
	defer eventDb.Close()

	err = eventDb.AutoMigrate()
	require.NoError(t, err)
	defer eventDb.Drop()

	setUpBlobbers(t, eventDb)
}

func TestBlobberGetCount(t *testing.T) {
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
	eventDb, err := NewEventDb(access)
	if err != nil {
		return
	}
	defer eventDb.Close()

	err = eventDb.AutoMigrate()
	require.NoError(t, err)
	defer eventDb.Drop()

	gotCount, err := eventDb.GetBlobberCount()
	require.NoError(t, err, "Error should not be present")
	require.Equal(t, int64(0), gotCount, "Blobber count not working")

	setUpBlobbers(t, eventDb)

	gotCount, err = eventDb.GetBlobberCount()
	require.NoError(t, err, "Error should not be present")
	require.Equal(t, int64(10), gotCount, "Blobber Count should be 10")
}

func setUpBlobbers(t *testing.T, eventDb *EventDb) {
	for i := 0; i < 10; i++ {
		res := eventDb.Store.Get().Create(&Blobber{
			BlobberID: fmt.Sprintf("somethingNew_%v", i),
		})
		if res.Error != nil {
			t.Errorf("Error while inserting blobber %v", i)
			t.FailNow()
			return
		}
	}
}
