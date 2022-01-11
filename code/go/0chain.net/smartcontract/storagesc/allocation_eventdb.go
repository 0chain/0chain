package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
	"time"
)

func allocationTableToStorageAllocation(alloc *event.Allocation, balances cstate.StateContextI) (*StorageAllocation, error) {

	var (
		storageNodes         []*StorageNode
		blobberDetails       []*BlobberAllocation
		blobberIDs           []string
		blobberIDTermMapping = make(map[string]struct {
			AllocationID string
			Terms
		})
		blobberMap = make(map[string]*BlobberAllocation)
	)

	for _, t := range alloc.Terms {
		blobberIDs = append(blobberIDs, t.BlobberID)
		blobberIDTermMapping[t.BlobberID] = struct {
			AllocationID string
			Terms
		}{
			AllocationID: t.AllocationID,
			Terms: Terms{
				ReadPrice:               t.ReadPrice,
				WritePrice:              t.WritePrice,
				MinLockDemand:           t.MinLockDemand,
				MaxOfferDuration:        t.MaxOfferDuration,
				ChallengeCompletionTime: t.ChallengeCompletionTime,
			}}
	}

	blobbers, err := balances.GetEventDB().GetBlobbersFromIDs(blobberIDs)
	if err != nil {
		return nil, fmt.Errorf("error retrieving blobbers from db: %v", err)
	}

	for _, b := range blobbers {
		storageNodes = append(storageNodes, &StorageNode{
			ID:      b.BlobberID,
			BaseURL: b.BaseURL,
			Geolocation: StorageNodeGeolocation{
				Latitude:  b.Latitude,
				Longitude: b.Longitude,
			},
			Terms:           blobberIDTermMapping[b.BlobberID].Terms,
			Capacity:        b.Capacity,
			Used:            b.Used,
			LastHealthCheck: common.Timestamp(b.LastHealthCheck),
			StakePoolSettings: stakePoolSettings{
				DelegateWallet: b.DelegateWallet,
				MinStake:       state.Balance(b.MinStake),
				MaxStake:       state.Balance(b.MaxStake),
				NumDelegates:   b.NumDelegates,
				ServiceCharge:  b.ServiceCharge,
			},
		})

		tempBlobberAllocation := &BlobberAllocation{
			BlobberID:    b.BlobberID,
			AllocationID: blobberIDTermMapping[b.BlobberID].AllocationID,
			Terms:        blobberIDTermMapping[b.BlobberID].Terms,
		}
		blobberDetails = append(blobberDetails, tempBlobberAllocation)
		blobberMap[b.BlobberID] = tempBlobberAllocation
	}

	sa := &StorageAllocation{
		ID:             alloc.AllocationID,
		Tx:             alloc.TransactionID,
		DataShards:     alloc.DataShards,
		ParityShards:   alloc.ParityShards,
		Size:           alloc.Size,
		Expiration:     common.Timestamp(alloc.Expiration),
		Blobbers:       storageNodes,
		Owner:          alloc.Owner,
		OwnerPublicKey: alloc.OwnerPublicKey,
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
		BlobberDetails:             blobberDetails,
		BlobberMap:                 blobberMap,
		IsImmutable:                alloc.IsImmutable,
		ReadPriceRange:             PriceRange{alloc.ReadPriceMin, alloc.ReadPriceMax},
		WritePriceRange:            PriceRange{alloc.WritePriceMin, alloc.WritePriceMax},
		MaxChallengeCompletionTime: time.Duration(alloc.MaxChallengeCompletionTime),
		// todo: to be added with WritePool : select user_id from WritePools where allocation_id = ?
		// WritePoolOwners:            nil,
		ChallengeCompletionTime: time.Duration(alloc.ChallengeCompletionTime),
		StartTime:               common.Timestamp(alloc.StartTime),
		Finalized:               alloc.Finalized,
		Canceled:                alloc.Cancelled,
		UsedSize:                alloc.UsedSize,
		MovedToChallenge:        alloc.MovedToChallenge,
		MovedBack:               alloc.MovedBack,
		MovedToValidators:       alloc.MovedToValidators,
		TimeUnit:                time.Duration(alloc.TimeUnit),
		Curators:                alloc.Curators,
	}

	return sa, nil
}

func storageAllocationToAllocationTable(sa *StorageAllocation) (*event.Allocation, error) {

	var allocationTerms []*event.AllocationTerm
	for _, b := range sa.BlobberDetails {
		allocationTerms = append(allocationTerms, &event.AllocationTerm{
			BlobberID:               b.BlobberID,
			AllocationID:            b.AllocationID,
			ReadPrice:               b.Terms.ReadPrice,
			WritePrice:              b.Terms.WritePrice,
			MinLockDemand:           b.Terms.MinLockDemand,
			MaxOfferDuration:        b.Terms.MaxOfferDuration,
			ChallengeCompletionTime: b.Terms.ChallengeCompletionTime,
		})
	}

	return &event.Allocation{
		AllocationID:               sa.ID,
		TransactionID:              sa.Tx,
		DataShards:                 sa.DataShards,
		ParityShards:               sa.ParityShards,
		Size:                       sa.Size,
		Expiration:                 int64(sa.Expiration),
		Terms:                      allocationTerms,
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
		Curators:                   sa.Curators,
		TimeUnit:                   int64(sa.TimeUnit),
		NumWrites:                  sa.Stats.NumWrites,
		NumReads:                   sa.Stats.NumReads,
		TotalChallenges:            sa.Stats.TotalChallenges,
		OpenChallenges:             sa.Stats.OpenChallenges,
		SuccessfulChallenges:       sa.Stats.SuccessChallenges,
		FailedChallenges:           sa.Stats.FailedChallenges,
		LatestClosedChallengeTxn:   sa.Stats.LastestClosedChallengeTxn,
	}, nil
}

func emitAddOrOverwriteAllocation(sa *StorageAllocation, balances cstate.StateContextI) error {

	alloc, err := storageAllocationToAllocationTable(sa)
	if err != nil {
		return err
	}

	data, err := json.Marshal(alloc)
	if err != nil {
		return fmt.Errorf("error marshalling allocation: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteAllocation, alloc.AllocationID, string(data))

	return nil
}

func getStorageAllocationFromDb(id string, balances cstate.StateContextI) (*StorageAllocation, error) {

	alloc, err := balances.GetEventDB().GetAllocation(id)
	if err != nil {
		return nil, err
	}

	sa, err := allocationTableToStorageAllocation(alloc, balances)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
