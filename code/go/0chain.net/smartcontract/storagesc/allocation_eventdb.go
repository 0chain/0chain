package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
)

func storageAllocationToAllocationTable(sa *StorageAllocation) (*event.Allocation, error) {

	var allocationTerms []*event.AllocationTerm
	for _, b := range sa.BlobberDetails {
		allocationTerms = append(allocationTerms, &event.AllocationTerm{
			BlobberID:               b.BlobberID,
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
