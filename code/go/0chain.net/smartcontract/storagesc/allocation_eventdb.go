package storagesc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/smartcontract/dbs"

	"0chain.net/chaincore/currency"
	common2 "0chain.net/smartcontract/common"

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
	blobberIDTermMapping := make(map[string]struct {
		AllocationID string
		Terms
	})
	blobberMap := make(map[string]*BlobberAllocation)

	curators, err := eventDb.GetCuratorsByAllocationID(alloc.AllocationID)
	if err != nil {
		return nil, fmt.Errorf("error finding curators: %v", err)
	}

	var allocTerms []event.AllocationTerm
	err = json.Unmarshal([]byte(alloc.Terms), &allocTerms)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling allocation terms: %v", err)
	}

	for _, t := range allocTerms {
		blobberIDs = append(blobberIDs, t.BlobberID)
		blobberIDTermMapping[t.BlobberID] = struct {
			AllocationID string
			Terms
		}{
			AllocationID: t.AllocationID,
			Terms: Terms{
				ReadPrice:        t.ReadPrice,
				WritePrice:       t.WritePrice,
				MinLockDemand:    t.MinLockDemand,
				MaxOfferDuration: t.MaxOfferDuration,
			}}
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
			ID:      b.BlobberID,
			BaseURL: b.BaseURL,
			Geolocation: StorageNodeGeolocation{
				Latitude:  b.Latitude,
				Longitude: b.Longitude,
			},
			Terms:           blobberIDTermMapping[b.BlobberID].Terms,
			Capacity:        b.Capacity,
			Allocated:       b.Allocated,
			SavedData:       b.SavedData,
			LastHealthCheck: common.Timestamp(b.LastHealthCheck),
			StakePoolSettings: stakepool.Settings{
				DelegateWallet:     b.DelegateWallet,
				MinStake:           currency.Coin(b.MinStake),
				MaxStake:           currency.Coin(b.MaxStake),
				MaxNumDelegates:    b.NumDelegates,
				ServiceChargeRatio: b.ServiceCharge,
			},
		})

		terms := blobberIDTermMapping[b.BlobberID].Terms
		tempBlobberAllocation := &BlobberAllocation{
			BlobberID:     b.BlobberID,
			AllocationID:  blobberIDTermMapping[b.BlobberID].AllocationID,
			Size:          b.Allocated,
			Terms:         terms,
			MinLockDemand: currency.Coin(float64(terms.WritePrice) * gbSize * terms.MinLockDemand * rdtu),
		}
		blobberDetails = append(blobberDetails, tempBlobberAllocation)
		blobberMap[b.BlobberID] = tempBlobberAllocation
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
		BlobberAllocs:           blobberDetails,
		BlobberAllocsMap:        blobberMap,
		IsImmutable:             alloc.IsImmutable,
		ReadPriceRange:          PriceRange{alloc.ReadPriceMin, alloc.ReadPriceMax},
		WritePriceRange:         PriceRange{alloc.WritePriceMin, alloc.WritePriceMax},
		ChallengeCompletionTime: time.Duration(alloc.ChallengeCompletionTime),
		StartTime:               common.Timestamp(alloc.StartTime),
		Finalized:               alloc.Finalized,
		Canceled:                alloc.Cancelled,
		UsedSize:                alloc.UsedSize,
		MovedToChallenge:        alloc.MovedToChallenge,
		MovedBack:               alloc.MovedBack,
		MovedToValidators:       alloc.MovedToValidators,
		TimeUnit:                time.Duration(alloc.TimeUnit),
		Curators:                curators,
	}

	return &StorageAllocationBlobbers{
		StorageAllocation: *sa,
		Blobbers:          storageNodes,
	}, nil
}

func (sa *StorageAllocation) marshalTerms() ([]byte, error) {
	allocationTerms := make([]event.AllocationTerm, 0)
	for _, b := range sa.BlobberAllocs {
		allocationTerms = append(allocationTerms, event.AllocationTerm{
			BlobberID:        b.BlobberID,
			AllocationID:     b.AllocationID,
			ReadPrice:        b.Terms.ReadPrice,
			WritePrice:       b.Terms.WritePrice,
			MinLockDemand:    b.Terms.MinLockDemand,
			MaxOfferDuration: b.Terms.MaxOfferDuration,
		})
	}

	termsByte, err := json.Marshal(allocationTerms)
	if err != nil {
		return nil, fmt.Errorf("error marshalling terms: %v", err)
	}
	return termsByte, nil
}

func storageAllocationToAllocationTable(sa *StorageAllocation) (*event.Allocation, error) {
	termsByte, err := sa.marshalTerms()
	if err != nil {
		return nil, err
	}

	alloc := &event.Allocation{
		AllocationID:            sa.ID,
		AllocationName:          sa.Name,
		TransactionID:           sa.Tx,
		DataShards:              sa.DataShards,
		ParityShards:            sa.ParityShards,
		Size:                    sa.Size,
		Expiration:              int64(sa.Expiration),
		Terms:                   string(termsByte),
		Owner:                   sa.Owner,
		OwnerPublicKey:          sa.OwnerPublicKey,
		IsImmutable:             sa.IsImmutable,
		ReadPriceMin:            sa.ReadPriceRange.Min,
		ReadPriceMax:            sa.ReadPriceRange.Max,
		WritePriceMin:           sa.WritePriceRange.Min,
		WritePriceMax:           sa.WritePriceRange.Max,
		ChallengeCompletionTime: int64(sa.ChallengeCompletionTime),
		StartTime:               int64(sa.StartTime),
		Finalized:               sa.Finalized,
		Cancelled:               sa.Canceled,
		UsedSize:                sa.UsedSize,
		MovedToChallenge:        sa.MovedToChallenge,
		MovedBack:               sa.MovedBack,
		MovedToValidators:       sa.MovedToValidators,
		TimeUnit:                int64(sa.TimeUnit),
		WritePool:               sa.WritePool,
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

	return alloc, nil
}

func (sa *StorageAllocation) buildDbUpdates() *dbs.DbUpdates {

	termsByte, _ := sa.marshalTerms() //err always is nil

	dbUpdates := &dbs.DbUpdates{
		Id: sa.ID,
		Updates: map[string]interface{}{
			"allocation_name":           sa.Name,
			"transaction_id":            sa.Tx,
			"data_shards":               sa.DataShards,
			"parity_shards":             sa.ParityShards,
			"size":                      sa.Size,
			"expiration":                int64(sa.Expiration),
			"terms":                     string(termsByte),
			"owner":                     sa.Owner,
			"owner_public_key":          sa.OwnerPublicKey,
			"is_immutable":              sa.IsImmutable,
			"read_price_min":            sa.ReadPriceRange.Min,
			"read_price_max":            sa.ReadPriceRange.Max,
			"write_price_min":           sa.WritePriceRange.Min,
			"write_price_max":           sa.WritePriceRange.Max,
			"challenge_completion_time": int64(sa.ChallengeCompletionTime),
			"start_time":                int64(sa.StartTime),
			"finalized":                 sa.Finalized,
			"cancelled":                 sa.Canceled,
			"used_size":                 sa.UsedSize,
			"moved_to_challenge":        sa.MovedToChallenge,
			"moved_back":                sa.MovedBack,
			"moved_to_validators":       sa.MovedToValidators,
			"time_unit":                 int64(sa.TimeUnit),
			"write_pool":                sa.WritePool,
		},
	}

	if sa.Stats != nil {
		dbUpdates.Updates["num_writes"] = sa.Stats.NumWrites
		dbUpdates.Updates["num_reads"] = sa.Stats.NumReads
		dbUpdates.Updates["total_challenges"] = sa.Stats.TotalChallenges
		dbUpdates.Updates["open_challenges"] = sa.Stats.OpenChallenges
		dbUpdates.Updates["successful_challenges"] = sa.Stats.SuccessChallenges
		dbUpdates.Updates["failed_challenges"] = sa.Stats.FailedChallenges
		dbUpdates.Updates["latest_closed_challenge_txn"] = sa.Stats.LastestClosedChallengeTxn
	}
	return dbUpdates
}

func (sa *StorageAllocation) emitAdd(balances cstate.StateContextI) error {
	alloc, err := storageAllocationToAllocationTable(sa)
	if err != nil {
		return err
	}

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
