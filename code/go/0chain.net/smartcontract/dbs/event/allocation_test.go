package event

import (
	"fmt"
	"testing"
	"time"

	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

const (
	KB      = 64 * 1024
	OwnerId = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"
)

// createMockAllocations - Creates "count" mock allocations and overwrites the first "len(presetAllocs)" of them with
// allocation entered optionally by the user.
func createMockAllocations(t *testing.T, edb *EventDb, count int, presetAllocs ...*Allocation) []string {
	var (
		ids    []string
		allocs []Allocation
	)
	i := 0
	for _, alloc := range presetAllocs {
		if alloc.AllocationID == "" {
			alloc.AllocationID = fmt.Sprintf("586925180648cfbc969561cbeeca2c0dbd9b68b29c5ccbd9e185bbb962e4a5d%v", i)
		}

		if alloc.TransactionID == "" {
			alloc.TransactionID = fmt.Sprintf("586925180648cfbc969561cbeeca2c0dbd9b68b29c5ccbd9e185bbb962e4a5d%v", i)
		}

		if alloc.Owner == "" {
			alloc.Owner = OwnerId
		}
		ids = append(ids, alloc.AllocationID)
		allocs = append(allocs, *alloc)
		i++
	}

	// Complete count with mock allocations
	for ; i < count; i++ {
		id := fmt.Sprintf("586925180648cfbc969561cbeeca2c0dbd9b68b29c5ccbd9e185bbb962e4a5d%v", i)
		allocs = append(allocs, Allocation{
			AllocationID:             id,
			TransactionID:            fmt.Sprintf("586925180648cfbc969561cbeeca2c0dbd9b68b29c5ccbd9e185bbb962e4a5d3%v", i),
			DataShards:               1,
			ParityShards:             1,
			FileOptions:              63,
			Size:                     100 * 1024 * 1024,   // 100 MB
			Expiration:               9223372036854775807, // Never expire
			Owner:                    OwnerId,
			OwnerPublicKey:           "owner_public_key",
			ReadPriceMin:             10,
			ReadPriceMax:             20,
			WritePriceMin:            10,
			WritePriceMax:            20,
			StartTime:                10212,
			Finalized:                true,
			Cancelled:                false,
			UsedSize:                 50,
			MovedToChallenge:         currency.Coin(10),
			MovedBack:                currency.Coin(1),
			MovedToValidators:        currency.Coin(1),
			WritePool:                currency.Coin(1),
			TimeUnit:                 12453,
			NumWrites:                5,
			NumReads:                 5,
			TotalChallenges:          12,
			OpenChallenges:           11,
			SuccessfulChallenges:     1,
			FailedChallenges:         0,
			LatestClosedChallengeTxn: "latest_closed_challenge_txn",
			ThirdPartyExtendable:     false,
		})
		ids = append(ids, id)
	}
	err := edb.addAllocations(allocs)
	assert.NoError(t, err, "inserting allocations failed")
	return ids
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
		MinStake currency.Coin `json:"min_stake"`
		// MaxStake allowed.
		MaxStake currency.Coin `json:"max_stake"`
		// NumDelegates maximum allowed.
		NumDelegates int `json:"num_delegates"`
		// ServiceCharge of the blobber. The blobber gets this % (actually, value in
		// [0; 1) range). If the ServiceCharge greater than max_charge of the SC
		// then the blobber can't be registered / updated.
		ServiceCharge float64 `json:"service_charge"`
	}

	type Terms struct {
		// ReadPrice is price for reading. Token / GB (no time unit).
		ReadPrice currency.Coin `json:"read_price"`
		// WritePrice is price for reading. Token / GB / time unit. Also,
		// it used to calculate min_lock_demand value.
		WritePrice currency.Coin `json:"write_price"`
		// MinLockDemand in number in [0; 1] range. It represents part of
		// allocation should be locked for the blobber rewards even if
		// user never write something to the blobber.
		MinLockDemand float64 `json:"min_lock_demand"`
		// MaxOfferDuration with this prices and the demand.
		MaxOfferDuration time.Duration `json:"max_offer_duration"`
	}

	type PriceRange struct {
		Min currency.Coin `json:"min"`
		Max currency.Coin `json:"max"`
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
		// StakePoolSettings used initially to create and setup stake pool.
		StakePoolSettings stakePoolSettings `json:"stake_pool_settings"`
	}

	type StorageAllocationStats struct {
		UsedSize                  int64  `json:"used_size"`
		NumWrites                 int64  `json:"num_of_writes"`
		ReadSize                  int64  `json:"read_size"`
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
		MinLockDemand currency.Coin `json:"min_lock_demand"`
		Spent         currency.Coin `json:"spent"`
		// Penalty o the blobber for the allocation in tokens.
		Penalty currency.Coin `json:"penalty"`
		// ReadReward of the blobber.
		ReadReward currency.Coin `json:"read_reward"`
		// Returned back to write pool on challenge failed.
		Returned currency.Coin `json:"returned"`
		// ChallengeReward of the blobber.
		ChallengeReward            currency.Coin `json:"challenge_reward"`
		FinalReward                currency.Coin `json:"final_reward"`
		ChallengePoolIntegralValue currency.Coin `json:"challenge_pool_integral_value"`
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

		// Requested ranges.
		ReadPriceRange  PriceRange `json:"read_price_range"`
		WritePriceRange PriceRange `json:"write_price_range"`

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
		MovedToChallenge currency.Coin `json:"moved_to_challenge,omitempty"`
		// MovedBack is number of tokens moved from challenge pool to
		// related write pool (the Back) if a data has deleted.
		MovedBack currency.Coin `json:"moved_back,omitempty"`
		// MovedToValidators is total number of tokens moved to validators
		// of the allocation.
		MovedToValidators currency.Coin `json:"moved_to_validators,omitempty"`

		// TimeUnit configured in Storage SC when the allocation created. It can't
		// be changed for this allocation anymore. Even using expire allocation.
		TimeUnit time.Duration `json:"time_unit"`

		Curators []string `json:"curators"`

		FileOptions uint16 `json:"file_options"`
	}

	convertSa := func(sa StorageAllocation) Allocation {
		var allocationTerms []AllocationBlobberTerm
		for _, b := range sa.BlobberDetails {
			allocationTerms = append(allocationTerms, AllocationBlobberTerm{
				BlobberID:        b.BlobberID,
				AllocationID:     b.AllocationID,
				ReadPrice:        int64(b.Terms.ReadPrice),
				WritePrice:       int64(b.Terms.WritePrice),
				MinLockDemand:    b.Terms.MinLockDemand,
				MaxOfferDuration: b.Terms.MaxOfferDuration,
			})
		}

		return Allocation{
			AllocationID:             sa.ID,
			TransactionID:            sa.Tx,
			DataShards:               sa.DataShards,
			ParityShards:             sa.ParityShards,
			Size:                     sa.Size,
			Expiration:               int64(sa.Expiration),
			Terms:                    allocationTerms,
			Owner:                    sa.Owner,
			OwnerPublicKey:           sa.OwnerPublicKey,
			ReadPriceMin:             sa.ReadPriceRange.Min,
			ReadPriceMax:             sa.ReadPriceRange.Max,
			WritePriceMin:            sa.WritePriceRange.Min,
			WritePriceMax:            sa.WritePriceRange.Max,
			StartTime:                int64(sa.StartTime),
			Finalized:                sa.Finalized,
			Cancelled:                sa.Canceled,
			UsedSize:                 sa.UsedSize,
			MovedToChallenge:         sa.MovedToChallenge,
			MovedBack:                sa.MovedBack,
			MovedToValidators:        sa.MovedToValidators,
			TimeUnit:                 int64(sa.TimeUnit),
			NumWrites:                sa.Stats.NumWrites,
			NumReads:                 sa.Stats.ReadSize / (64 * KB),
			TotalChallenges:          sa.Stats.TotalChallenges,
			OpenChallenges:           sa.Stats.OpenChallenges,
			SuccessfulChallenges:     sa.Stats.SuccessChallenges,
			FailedChallenges:         sa.Stats.FailedChallenges,
			LatestClosedChallengeTxn: sa.Stats.LastestClosedChallengeTxn,
			FileOptions:              sa.FileOptions,
		}
	}

	t.Run("test addAllocation", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create the owner
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: OwnerId,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

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
						ReadPrice:        10,
						WritePrice:       10,
						MinLockDemand:    2,
						MaxOfferDuration: 100,
					},
					Capacity:        100,
					Allocated:       50,
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
			Owner:          OwnerId,
			OwnerPublicKey: "owner_public_key",
			Stats: &StorageAllocationStats{
				UsedSize:                  20,
				NumWrites:                 5,
				ReadSize:                  5,
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
						ReadPrice:        10,
						WritePrice:       10,
						MinLockDemand:    2,
						MaxOfferDuration: 100,
					},
				},
			},
			BlobberMap: map[string]*BlobberAllocation{
				"hello": {
					BlobberID:    "blobber_1",
					AllocationID: "storage_allocation_id",
					Terms: Terms{
						ReadPrice:        10,
						WritePrice:       10,
						MinLockDemand:    2,
						MaxOfferDuration: 100,
					},
				},
			},
			FileOptions: 63,
			ReadPriceRange: PriceRange{
				Min: 10,
				Max: 20,
			},
			WritePriceRange: PriceRange{
				Min: 10,
				Max: 20,
			},
			ChallengeCompletionTime: 12,
			StartTime:               10212,
			Finalized:               true,
			Canceled:                false,
			UsedSize:                50,
			MovedToChallenge:        10,
			MovedBack:               1,
			MovedToValidators:       1,
			TimeUnit:                12453,
			Curators:                []string{"curator1"},
		}

		saAllocation := convertSa(sa)
		err = eventDb.addAllocations([]Allocation{saAllocation})
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		alloc, err := eventDb.GetAllocation(saAllocation.AllocationID)
		require.NoError(t, err)
		require.EqualValues(t, sa.DataShards, alloc.DataShards)

		allocs, err := eventDb.GetClientsAllocation(sa.Owner, common2.Pagination{Limit: 20, IsDescending: true})
		require.NoError(t, err)
		require.EqualValues(t, 1, len(allocs))
		require.EqualValues(t, allocs[0].Size, sa.Size)
	})

	t.Run("test edb.updateAllocation", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		otherOwner := "2746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		// Create the owners
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: OwnerId,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		err = eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: otherOwner,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		// Create the allocations
		allocIds := createMockAllocations(t, eventDb, 2)

		// Assert allocation entered successfuylly (1)
		alloc1, err := eventDb.GetAllocation(allocIds[0])
		require.NoError(t, err)

		alloc2, err := eventDb.GetAllocation(allocIds[1])
		require.NoError(t, err)

		// Update the allocations
		err = eventDb.updateAllocations([]Allocation{
			{
				AllocationID:             alloc1.AllocationID,
				DataShards:               2,
				ParityShards:             2,
				FileOptions:              60,
				Size:                     200 * 1024 * 1024, // 100 MB
				Expiration:               10000,             // Never expire
				Owner:                    otherOwner,
				OwnerPublicKey:           "owner_public_key2",
				ReadPriceMin:             currency.Coin(5),
				ReadPriceMax:             currency.Coin(40),
				WritePriceMin:            currency.Coin(5),
				WritePriceMax:            currency.Coin(40),
				StartTime:                10215,
				Finalized:                false,
				Cancelled:                true,
				UsedSize:                 500,
				MovedToChallenge:         currency.Coin(20),
				MovedBack:                currency.Coin(2),
				MovedToValidators:        currency.Coin(2),
				WritePool:                currency.Coin(2),
				TimeUnit:                 22453,
				NumWrites:                10,
				NumReads:                 10,
				TotalChallenges:          24,
				OpenChallenges:           20,
				SuccessfulChallenges:     2,
				FailedChallenges:         2,
				LatestClosedChallengeTxn: "latest_closed_challenge_txn_updated",
				ThirdPartyExtendable:     true,
			}, {
				AllocationID:             alloc2.AllocationID,
				DataShards:               2,
				ParityShards:             2,
				FileOptions:              60,
				Size:                     200 * 1024 * 1024, // 100 MB
				Expiration:               10000,             // Never expire
				Owner:                    otherOwner,
				OwnerPublicKey:           "owner_public_key2",
				ReadPriceMin:             5,
				ReadPriceMax:             40,
				WritePriceMin:            5,
				WritePriceMax:            40,
				StartTime:                10215,
				Finalized:                false,
				Cancelled:                true,
				UsedSize:                 500,
				MovedToChallenge:         currency.Coin(20),
				MovedBack:                currency.Coin(2),
				MovedToValidators:        currency.Coin(2),
				WritePool:                currency.Coin(2),
				TimeUnit:                 22453,
				NumWrites:                10,
				NumReads:                 10,
				TotalChallenges:          24,
				OpenChallenges:           20,
				SuccessfulChallenges:     2,
				FailedChallenges:         2,
				LatestClosedChallengeTxn: "latest_closed_challenge_txn_updated",
				ThirdPartyExtendable:     true,
			},
		})
		require.NoError(t, err, "update allocations failed")

		// Assert allocations updated successfuylly (1)
		alloc1, err = eventDb.GetAllocation(allocIds[0])
		require.NoError(t, err)

		alloc2, err = eventDb.GetAllocation(allocIds[1])
		require.NoError(t, err)

		// Check values updated successfully
		require.Equal(t, int(2), alloc1.DataShards)
		require.Equal(t, int(2), alloc1.ParityShards)
		require.Equal(t, uint16(60), alloc1.FileOptions)
		require.Equal(t, int64(200*1024*1024), alloc1.Size)
		require.Equal(t, int64(10000), alloc1.Expiration)
		require.Equal(t, otherOwner, alloc1.Owner)
		require.Equal(t, "owner_public_key2", alloc1.OwnerPublicKey)
		require.Equal(t, uint64(5), uint64(alloc1.ReadPriceMin))
		require.Equal(t, uint64(40), uint64(alloc1.ReadPriceMax))
		require.Equal(t, uint64(5), uint64(alloc1.WritePriceMin))
		require.Equal(t, uint64(40), uint64(alloc1.WritePriceMax))
		require.Equal(t, int64(10215), alloc1.StartTime)
		require.Equal(t, false, alloc1.Finalized)
		require.Equal(t, true, alloc1.Cancelled)
		require.Equal(t, int64(500), alloc1.UsedSize)
		require.Equal(t, uint64(20), uint64(alloc1.MovedToChallenge))
		require.Equal(t, uint64(2), uint64(alloc1.MovedBack))
		require.Equal(t, uint64(2), uint64(alloc1.MovedToValidators))
		require.Equal(t, uint64(2), uint64(alloc1.WritePool))
		require.Equal(t, int64(22453), alloc1.TimeUnit)
		require.Equal(t, int64(10), alloc1.NumWrites)
		require.Equal(t, int64(10), alloc1.NumReads)
		require.Equal(t, int64(24), alloc1.TotalChallenges)
		require.Equal(t, int64(20), alloc1.OpenChallenges)
		require.Equal(t, int64(2), alloc1.SuccessfulChallenges)
		require.Equal(t, int64(2), alloc1.FailedChallenges)
		require.Equal(t, "latest_closed_challenge_txn_updated", alloc1.LatestClosedChallengeTxn)
		require.Equal(t, true, alloc1.ThirdPartyExtendable)

		require.Equal(t, int(2), alloc2.DataShards)
		require.Equal(t, int(2), alloc2.ParityShards)
		require.Equal(t, uint16(60), alloc2.FileOptions)
		require.Equal(t, int64(200*1024*1024), alloc2.Size)
		require.Equal(t, int64(10000), alloc2.Expiration)
		require.Equal(t, otherOwner, alloc2.Owner)
		require.Equal(t, "owner_public_key2", alloc2.OwnerPublicKey)
		require.Equal(t, uint64(5), uint64(alloc2.ReadPriceMin))
		require.Equal(t, uint64(40), uint64(alloc2.ReadPriceMax))
		require.Equal(t, uint64(5), uint64(alloc2.WritePriceMin))
		require.Equal(t, uint64(40), uint64(alloc2.WritePriceMax))
		require.Equal(t, int64(10215), alloc2.StartTime)
		require.Equal(t, false, alloc2.Finalized)
		require.Equal(t, true, alloc2.Cancelled)
		require.Equal(t, int64(500), alloc2.UsedSize)
		require.Equal(t, uint64(20), uint64(alloc2.MovedToChallenge))
		require.Equal(t, uint64(2), uint64(alloc2.MovedBack))
		require.Equal(t, uint64(2), uint64(alloc2.MovedToValidators))
		require.Equal(t, uint64(2), uint64(alloc2.WritePool))
		require.Equal(t, int64(22453), alloc2.TimeUnit)
		require.Equal(t, int64(10), alloc2.NumWrites)
		require.Equal(t, int64(10), alloc2.NumReads)
		require.Equal(t, int64(24), alloc2.TotalChallenges)
		require.Equal(t, int64(20), alloc2.OpenChallenges)
		require.Equal(t, int64(2), alloc2.SuccessfulChallenges)
		require.Equal(t, int64(2), alloc2.FailedChallenges)
		require.Equal(t, "latest_closed_challenge_txn_updated", alloc2.LatestClosedChallengeTxn)
		require.Equal(t, true, alloc2.ThirdPartyExtendable)
	})

	t.Run("test edb.updateAllocationStake", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create the owner
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: OwnerId,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		// Create 2 allocations
		allocIds := createMockAllocations(t, eventDb, 2,
			&Allocation{
				WritePool:         currency.Coin(10),
				MovedToChallenge:  currency.Coin(10),
				MovedBack:         currency.Coin(10),
				MovedToValidators: currency.Coin(10),
			},
			&Allocation{
				WritePool:         currency.Coin(20),
				MovedToChallenge:  currency.Coin(20),
				MovedBack:         currency.Coin(20),
				MovedToValidators: currency.Coin(20),
			},
		)

		aid1, aid2 := allocIds[0], allocIds[1]

		// Assert allocation entered successfuylly (1)
		alloc, err := eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid1))
		require.Equal(t, alloc.WritePool, currency.Coin(10))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(10))
		require.Equal(t, alloc.MovedBack, currency.Coin(10))
		require.Equal(t, alloc.MovedToValidators, currency.Coin(10))

		// Assert allocation entered successfuylly (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid2))
		require.Equal(t, alloc.WritePool, currency.Coin(20))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(20))
		require.Equal(t, alloc.MovedBack, currency.Coin(20))
		require.Equal(t, alloc.MovedToValidators, currency.Coin(20))

		// Update the 2 allocations doubling the amounts
		err = eventDb.updateAllocationStakes([]Allocation{
			{
				AllocationID:      aid1,
				WritePool:         currency.Coin(20),
				MovedToChallenge:  currency.Coin(20),
				MovedBack:         currency.Coin(20),
				MovedToValidators: currency.Coin(20),
			},
			{
				AllocationID:      aid2,
				WritePool:         currency.Coin(40),
				MovedToChallenge:  currency.Coin(40),
				MovedBack:         currency.Coin(40),
				MovedToValidators: currency.Coin(40),
			},
		})

		require.NoError(t, err, "couldn't update allocation stakes")

		// Test update was successful (1)
		alloc, err = eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid1))
		require.Equal(t, alloc.WritePool, currency.Coin(20))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(20))
		require.Equal(t, alloc.MovedBack, currency.Coin(20))
		require.Equal(t, alloc.MovedToValidators, currency.Coin(20))

		// Test update was successful (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid2))
		require.Equal(t, alloc.WritePool, currency.Coin(40))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(40))
		require.Equal(t, alloc.MovedBack, currency.Coin(40))
		require.Equal(t, alloc.MovedToValidators, currency.Coin(40))
	})

	t.Run("test edb.updateAllocationsStats", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create the owner
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: OwnerId,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		allocIds := createMockAllocations(t, eventDb, 2,
			&Allocation{
				UsedSize:         10000,
				NumWrites:        10,
				MovedToChallenge: currency.Coin(100),
				MovedBack:        currency.Coin(200),
				WritePool:        currency.Coin(100),
			},
			&Allocation{
				UsedSize:         20000,
				NumWrites:        20,
				MovedToChallenge: currency.Coin(200),
				MovedBack:        currency.Coin(400),
				WritePool:        currency.Coin(200),
			},
		)

		aid1, aid2 := allocIds[0], allocIds[1]

		// Assert allocation entered successfuylly (1)
		alloc, err := eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid1))
		require.Equal(t, alloc.UsedSize, int64(10000))
		require.Equal(t, alloc.NumWrites, int64(10))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(100))
		require.Equal(t, alloc.MovedBack, currency.Coin(200))
		require.Equal(t, alloc.WritePool, currency.Coin(100))

		// Assert allocation entered successfuylly (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid2))
		require.Equal(t, alloc.UsedSize, int64(20000))
		require.Equal(t, alloc.NumWrites, int64(20))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(200))
		require.Equal(t, alloc.MovedBack, currency.Coin(400))
		require.Equal(t, alloc.WritePool, currency.Coin(200))

		// Update the 2 allocations doubling the amounts
		err = eventDb.updateAllocationsStats([]Allocation{
			{
				AllocationID:     aid1,
				UsedSize:         10000,
				NumWrites:        10,
				MovedToChallenge: currency.Coin(100),
				MovedBack:        currency.Coin(200),
			},
			{
				AllocationID:     aid2,
				UsedSize:         20000,
				NumWrites:        20,
				MovedToChallenge: currency.Coin(200),
				MovedBack:        currency.Coin(400),
			},
		})

		require.NoError(t, err, "couldn't update allocation stats")

		// Test update was successful (1)
		alloc, err = eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid1))
		require.Equal(t, alloc.UsedSize, int64(20000))
		require.Equal(t, alloc.NumWrites, int64(20))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(200))
		require.Equal(t, alloc.MovedBack, currency.Coin(400))
		require.Equal(t, alloc.WritePool, currency.Coin(200))

		// Test update was successful (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid2))
		require.Equal(t, alloc.UsedSize, int64(40000))
		require.Equal(t, alloc.NumWrites, int64(40))
		require.Equal(t, alloc.MovedToChallenge, currency.Coin(400))
		require.Equal(t, alloc.MovedBack, currency.Coin(800))
		require.Equal(t, alloc.WritePool, currency.Coin(400))
	})

	t.Run("test edb.updateAllocationChallenges", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create the owner
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: OwnerId,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		allocIds := createMockAllocations(t, eventDb, 2,
			&Allocation{
				OpenChallenges:           20,
				LatestClosedChallengeTxn: "1001",
				SuccessfulChallenges:     10,
				FailedChallenges:         10,
			},
			&Allocation{
				OpenChallenges:           40,
				LatestClosedChallengeTxn: "2001",
				SuccessfulChallenges:     20,
				FailedChallenges:         20,
			},
		)

		aid1, aid2 := allocIds[0], allocIds[1]

		// Assert allocation entered successfuylly (1)
		alloc, err := eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid1))
		require.Equal(t, alloc.OpenChallenges, int64(20))
		require.Equal(t, alloc.LatestClosedChallengeTxn, "1001")
		require.Equal(t, alloc.SuccessfulChallenges, int64(10))
		require.Equal(t, alloc.FailedChallenges, int64(10))

		// Assert allocation entered successfuylly (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid2))
		require.Equal(t, alloc.OpenChallenges, int64(40))
		require.Equal(t, alloc.LatestClosedChallengeTxn, "2001")
		require.Equal(t, alloc.SuccessfulChallenges, int64(20))
		require.Equal(t, alloc.FailedChallenges, int64(20))

		// Update the 2 allocations changing the amounts
		err = eventDb.updateAllocationChallenges([]Allocation{
			{
				AllocationID:             aid1,
				OpenChallenges:           10,
				LatestClosedChallengeTxn: "1002",
				SuccessfulChallenges:     5,
				FailedChallenges:         5,
			},
			{
				AllocationID:             aid2,
				OpenChallenges:           20,
				LatestClosedChallengeTxn: "2002",
				SuccessfulChallenges:     10,
				FailedChallenges:         10,
			},
		})

		require.NoError(t, err, "couldn't update allocation stats")

		// Test update was successful (1)
		alloc, err = eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid1))
		require.Equal(t, alloc.OpenChallenges, int64(10))
		require.Equal(t, alloc.LatestClosedChallengeTxn, "1002")
		require.Equal(t, alloc.SuccessfulChallenges, int64(15))
		require.Equal(t, alloc.FailedChallenges, int64(15))

		// Test update was successful (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid2))
		require.Equal(t, alloc.OpenChallenges, int64(20))
		require.Equal(t, alloc.LatestClosedChallengeTxn, "2002")
		require.Equal(t, alloc.SuccessfulChallenges, int64(30))
		require.Equal(t, alloc.FailedChallenges, int64(30))
	})

	t.Run("test edb.addChallengesToAllocations", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create the owner
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserMetrics: UserMetrics{
				UserID: OwnerId,
			},
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		allocIds := createMockAllocations(t, eventDb, 2,
			&Allocation{
				TotalChallenges: 20,
				OpenChallenges:  10,
			},
			&Allocation{
				TotalChallenges: 40,
				OpenChallenges:  20,
			},
		)

		aid1, aid2 := allocIds[0], allocIds[1]

		// Assert allocation entered successfuylly (1)
		alloc, err := eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid1))
		require.Equal(t, alloc.TotalChallenges, int64(20))
		require.Equal(t, alloc.OpenChallenges, int64(10))

		// Assert allocation entered successfuylly (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not inserted", aid2))
		require.Equal(t, alloc.TotalChallenges, int64(40))
		require.Equal(t, alloc.OpenChallenges, int64(20))

		// Update the 2 allocations doubling the amounts
		err = eventDb.addChallengesToAllocations([]Allocation{
			{
				AllocationID:    aid1,
				TotalChallenges: 20,
				OpenChallenges:  10,
			},
			{
				AllocationID:    aid2,
				TotalChallenges: 40,
				OpenChallenges:  20,
			},
		})

		require.NoError(t, err, "couldn't update allocation stats")

		// Test update was successful (1)
		alloc, err = eventDb.GetAllocation(aid1)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid1))
		require.Equal(t, alloc.TotalChallenges, int64(40))
		require.Equal(t, alloc.OpenChallenges, int64(20))

		// Test update was successful (2)
		alloc, err = eventDb.GetAllocation(aid2)
		require.NoError(t, err, fmt.Sprintf("allocation %v not found after update", aid2))
		require.Equal(t, alloc.TotalChallenges, int64(80))
		require.Equal(t, alloc.OpenChallenges, int64(40))
	})
}
