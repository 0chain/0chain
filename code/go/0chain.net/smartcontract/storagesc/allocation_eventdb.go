package storagesc

import (
	"fmt"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
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

	for _, t := range alloc.Terms {
		blobberIDs = append(blobberIDs, t.BlobberID)
		blobberTermsMap[t.BlobberID] = Terms{
			ReadPrice:  currency.Coin(t.ReadPrice),
			WritePrice: currency.Coin(t.WritePrice),
		}
	}

	logging.Logger.Info("Jayash1", zap.Any("blobberIDs", blobberIDs))

	blobbers, err := eventDb.GetBlobbersFromIDs(blobberIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobbers from db: %v", err)
	}

	logging.Logger.Info("Jayash2", zap.Any("blobbers", blobbers))

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
			IsRestricted:    b.IsRestricted,
			IsSpecialStatus: b.IsSpecialStatus,
		})

		terms := blobberTermsMap[b.ID]

		ba := &BlobberAllocation{
			BlobberID:    b.ID,
			AllocationID: alloc.AllocationID,
			Size:         blobberSize,
			Terms:        terms,
		}
		blobberDetails = append(blobberDetails, ba)
	}

	logging.Logger.Info("Jayash2.1", zap.Any("blobbers", blobbers), zap.Any("blobberDetails", blobberDetails), zap.Any("storageNodes", storageNodes))

	sa := &StorageAllocation{}
	sa.SetEntity(&storageAllocationV2{
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
		IsSpecialStatus:   &alloc.IsSpecialStatus,
	})

	logging.Logger.Info("Jayash3", zap.Any("sa", sa))

	res := &StorageAllocationBlobbers{
		StorageAllocation: *sa,
		Blobbers:          storageNodes,
	}

	logging.Logger.Info("Jayash3.1", zap.Any("res", res), zap.Any("storagenodes", storageNodes))

	return res, nil
}

func storageAllocationToAllocationTable(sa *StorageAllocation) *event.Allocation {
	sab := sa.mustBase()
	alloc := &event.Allocation{
		AllocationID:         sab.ID,
		TransactionID:        sab.Tx,
		DataShards:           sab.DataShards,
		ParityShards:         sab.ParityShards,
		Size:                 sab.Size,
		Expiration:           int64(sab.Expiration),
		Terms:                sab.buildEventBlobberTerms(),
		Owner:                sab.Owner,
		OwnerPublicKey:       sab.OwnerPublicKey,
		ReadPriceMin:         sab.ReadPriceRange.Min,
		ReadPriceMax:         sab.ReadPriceRange.Max,
		WritePriceMin:        sab.WritePriceRange.Min,
		WritePriceMax:        sab.WritePriceRange.Max,
		StartTime:            int64(sab.StartTime),
		Finalized:            sab.Finalized,
		Cancelled:            sab.Canceled,
		UsedSize:             sab.Stats.UsedSize,
		MovedToChallenge:     sab.MovedToChallenge,
		MovedBack:            sab.MovedBack,
		MovedToValidators:    sab.MovedToValidators,
		TimeUnit:             int64(sab.TimeUnit),
		WritePool:            sab.WritePool,
		ThirdPartyExtendable: sab.ThirdPartyExtendable,
		FileOptions:          sab.FileOptions,
	}

	if v2 := sa.Entity().(*storageAllocationV2); v2.IsSpecialStatus != nil {
		alloc.IsSpecialStatus = *v2.IsSpecialStatus
	}

	if sab.Stats != nil {
		alloc.NumWrites = sab.Stats.NumWrites
		alloc.NumReads = sab.Stats.NumReads
		alloc.TotalChallenges = sab.Stats.TotalChallenges
		alloc.OpenChallenges = sab.Stats.OpenChallenges
		alloc.SuccessfulChallenges = sab.Stats.SuccessChallenges
		alloc.FailedChallenges = sab.Stats.FailedChallenges
		alloc.LatestClosedChallengeTxn = sab.Stats.LastestClosedChallengeTxn
	}

	return alloc
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
