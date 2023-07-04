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
	StorageAllocation `json:",inline"`
	Blobbers          []*storageNodeResponse `json:"blobbers"`
}

func allocationTableToStorageAllocationBlobbers(alloc *event.Allocation, eventDb *event.EventDb) (*StorageAllocationBlobbers, error) {
	storageNodes := make([]*storageNodeResponse, 0)
	blobberDetails := make([]*BlobberAllocation, 0)
	blobberIDs := make([]string, 0)
	blobberTermsMap := make(map[string]Terms)
	blobberMap := make(map[string]*BlobberAllocation)

	for _, t := range alloc.Terms {
		blobberIDs = append(blobberIDs, t.BlobberID)
		blobberTermsMap[t.BlobberID] = Terms{
			ReadPrice:     currency.Coin(t.ReadPrice),
			WritePrice:    currency.Coin(t.WritePrice),
			MinLockDemand: t.MinLockDemand,
		}
	}

	blobbers, err := eventDb.GetBlobbersFromIDs(blobberIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobbers from db: %v", err)
	}

	var dpsSze = alloc.DataShards + alloc.ParityShards
	var gbSize = sizeInGB((alloc.Size + int64(dpsSze-1)) / int64(dpsSze))
	var rdtu = float64(time.Second*time.Duration(alloc.Expiration-alloc.StartTime)) / float64(alloc.TimeUnit)
	bs := make([]*AllocBlobber, 0, len(blobbers))
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

		bwF := gbSize * terms.MinLockDemand * rdtu
		minLockDemand, err := currency.MultFloat64(terms.WritePrice, bwF)
		if err != nil {
			return nil, err
		}
		bs = append(bs, &AllocBlobber{
			BlobberID:     b.ID,
			Terms:         terms,
			MinLockDemand: minLockDemand,
		})

		ba := &BlobberAllocation{
			BlobberID:    b.ID,
			AllocationID: alloc.AllocationID,
		}
		blobberDetails = append(blobberDetails, ba)
		blobberMap[b.ID] = ba
	}

	sa := &StorageAllocation{
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
		Blobbers:          bs,
		BlobberAllocs:     blobberDetails,
		BlobberAllocsMap:  blobberMap,
		ReadPriceRange:    PriceRange{alloc.ReadPriceMin, alloc.ReadPriceMax},
		WritePriceRange:   PriceRange{alloc.WritePriceMin, alloc.WritePriceMax},
		StartTime:         common.Timestamp(alloc.StartTime),
		Finalized:         alloc.Finalized,
		Canceled:          alloc.Cancelled,
		UsedSize:          alloc.UsedSize,
		MovedToChallenge:  alloc.MovedToChallenge,
		MovedBack:         alloc.MovedBack,
		MovedToValidators: alloc.MovedToValidators,
		TimeUnit:          time.Duration(alloc.TimeUnit),
	}

	return &StorageAllocationBlobbers{
		StorageAllocation: *sa,
		Blobbers:          storageNodes,
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
			AllocationID:  sa.ID,
			BlobberID:     b.BlobberID,
			ReadPrice:     int64(sa.bTerms(i).ReadPrice),
			WritePrice:    int64(sa.bTerms(i).WritePrice),
			MinLockDemand: sa.bTerms(i).MinLockDemand,
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
