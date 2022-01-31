package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"strings"
	"testing"
	"time"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestAllocations(t *testing.T) {

	type StorageNodeGeolocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		// reserved / Accuracy float64 `mapstructure:"accuracy"`
	}

	type stakePoolSettings struct {
		// DelegateWallet for pool owner.
		DelegateWallet string `json:"delegate_wallet"`
		// MinStake allowed.
		MinStake state.Balance `json:"min_stake"`
		// MaxStake allowed.
		MaxStake state.Balance `json:"max_stake"`
		// NumDelegates maximum allowed.
		NumDelegates int `json:"num_delegates"`
		// ServiceCharge of the blobber. The blobber gets this % (actually, value in
		// [0; 1) range). If the ServiceCharge greater than max_charge of the SC
		// then the blobber can't be registered / updated.
		ServiceCharge float64 `json:"service_charge"`
	}

	type Terms struct {
		// ReadPrice is price for reading. Token / GB (no time unit).
		ReadPrice state.Balance `json:"read_price"`
		// WritePrice is price for reading. Token / GB / time unit. Also,
		// it used to calculate min_lock_demand value.
		WritePrice state.Balance `json:"write_price"`
		// MinLockDemand in number in [0; 1] range. It represents part of
		// allocation should be locked for the blobber rewards even if
		// user never write something to the blobber.
		MinLockDemand float64 `json:"min_lock_demand"`
		// MaxOfferDuration with this prices and the demand.
		MaxOfferDuration time.Duration `json:"max_offer_duration"`
		// ChallengeCompletionTime is duration required to complete a challenge.
		ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
	}

	type PriceRange struct {
		Min state.Balance `json:"min"`
		Max state.Balance `json:"max"`
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

	type StorageAllocationStats struct {
		UsedSize                  int64  `json:"used_size"`
		NumWrites                 int64  `json:"num_of_writes"`
		NumReads                  int64  `json:"num_of_reads"`
		TotalChallenges           int64  `json:"total_challenges"`
		OpenChallenges            int64  `json:"num_open_challenges"`
		SuccessChallenges         int64  `json:"num_success_challenges"`
		FailedChallenges          int64  `json:"num_failed_challenges"`
		LastestClosedChallengeTxn string `json:"latest_closed_challenge"`
	}

	type WriteMarker struct {
		AllocationRoot         string           `json:"allocation_root"`
		PreviousAllocationRoot string           `json:"prev_allocation_root"`
		AllocationID           string           `json:"allocation_id"`
		Size                   int64            `json:"size"`
		BlobberID              string           `json:"blobber_id"`
		Timestamp              common.Timestamp `json:"timestamp"`
		ClientID               string           `json:"client_id"`
		Signature              string           `json:"signature"`
	}

	type BlobberAllocation struct {
		BlobberID       string                  `json:"blobber_id"`
		AllocationID    string                  `json:"allocation_id"`
		Size            int64                   `json:"size"`
		AllocationRoot  string                  `json:"allocation_root"`
		LastWriteMarker *WriteMarker            `json:"write_marker"`
		Stats           *StorageAllocationStats `json:"stats"`
		Terms           Terms                   `json:"terms"`
		// MinLockDemand for the allocation in tokens.
		MinLockDemand state.Balance `json:"min_lock_demand"`
		Spent         state.Balance `json:"spent"`
		// Penalty o the blobber for the allocation in tokens.
		Penalty state.Balance `json:"penalty"`
		// ReadReward of the blobber.
		ReadReward state.Balance `json:"read_reward"`
		// Returned back to write pool on challenge failed.
		Returned state.Balance `json:"returned"`
		// ChallengeReward of the blobber.
		ChallengeReward            state.Balance `json:"challenge_reward"`
		FinalReward                state.Balance `json:"final_reward"`
		ChallengePoolIntegralValue state.Balance `json:"challenge_pool_integral_value"`
	}

	type StorageAllocation struct {
		// ID is unique allocation ID that is equal to hash of transaction with
		// which the allocation has created.
		ID string `json:"id"`
		// Tx keeps hash with which the allocation has created or updated.
		Tx string `json:"tx"`

		DataShards        int                           `json:"data_shards"`
		ParityShards      int                           `json:"parity_shards"`
		Size              int64                         `json:"size"`
		Expiration        common.Timestamp              `json:"expiration_date"`
		Blobbers          []*StorageNode                `json:"blobbers"`
		Owner             string                        `json:"owner_id"`
		OwnerPublicKey    string                        `json:"owner_public_key"`
		Stats             *StorageAllocationStats       `json:"stats"`
		DiverseBlobbers   bool                          `json:"diverse_blobbers"`
		PreferredBlobbers []string                      `json:"preferred_blobbers"`
		BlobberDetails    []*BlobberAllocation          `json:"blobber_details"`
		BlobberMap        map[string]*BlobberAllocation `json:"-"`
		IsImmutable       bool                          `json:"is_immutable"`

		// Requested ranges.
		ReadPriceRange             PriceRange    `json:"read_price_range"`
		WritePriceRange            PriceRange    `json:"write_price_range"`
		MaxChallengeCompletionTime time.Duration `json:"max_challenge_completion_time"`

		//AllocationPools allocationPools `json:"allocation_pools"`
		WritePoolOwners []string `json:"write_pool_owners"`

		// ChallengeCompletionTime is max challenge completion time of
		// all blobbers of the allocation.
		ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
		// StartTime is time when the allocation has been created. We will
		// use it to check blobber's MaxOfferTime extending the allocation.
		StartTime common.Timestamp `json:"start_time"`
		// Finalized is true where allocation has been finalized.
		Finalized bool `json:"finalized,omitempty"`
		// Canceled set to true where allocation finalized by cancel_allocation
		// transaction.
		Canceled bool `json:"canceled,omitempty"`
		// UsedSize used to calculate blobber reward ratio.
		UsedSize int64 `json:"-"`

		// MovedToChallenge is number of tokens moved to challenge pool.
		MovedToChallenge state.Balance `json:"moved_to_challenge,omitempty"`
		// MovedBack is number of tokens moved from challenge pool to
		// related write pool (the Back) if a data has deleted.
		MovedBack state.Balance `json:"moved_back,omitempty"`
		// MovedToValidators is total number of tokens moved to validators
		// of the allocation.
		MovedToValidators state.Balance `json:"moved_to_validators,omitempty"`

		// TimeUnit configured in Storage SC when the allocation created. It can't
		// be changed for this allocation anymore. Even using expire allocation.
		TimeUnit time.Duration `json:"time_unit"`

		Curators []string `json:"curators"`
	}

	convertSa := func(sa StorageAllocation) Allocation {
		var allocationTerms []AllocationTerm
		for _, b := range sa.BlobberDetails {
			allocationTerms = append(allocationTerms, AllocationTerm{
				BlobberID:               b.BlobberID,
				AllocationID:            b.AllocationID,
				ReadPrice:               b.Terms.ReadPrice,
				WritePrice:              b.Terms.WritePrice,
				MinLockDemand:           b.Terms.MinLockDemand,
				MaxOfferDuration:        b.Terms.MaxOfferDuration,
				ChallengeCompletionTime: b.Terms.ChallengeCompletionTime,
			})
		}
		termsByte, err := json.Marshal(allocationTerms)
		require.NoError(t, err)

		return Allocation{
			AllocationID:               sa.ID,
			TransactionID:              sa.Tx,
			DataShards:                 sa.DataShards,
			ParityShards:               sa.ParityShards,
			Size:                       sa.Size,
			Expiration:                 int64(sa.Expiration),
			Terms:                      string(termsByte),
			Owner:                      sa.Owner,
			OwnerPublicKey:             sa.OwnerPublicKey,
			IsImmutable:                sa.IsImmutable,
			ReadPriceMin:               sa.ReadPriceRange.Min,
			ReadPriceMax:               sa.ReadPriceRange.Max,
			WritePriceMin:              sa.WritePriceRange.Min,
			WritePriceMax:              sa.WritePriceRange.Max,
			MaxChallengeCompletionTime: int64(sa.MaxChallengeCompletionTime),
			ChallengeCompletionTime:    int64(sa.ChallengeCompletionTime),
			StartTime:                  int64(sa.StartTime),
			Finalized:                  sa.Finalized,
			Cancelled:                  sa.Canceled,
			UsedSize:                   sa.UsedSize,
			MovedToChallenge:           sa.MovedToChallenge,
			MovedBack:                  sa.MovedBack,
			MovedToValidators:          sa.MovedToValidators,
			Curators:                   strings.Join(sa.Curators, ","),
			TimeUnit:                   int64(sa.TimeUnit),
			NumWrites:                  sa.Stats.NumWrites,
			NumReads:                   sa.Stats.NumReads,
			TotalChallenges:            sa.Stats.TotalChallenges,
			OpenChallenges:             sa.Stats.OpenChallenges,
			SuccessfulChallenges:       sa.Stats.SuccessChallenges,
			FailedChallenges:           sa.Stats.FailedChallenges,
			LatestClosedChallengeTxn:   sa.Stats.LastestClosedChallengeTxn,
		}
	}

	access := dbs.DbAccess{
		Enabled:         true,
		Name:            "events_db",
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

	err = eventDb.drop()
	require.NoError(t, err)
	err = eventDb.AutoMigrate()
	require.NoError(t, err)

	sa := StorageAllocation{
		ID:           "storage_allocation_id",
		Tx:           "txn1",
		DataShards:   10,
		ParityShards: 6,
		Size:         100,
		Expiration:   1512,
		Blobbers: []*StorageNode{
			{
				ID:      "blobber_1",
				BaseURL: "base_url",
				Geolocation: StorageNodeGeolocation{
					Latitude:  120,
					Longitude: 141,
				},
				Terms: Terms{
					ReadPrice:               10,
					WritePrice:              10,
					MinLockDemand:           2,
					MaxOfferDuration:        100,
					ChallengeCompletionTime: 21,
				},
				Capacity:        100,
				Used:            50,
				LastHealthCheck: 17456,
				PublicKey:       "public_key",
				StakePoolSettings: stakePoolSettings{
					DelegateWallet: "delegate_wallet",
					MinStake:       10,
					MaxStake:       12,
					NumDelegates:   2,
					ServiceCharge:  0.5,
				},
			},
		},
		Owner:          "owner_id",
		OwnerPublicKey: "owner_public_key",
		Stats: &StorageAllocationStats{
			UsedSize:                  20,
			NumWrites:                 5,
			NumReads:                  5,
			TotalChallenges:           12,
			OpenChallenges:            11,
			SuccessChallenges:         1,
			FailedChallenges:          0,
			LastestClosedChallengeTxn: "latest_closed_challenge_txn",
		},
		BlobberDetails: []*BlobberAllocation{
			{
				BlobberID:    "blobber_1",
				AllocationID: "storage_allocation_id",
				Terms: Terms{
					ReadPrice:               10,
					WritePrice:              10,
					MinLockDemand:           2,
					MaxOfferDuration:        100,
					ChallengeCompletionTime: 21,
				},
			},
		},
		BlobberMap: map[string]*BlobberAllocation{
			"hello": {
				BlobberID:    "blobber_1",
				AllocationID: "storage_allocation_id",
				Terms: Terms{
					ReadPrice:               10,
					WritePrice:              10,
					MinLockDemand:           2,
					MaxOfferDuration:        100,
					ChallengeCompletionTime: 21,
				},
			},
		},
		IsImmutable: false,
		ReadPriceRange: PriceRange{
			Min: 10,
			Max: 20,
		},
		WritePriceRange: PriceRange{
			Min: 10,
			Max: 20,
		},
		MaxChallengeCompletionTime: 20,
		ChallengeCompletionTime:    12,
		StartTime:                  10212,
		Finalized:                  true,
		Canceled:                   false,
		UsedSize:                   50,
		MovedToChallenge:           10,
		MovedBack:                  1,
		MovedToValidators:          1,
		TimeUnit:                   12453,
		Curators:                   []string{"curator1"},
	}

	saAllocation := convertSa(sa)
	data, err := json.Marshal(&saAllocation)
	require.NoError(t, err)

	eventAddSa := Event{
		Model:       gorm.Model{},
		BlockNumber: 1,
		TxHash:      "txn_hash",
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteAllocation),
		Index:       saAllocation.AllocationID,
		Data:        string(data),
	}
	eventDb.AddEvents(context.TODO(), []Event{eventAddSa})

	alloc, err := eventDb.GetAllocation(saAllocation.AllocationID)
	require.NoError(t, err)
	require.EqualValues(t, alloc.DataShards, sa.DataShards)

	sa.Size = 271
	saAllocation = convertSa(sa)
	data, err = json.Marshal(&saAllocation)
	require.NoError(t, err)

	eventOverwriteSa := Event{
		Model:       gorm.Model{},
		BlockNumber: 2,
		TxHash:      "txn_hash2",
		Type:        int(TypeStats),
		Tag:         int(TagAddOrOverwriteAllocation),
		Index:       saAllocation.AllocationID,
		Data:        string(data),
	}
	eventDb.AddEvents(context.TODO(), []Event{eventOverwriteSa})

	alloc, err = eventDb.GetAllocation(saAllocation.AllocationID)
	require.NoError(t, err)
	require.EqualValues(t, alloc.Size, sa.Size)

	allocs, err := eventDb.GetClientsAllocation(sa.Owner)
	require.NoError(t, err)
	require.EqualValues(t, 1, len(allocs))
	require.EqualValues(t, allocs[0].Size, sa.Size)

}
