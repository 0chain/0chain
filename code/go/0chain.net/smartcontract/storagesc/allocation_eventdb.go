package storagesc

import (
	"fmt"
	"time"

	"0chain.net/chaincore/transaction"

	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
)

type StorageAllocationBlobbers struct {
	StorageAllocationResponse `json:",inline"`
	Blobbers                  []*storageNodeResponse `json:"blobbers"`
}

type StorageAllocationResponse struct {
	FileOptions  uint16 `json:"file_options" msg:"fo"`
	DataShards   int    `json:"data_shards" msg:"d"`
	ParityShards int    `json:"parity_shards" msg:"p"`
	Size         int64  `json:"size" msg:"s"`
	BSize        int64  `json:"bsize" msg:"bz"`

	// Requested ranges.
	ReadPriceRange  PriceRange `json:"read_price_range" msg:"rp"`
	WritePriceRange PriceRange `json:"write_price_range" msg:"wp"`

	//DiverseBlobbers bool `json:"diverse_blobbers" msg:"db"`
	// Flag to determine if anyone can extend this allocation
	ThirdPartyExtendable bool `json:"third_party_extendable" msg:"tpe"`
	// Finalized is true where allocation has been finalized.
	Finalized bool `json:"finalized,omitempty" msg:"f"`
	// Canceled set to true where allocation finalized by cancel_allocation
	// transaction.
	Canceled bool `json:"canceled,omitempty" msg:"c"`

	WritePool     currency.Coin `json:"write_pool" msg:"w"`
	ChallengePool currency.Coin `json:"challenge_pool" msg:"cp"`

	// MinLockDemand in number in [0; 1] range. It represents part of
	// allocation should be locked for the blobber rewards even if
	// user never write something to the blobber.
	MinLockDemand float64 `json:"min_lock_demand"`

	// MovedToChallenge is number of tokens moved to challenge pool.
	MovedToChallenge currency.Coin `json:"moved_to_challenge,omitempty" msg:"mtc"`
	// MovedBack is number of tokens moved from challenge pool to
	// related write pool (the Back) if a data has deleted.
	MovedBack currency.Coin `json:"moved_back,omitempty" msg:"mb"`
	// MovedToValidators is total number of tokens moved to validators
	// of the allocation.
	MovedToValidators currency.Coin `json:"moved_to_validators,omitempty" msg:"mv"`
	CancelCost        currency.Coin `json:"cancel_cost" msg:"cc"`

	Expiration common.Timestamp `json:"expiration_date" msg:"ep"`
	// StartTime is time when the allocation has been created. We will
	// use it to check blobber's MaxOfferTime extending the allocation.
	StartTime common.Timestamp `json:"start_time" msg:"st"`
	TimeUnit  time.Duration    `json:"time_unit" msg:"tu"`
	ID        string           `json:"id" msg:"i"`
	// Tx keeps hash with which the allocation has created or updated. todo do we need this field?
	Tx string `json:"tx" msg:"t"`

	Owner          string `json:"owner_id" msg:"o"`
	OwnerPublicKey string `json:"owner_public_key" msg:"op"`

	Stats *StorageAllocationStats `json:"stats"`

	BlobberAllocs []*BlobberAllocationResponse `json:"blobber_details"`
}

type BlobberAllocationResponse struct {
	BlobberID      string        `json:"blobber_id"`
	AllocationID   string        `json:"allocation_id"`
	AllocationRoot string        `json:"allocation_root"`
	Size           int64         `json:"size"`
	MinLockDemand  currency.Coin `json:"min_lock_demand"`
	Terms          Terms         `json:"terms"`
}

func allocationTableToStorageAllocationBlobbers(alloc *event.Allocation, eventDb *event.EventDb) (*StorageAllocationBlobbers, error) {
	storageNodes := make([]*storageNodeResponse, 0)
	blobberDetails := make([]*BlobberAllocationResponse, 0)
	blobberIDs := make([]string, 0)
	blobberTermsMap := make(map[string]Terms)

	for _, t := range alloc.Terms {
		blobberIDs = append(blobberIDs, t.BlobberID)
		blobberTermsMap[t.BlobberID] = Terms{
			ReadPrice:  currency.Coin(t.ReadPrice),
			WritePrice: currency.Coin(t.WritePrice),
		}
	}

	blobbers, err := eventDb.GetBlobbersFromIDs(blobberIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobbers from db: %v", err)
	}

	var dpsSze = alloc.DataShards + alloc.ParityShards
	var gbSize = sizeInGB((alloc.Size + int64(dpsSze-1)) / int64(dpsSze))
	var rdtu = float64(time.Second*time.Duration(alloc.Expiration-alloc.StartTime)) / float64(alloc.TimeUnit)
	for _, b := range blobbers {
		storageNodes = append(storageNodes, &storageNodeResponse{
			ID:      b.ID,
			BaseURL: b.BaseURL,
			Geolocation: StorageNodeGeolocation{
				Latitude:  b.Latitude,
				Longitude: b.Longitude,
			},
			Terms:           blobberTermsMap[b.ID],
			Capacity:        b.Capacity,
			Allocated:       b.Allocated,
			SavedData:       b.SavedData,
			LastHealthCheck: b.LastHealthCheck,
			StakePoolSettings: stakepool.Settings{
				DelegateWallet:     b.DelegateWallet,
				MaxNumDelegates:    b.NumDelegates,
				ServiceChargeRatio: b.ServiceCharge,
			},
		})

		terms := blobberTermsMap[b.ID]

		bwF := gbSize * alloc.MinLockDemand * rdtu
		minLockDemand, err := currency.MultFloat64(terms.WritePrice, bwF)
		if err != nil {
			return nil, err
		}

		ba := &BlobberAllocationResponse{
			BlobberID:     b.ID,
			AllocationID:  alloc.AllocationID,
			Size:          b.Allocated,
			Terms:         terms,
			MinLockDemand: minLockDemand,
		}
		blobberDetails = append(blobberDetails, ba)
	}

	sa := &StorageAllocationResponse{
		ID:                   alloc.AllocationID,
		Tx:                   alloc.TransactionID,
		DataShards:           alloc.DataShards,
		ParityShards:         alloc.ParityShards,
		Size:                 alloc.Size,
		Expiration:           common.Timestamp(alloc.Expiration),
		Owner:                alloc.Owner,
		OwnerPublicKey:       alloc.OwnerPublicKey,
		WritePool:            alloc.WritePool,
		ThirdPartyExtendable: alloc.ThirdPartyExtendable,
		FileOptions:          alloc.FileOptions,
		Stats: &StorageAllocationStats{
			UsedSize:                  alloc.UsedSize,
			NumWrites:                 alloc.NumWrites,
			NumReads:                  alloc.NumReads,
			TotalChallenges:           alloc.TotalChallenges,
			OpenChallenges:            alloc.OpenChallenges,
			SuccessChallenges:         alloc.SuccessfulChallenges,
			FailedChallenges:          alloc.FailedChallenges,
			LastestClosedChallengeTxn: alloc.LatestClosedChallengeTxn,
		},
		BlobberAllocs:     blobberDetails,
		ReadPriceRange:    PriceRange{alloc.ReadPriceMin, alloc.ReadPriceMax},
		WritePriceRange:   PriceRange{alloc.WritePriceMin, alloc.WritePriceMax},
		StartTime:         common.Timestamp(alloc.StartTime),
		Finalized:         alloc.Finalized,
		Canceled:          alloc.Cancelled,
		MovedToChallenge:  alloc.MovedToChallenge,
		MovedBack:         alloc.MovedBack,
		MovedToValidators: alloc.MovedToValidators,
		TimeUnit:          time.Duration(alloc.TimeUnit),
	}

	return &StorageAllocationBlobbers{
		StorageAllocationResponse: *sa,
		Blobbers:                  storageNodes,
	}, nil
}

func storageAllocationToAllocationTable(sa *StorageAllocation) *event.Allocation {
	alloc := &event.Allocation{
		AllocationID:         sa.ID,
		TransactionID:        sa.Tx,
		DataShards:           sa.DataShards,
		ParityShards:         sa.ParityShards,
		Size:                 sa.Size,
		Expiration:           int64(sa.Expiration),
		Terms:                sa.buildEventBlobberTerms(),
		Owner:                sa.Owner,
		OwnerPublicKey:       sa.OwnerPublicKey,
		ReadPriceMin:         sa.ReadPriceRange.Min,
		ReadPriceMax:         sa.ReadPriceRange.Max,
		WritePriceMin:        sa.WritePriceRange.Min,
		WritePriceMax:        sa.WritePriceRange.Max,
		StartTime:            int64(sa.StartTime),
		Finalized:            sa.Finalized,
		Cancelled:            sa.Canceled,
		UsedSize:             sa.UsedSize,
		MovedToChallenge:     sa.MovedToChallenge,
		MovedBack:            sa.MovedBack,
		MovedToValidators:    sa.MovedToValidators,
		TimeUnit:             int64(sa.TimeUnit),
		WritePool:            sa.WritePool,
		ThirdPartyExtendable: sa.ThirdPartyExtendable,
		FileOptions:          sa.FileOptions,
	}

	if sa.Stats != nil {
		alloc.NumWrites = sa.Stats.NumWrites
		alloc.NumReads = sa.Stats.NumReads
		alloc.TotalChallenges = sa.Stats.TotalChallenges
		alloc.OpenChallenges = sa.Stats.OpenChallenges
		alloc.SuccessfulChallenges = sa.Stats.SuccessChallenges
		alloc.FailedChallenges = sa.Stats.FailedChallenges
		alloc.LatestClosedChallengeTxn = sa.Stats.LastestClosedChallengeTxn
	}

	return alloc
}

func (sa *StorageAllocation) buildEventBlobberTerms() []event.AllocationBlobberTerm {
	bTerms := make([]event.AllocationBlobberTerm, 0, len(sa.BlobberAllocs))
	for i, b := range sa.Blobbers {
		bTerms = append(bTerms, event.AllocationBlobberTerm{
			AllocationID: sa.ID,
			BlobberID:    b.BlobberID,
			ReadPrice:    int64(sa.bTerms(i).ReadPrice),
			WritePrice:   int64(sa.bTerms(i).WritePrice),
		})
	}

	return bTerms
}

func (sa *StorageAllocation) buildDbUpdates() event.Allocation {
	eAlloc := event.Allocation{
		AllocationID:         sa.ID,
		TransactionID:        sa.Tx,
		DataShards:           sa.DataShards,
		ParityShards:         sa.ParityShards,
		Size:                 sa.Size,
		Expiration:           int64(sa.Expiration),
		Owner:                sa.Owner,
		OwnerPublicKey:       sa.OwnerPublicKey,
		ReadPriceMin:         sa.ReadPriceRange.Min,
		ReadPriceMax:         sa.ReadPriceRange.Max,
		WritePriceMin:        sa.WritePriceRange.Min,
		WritePriceMax:        sa.WritePriceRange.Max,
		StartTime:            int64(sa.StartTime),
		Finalized:            sa.Finalized,
		Cancelled:            sa.Canceled,
		UsedSize:             sa.UsedSize,
		MovedToChallenge:     sa.MovedToChallenge,
		MovedBack:            sa.MovedBack,
		MovedToValidators:    sa.MovedToValidators,
		TimeUnit:             int64(sa.TimeUnit),
		WritePool:            sa.WritePool,
		ThirdPartyExtendable: sa.ThirdPartyExtendable,
		FileOptions:          sa.FileOptions,
	}

	if sa.Stats != nil {
		eAlloc.NumWrites = sa.Stats.NumWrites
		eAlloc.NumReads = sa.Stats.NumReads
		eAlloc.TotalChallenges = sa.Stats.TotalChallenges
		eAlloc.OpenChallenges = sa.Stats.OpenChallenges
		eAlloc.SuccessfulChallenges = sa.Stats.SuccessChallenges
		eAlloc.FailedChallenges = sa.Stats.FailedChallenges
		eAlloc.LatestClosedChallengeTxn = sa.Stats.LastestClosedChallengeTxn
	}

	return eAlloc
}

func (sa *StorageAllocation) buildStakeUpdateEvent() event.Allocation {
	return event.Allocation{
		AllocationID:      sa.ID,
		WritePool:         sa.WritePool,
		MovedToChallenge:  sa.MovedToChallenge,
		MovedBack:         sa.MovedBack,
		MovedToValidators: sa.MovedToValidators,
	}
}

func (sa *StorageAllocation) emitAdd(balances cstate.StateContextI) error {
	alloc := storageAllocationToAllocationTable(sa)
	balances.EmitEvent(event.TypeStats, event.TagAddAllocation, alloc.AllocationID, alloc)

	return nil
}

func getClientAllocationsFromDb(clientID string, eventDb *event.EventDb, limit common2.Pagination) ([]*StorageAllocationBlobbers, error) {

	sas := make([]*StorageAllocationBlobbers, 0)

	allocs, err := eventDb.GetClientsAllocation(clientID, limit)
	if err != nil {
		return nil, err
	}

	for _, alloc := range allocs {
		sa, err := allocationTableToStorageAllocationBlobbers(&alloc, eventDb)
		if err != nil {
			return nil, err
		}

		sas = append(sas, sa)
	}

	return sas, nil
}

func prepareAllocationsResponse(eventDb *event.EventDb, eAllocs []event.Allocation) ([]*StorageAllocationBlobbers, error) {
	sas := make([]*StorageAllocationBlobbers, 0, len(eAllocs))
	for _, eAlloc := range eAllocs {
		sa, err := allocationTableToStorageAllocationBlobbers(&eAlloc, eventDb)
		if err != nil {
			return nil, err
		}

		sas = append(sas, sa)
	}

	return sas, nil
}

func emitAddOrOverwriteAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteAllocationBlobberTerm, t.Hash, sa.buildEventBlobberTerms())
}

func emitUpdateAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationBlobberTerm, sa.ID, sa.buildEventBlobberTerms())
}

func emitDeleteAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagDeleteAllocationBlobberTerm, t.Hash, sa.buildEventBlobberTerms())
}
