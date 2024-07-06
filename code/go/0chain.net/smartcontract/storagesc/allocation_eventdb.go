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
			ReadPrice:  currency.Coin(t.ReadPrice),
			WritePrice: currency.Coin(t.WritePrice),
		}
	}

	blobbers, err := eventDb.GetBlobbersFromIDs(blobberIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobbers from db: %v", err)
	}

	blobberSize := bSize(alloc.Size, alloc.DataShards)

	for _, b := range blobbers {
		storageNodes = append(storageNodes, &storageNodeResponse{
			ID:              b.ID,
			BaseURL:         b.BaseURL,
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

		ba := &BlobberAllocation{
			BlobberID:    b.ID,
			AllocationID: alloc.AllocationID,
			Size:         blobberSize,
			Terms:        terms,
		}
		blobberDetails = append(blobberDetails, ba)
		blobberMap[b.ID] = ba
	}

	sa := &StorageAllocation{}

	_ = sa.mustUpdateBase(func(base *storageAllocationBase) error {
		base.ID = alloc.AllocationID
		base.Tx = alloc.TransactionID
		base.DataShards = alloc.DataShards
		base.ParityShards = alloc.ParityShards
		base.Size = alloc.Size
		base.Expiration = common.Timestamp(alloc.Expiration)
		base.Owner = alloc.Owner
		base.OwnerPublicKey = alloc.OwnerPublicKey
		base.WritePool = alloc.WritePool
		base.ThirdPartyExtendable = alloc.ThirdPartyExtendable
		base.FileOptions = alloc.FileOptions
		base.Stats = &StorageAllocationStats{
			UsedSize:                  alloc.UsedSize,
			NumWrites:                 alloc.NumWrites,
			NumReads:                  alloc.NumReads,
			TotalChallenges:           alloc.TotalChallenges,
			OpenChallenges:            alloc.OpenChallenges,
			SuccessChallenges:         alloc.SuccessfulChallenges,
			FailedChallenges:          alloc.FailedChallenges,
			LastestClosedChallengeTxn: alloc.LatestClosedChallengeTxn,
		}
		base.BlobberAllocs = blobberDetails
		base.BlobberAllocsMap = blobberMap
		base.ReadPriceRange = PriceRange{alloc.ReadPriceMin, alloc.ReadPriceMax}
		base.WritePriceRange = PriceRange{alloc.WritePriceMin, alloc.WritePriceMax}
		base.StartTime = common.Timestamp(alloc.StartTime)
		base.Finalized = alloc.Finalized
		base.Canceled = alloc.Cancelled
		base.MovedToChallenge = alloc.MovedToChallenge
		base.MovedBack = alloc.MovedBack
		base.MovedToValidators = alloc.MovedToValidators
		base.TimeUnit = time.Duration(alloc.TimeUnit)

		return nil
	})

	return &StorageAllocationBlobbers{
		StorageAllocation: *sa,
		Blobbers:          storageNodes,
	}, nil
}

func storageAllocationToAllocationTable(sa *storageAllocationBase) *event.Allocation {
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
		UsedSize:             sa.Stats.UsedSize,
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

func (sa *StorageAllocation) emitAdd(balances cstate.StateContextI) error {
	alloc := storageAllocationToAllocationTable(sa.mustBase())
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

func getExpiredAllocationsFromDb(blobberID string, eventDb *event.EventDb) ([]string, error) {
	allocs, err := eventDb.GetExpiredAllocation(blobberID)
	if err != nil {
		return nil, err
	}

	return allocs, nil
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
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteAllocationBlobberTerm, t.Hash, sa.mustBase().buildEventBlobberTerms())
}

//nolint:unused
func emitUpdateAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationBlobberTerm, sa.mustBase().ID, sa.mustBase().buildEventBlobberTerms())
}

//nolint:unused
func emitDeleteAllocationBlobberTerms(sa *StorageAllocation, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeStats, event.TagDeleteAllocationBlobberTerm, t.Hash, sa.mustBase().buildEventBlobberTerms())
}
