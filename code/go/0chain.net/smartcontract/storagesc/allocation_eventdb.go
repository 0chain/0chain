package storagesc

import (
	"fmt"
	"time"

	"0chain.net/smartcontract/provider"

	"0chain.net/chaincore/transaction"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
)

type StorageAllocationBlobbers struct {
	StorageAllocation `json:",inline"`
	Blobbers          []*StorageNode `json:"blobbers"`
}

func allocationTableToStorageAllocationBlobbers(alloc *event.Allocation, eventDb *event.EventDb) (*StorageAllocationBlobbers, error) {
	storageNodes := make([]*StorageNode, 0)
	blobberDetails := make([]*BlobberAllocation, 0)
	blobberIDs := make([]string, 0)
	blobberTermsMap := make(map[string]Terms)
	blobberMap := make(map[string]*BlobberAllocation)

	curators, err := eventDb.GetCuratorsByAllocationID(alloc.AllocationID)
	if err != nil {
		return nil, fmt.Errorf("error finding curators: %v", err)
	}

	for _, t := range alloc.Terms {
		blobberIDs = append(blobberIDs, t.BlobberID)
		blobberTermsMap[t.BlobberID] = Terms{
			ReadPrice:        currency.Coin(t.ReadPrice),
			WritePrice:       currency.Coin(t.WritePrice),
			MinLockDemand:    t.MinLockDemand,
			MaxOfferDuration: t.MaxOfferDuration,
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
		storageNodes = append(storageNodes, &StorageNode{
			Provider: &provider.Provider{
				ID:              b.ID,
				LastHealthCheck: common.Timestamp(b.LastHealthCheck),
			},
			BaseURL:      b.BaseURL,
			ProviderType: spenum.Blobber,
			Geolocation: StorageNodeGeolocation{
				Latitude:  b.Latitude,
				Longitude: b.Longitude,
			},
			Terms:     blobberTermsMap[b.ID],
			Capacity:  b.Capacity,
			Allocated: b.Allocated,
			SavedData: b.SavedData,
			StakePoolSettings: stakepool.Settings{
				DelegateWallet:     b.DelegateWallet,
				MinStake:           currency.Coin(b.MinStake),
				MaxStake:           currency.Coin(b.MaxStake),
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

		ba := &BlobberAllocation{
			BlobberID:     b.ID,
			AllocationID:  alloc.AllocationID,
			Size:          b.Allocated,
			Terms:         terms,
			MinLockDemand: minLockDemand,
		}
		blobberDetails = append(blobberDetails, ba)
		blobberMap[b.ID] = ba
	}

	sa := &StorageAllocation{
		ID:             alloc.AllocationID,
		Tx:             alloc.TransactionID,
		Name:           alloc.AllocationName,
		DataShards:     alloc.DataShards,
		ParityShards:   alloc.ParityShards,
		Size:           alloc.Size,
		Expiration:     common.Timestamp(alloc.Expiration),
		Owner:          alloc.Owner,
		OwnerPublicKey: alloc.OwnerPublicKey,
		WritePool:      alloc.WritePool,
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
		BlobberAllocsMap:  blobberMap,
		IsImmutable:       alloc.IsImmutable,
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
		Curators:          curators,
	}

	return &StorageAllocationBlobbers{
		StorageAllocation: *sa,
		Blobbers:          storageNodes,
	}, nil
}

func storageAllocationToAllocationTable(sa *StorageAllocation) *event.Allocation {
	alloc := &event.Allocation{
		AllocationID:      sa.ID,
		AllocationName:    sa.Name,
		TransactionID:     sa.Tx,
		DataShards:        sa.DataShards,
		ParityShards:      sa.ParityShards,
		Size:              sa.Size,
		Expiration:        int64(sa.Expiration),
		Terms:             sa.buildEventBlobberTerms(),
		Owner:             sa.Owner,
		OwnerPublicKey:    sa.OwnerPublicKey,
		IsImmutable:       sa.IsImmutable,
		ReadPriceMin:      sa.ReadPriceRange.Min,
		ReadPriceMax:      sa.ReadPriceRange.Max,
		WritePriceMin:     sa.WritePriceRange.Min,
		WritePriceMax:     sa.WritePriceRange.Max,
		StartTime:         int64(sa.StartTime),
		Finalized:         sa.Finalized,
		Cancelled:         sa.Canceled,
		UsedSize:          sa.UsedSize,
		MovedToChallenge:  sa.MovedToChallenge,
		MovedBack:         sa.MovedBack,
		MovedToValidators: sa.MovedToValidators,
		TimeUnit:          int64(sa.TimeUnit),
		WritePool:         sa.WritePool,
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
	for _, b := range sa.BlobberAllocs {
		bTerms = append(bTerms, event.AllocationBlobberTerm{
			AllocationID:     sa.ID,
			BlobberID:        b.BlobberID,
			ReadPrice:        int64(b.Terms.ReadPrice),
			WritePrice:       int64(b.Terms.WritePrice),
			MinLockDemand:    b.Terms.MinLockDemand,
			MaxOfferDuration: b.Terms.MaxOfferDuration,
		})
	}

	return bTerms
}

func (sa *StorageAllocation) buildDbUpdates() event.Allocation {
	eAlloc := event.Allocation{
		AllocationID:      sa.ID,
		AllocationName:    sa.Name,
		TransactionID:     sa.Tx,
		DataShards:        sa.DataShards,
		ParityShards:      sa.ParityShards,
		Size:              sa.Size,
		Expiration:        int64(sa.Expiration),
		Owner:             sa.Owner,
		OwnerPublicKey:    sa.OwnerPublicKey,
		IsImmutable:       sa.IsImmutable,
		ReadPriceMin:      sa.ReadPriceRange.Min,
		ReadPriceMax:      sa.ReadPriceRange.Max,
		WritePriceMin:     sa.WritePriceRange.Min,
		WritePriceMax:     sa.WritePriceRange.Max,
		StartTime:         int64(sa.StartTime),
		Finalized:         sa.Finalized,
		Cancelled:         sa.Canceled,
		UsedSize:          sa.UsedSize,
		MovedToChallenge:  sa.MovedToChallenge,
		MovedBack:         sa.MovedBack,
		MovedToValidators: sa.MovedToValidators,
		TimeUnit:          int64(sa.TimeUnit),
		WritePool:         sa.WritePool,
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

func emitAddOrOverwriteAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteAllocationBlobberTerm, t.Hash, sa.buildEventBlobberTerms())
}

func emitUpdateAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationBlobberTerm, sa.ID, sa.buildEventBlobberTerms())
}

func emitDeleteAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagDeleteAllocationBlobberTerm, t.Hash, sa.buildEventBlobberTerms())
}
