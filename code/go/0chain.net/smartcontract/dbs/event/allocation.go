package event

import (
	"fmt"
	"time"

	"0chain.net/core/util"
	corecommon "0chain.net/core/common" 
	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

type Allocation struct {
	model.UpdatableModel
	AllocationID             string        `json:"allocation_id" gorm:"uniqueIndex"`
	TransactionID            string        `json:"transaction_id"`
	DataShards               int           `json:"data_shards"`
	ParityShards             int           `json:"parity_shards"`
	Size                     int64         `json:"size"`
	Expiration               int64         `json:"expiration"`
	Owner                    string        `json:"owner" gorm:"index:idx_aowner"`
	OwnerPublicKey           string        `json:"owner_public_key"`
	ReadPriceMin             currency.Coin `json:"read_price_min"`
	ReadPriceMax             currency.Coin `json:"read_price_max"`
	WritePriceMin            currency.Coin `json:"write_price_min"`
	WritePriceMax            currency.Coin `json:"write_price_max"`
	StartTime                int64         `json:"start_time" gorm:"index:idx_astart_time"`
	Finalized                bool          `json:"finalized"`
	Cancelled                bool          `json:"cancelled"`
	UsedSize                 int64         `json:"used_size"`
	MovedToChallenge         currency.Coin `json:"moved_to_challenge"`
	MovedBack                currency.Coin `json:"moved_back"`
	MovedToValidators        currency.Coin `json:"moved_to_validators"`
	TimeUnit                 int64         `json:"time_unit"`
	NumWrites                int64         `json:"num_writes"`
	NumReads                 int64         `json:"num_reads"`
	TotalChallenges          int64         `json:"total_challenges"`
	OpenChallenges           int64         `json:"open_challenges"`
	SuccessfulChallenges     int64         `json:"successful_challenges"`
	FailedChallenges         int64         `json:"failed_challenges"`
	LatestClosedChallengeTxn string        `json:"latest_closed_challenge_txn"`
	WritePool                currency.Coin `json:"write_pool"`
	ThirdPartyExtendable     bool          `json:"third_party_extendable"`
	FileOptions              uint16        `json:"file_options"`

	//ref
	User  User                    `gorm:"foreignKey:Owner;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Terms []AllocationBlobberTerm `json:"terms" gorm:"foreignKey:AllocationID;references:AllocationID"`
}

func (edb *EventDb) GetAllocation(id string) (*Allocation, error) {
	var alloc Allocation
	err := edb.Store.Get().Preload("Terms").Model(&Allocation{}).Where("allocation_id = ?", id).First(&alloc).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving allocation: %v, error: %v", id, err)
	}

	return &alloc, nil
}

func (edb *EventDb) GetClientsAllocation(clientID string, limit common.Pagination) ([]Allocation, error) {
	allocs := make([]Allocation, 0)

	err := edb.Store.Get().
		Preload("Terms").
		Model(&Allocation{}).Where("owner = ?", clientID).Limit(limit.Limit).Offset(limit.Offset).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "start_time"},
			Desc:   limit.IsDescending,
		}).Find(&allocs).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving allocation for client: %v, error: %v", clientID, err)
	}

	return allocs, nil
}

func (edb *EventDb) GetActiveAllocationsCount() (int64, error) {
	var count int64
	result := edb.Store.Get().Model(&Allocation{}).Where("finalized = ? AND cancelled = ?", false, false).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("error retrieving active allocations , error: %v", result.Error)
	}

	return count, nil
}

func (edb *EventDb) addAllocations(allocs []Allocation) error {
	return edb.Store.Get().Create(&allocs).Error
}

func mergeAddAllocationEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagAddAllocation)
}

func (edb *EventDb) updateAllocations(allocs []Allocation) error {
	ts := time.Now()
	updateColumns := []string{
		"allocation_name",
		"transaction_id",
		"data_shards",
		"parity_shards",
		"size",
		"expiration",
		"owner",
		"owner_public_key",
		"read_price_min",
		"read_price_max",
		"write_price_min",
		"write_price_max",
		"start_time",
		"finalized",
		"cancelled",
		"used_size",
		"moved_to_challenge",
		"moved_back",
		"moved_to_validators",
		"time_unit",
		"write_pool",
		"num_writes",
		"num_reads",
		"total_challenges",
		"open_challenges",
		"successful_challenges",
		"failed_challenges",
		"latest_closed_challenge_txn",
		"third_party_extendable",
		"file_options",
	}

	columns, err := util.Columnize(allocs)
	if err != nil {
		return err
	}
	ids, ok := columns["allocation_id"]
	if !ok {
		return corecommon.NewError("update_allocation", "no id field provided in event Data")
	}

	updater := CreateBuilder("allocations", "allocation_id", ids)
	for _, fieldKey := range updateColumns {
		if fieldKey == "allocation_id" {
			continue
		}

		fieldList, ok := columns[fieldKey]
		if !ok {
			return corecommon.NewErrorf("update_allocation", "required field %v for update is not found in provided data", fieldKey)
		}
		updater = updater.AddUpdate(fieldKey, fieldList)
	}

	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - update allocation slow",
				zap.Duration("duration", du),
				zap.Int("num", len(allocs)))
		}
	}()

	return updater.Exec(edb).Debug().Error
}

func mergeUpdateAllocEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagUpdateAllocation, withUniqueEventOverwrite())
}

func (edb *EventDb) updateAllocationStakes(allocs []Allocation) error {
	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - update allocation stakes slow",
				zap.Duration("duration", du),
				zap.Int("num", len(allocs)))
		}
	}()

	var (
		allocationIdList 	  	[]string
		writePoolList 		  	[]int64
		movedToChallengeList  	[]int64
		movedBackList		  	[]int64
		movedToValidatorsList 	[]int64
		coinValue				int64
		err						error
	)

	for _, alloc := range allocs {
		allocationIdList = append(allocationIdList, alloc.AllocationID)

		coinValue, err = alloc.WritePool.Int64()
		if err != nil {
			return err
		}
		writePoolList = append(writePoolList, coinValue)

		coinValue, err = alloc.MovedToChallenge.Int64()
		if err != nil {
			return err
		}
		movedToChallengeList = append(movedToChallengeList, coinValue)

		coinValue, err = alloc.MovedBack.Int64()
		if err != nil {
			return err
		}
		movedBackList = append(movedBackList, coinValue)

		coinValue, err = alloc.MovedToValidators.Int64()
		if err != nil {
			return err
		}
		movedToValidatorsList = append(movedToValidatorsList, coinValue)
	}

	return CreateBuilder("allocations", "allocation_id", allocationIdList).
		AddUpdate("write_pool", writePoolList).
		AddUpdate("moved_to_challenge", movedToChallengeList).
		AddUpdate("moved_back", movedBackList).
		AddUpdate("moved_to_validators", movedToValidatorsList).Exec(edb).Error
}

func mergeUpdateAllocStatsEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagUpdateAllocationStakes, withUniqueEventOverwrite())
}

func mergeAllocationStatsEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagUpdateAllocationStat, withAllocStatsMerged())
}

func withAllocStatsMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *Allocation) (*Allocation, error) {
		a.UsedSize += b.UsedSize
		a.NumWrites += b.NumWrites
		a.MovedToChallenge += b.MovedToChallenge
		a.MovedBack += b.MovedBack
		return a, nil
	})
}

func (edb *EventDb) updateAllocationsStats(allocs []Allocation) error {
	var (
		allocationIdList	[]string
		usedSizeList		[]int64
		numWritesList		[]int64
		movedToChallengeList []int64
		movedBackList		[]int64
		writePoolList		[]int64			
		coinValue			int64
		err					error
	)

	for _, alloc := range allocs {
		allocationIdList = append(allocationIdList, alloc.AllocationID)
		usedSizeList = append(usedSizeList, alloc.UsedSize)
		numWritesList = append(numWritesList, alloc.NumWrites)

		coinValue, err = alloc.WritePool.Int64()
		if err != nil {
			return err
		}
		writePoolList = append(writePoolList, coinValue)

		coinValue, err = alloc.MovedToChallenge.Int64()
		if err != nil {
			return err
		}
		movedToChallengeList = append(movedToChallengeList, coinValue)

		coinValue, err = alloc.MovedBack.Int64()
		if err != nil {
			return err
		}
		movedBackList = append(movedBackList, coinValue)
	}

	return CreateBuilder("allocations", "allocation_id", allocationIdList).
		AddUpdate("used_size", usedSizeList, "allocations.used_size + t.used_size").
		AddUpdate("num_writes", numWritesList, "allocations.num_writes + t.num_writes").
		AddUpdate("moved_to_challenge", movedToChallengeList, "allocations.moved_to_challenge + t.moved_to_challenge").
		AddUpdate("moved_back", movedBackList, "allocations.moved_back + t.moved_back").
		AddUpdate("write_pool", writePoolList, "allocations.write_pool - t.moved_to_challenge + t.moved_back").Exec(edb).Error
}

func mergeUpdateAllocBlobbersTermsEvents() *eventsMergerImpl[AllocationBlobberTerm] {
	return newEventsMerger[AllocationBlobberTerm](TagUpdateAllocationBlobberTerm, withAllocBlobberTermsMerged())
}

func mergeAddOrOverwriteAllocBlobbersTermsEvents() *eventsMergerImpl[AllocationBlobberTerm] {
	return newEventsMerger[AllocationBlobberTerm](TagAddOrOverwriteAllocationBlobberTerm, withAllocBlobberTermsMerged())
}

func mergeDeleteAllocBlobbersTermsEvents() *eventsMergerImpl[AllocationBlobberTerm] {
	return newEventsMerger[AllocationBlobberTerm](TagDeleteAllocationBlobberTerm, withAllocBlobberTermsMerged())
}

func withAllocBlobberTermsMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *[]AllocationBlobberTerm) (*[]AllocationBlobberTerm, error) {
		var (
			aMap = make(map[string]AllocationBlobberTerm, len(*a))
			pa   = *a
			pb   = *b
		)
		for i, ai := range pa {
			aMap[ai.BlobberID] = pa[i]
		}

		for _, bi := range pb {
			aMap[bi.BlobberID] = bi
		}

		ret := make([]AllocationBlobberTerm, 0, len(aMap))
		for _, v := range aMap {
			ret = append(ret, v)
		}

		return &ret, nil
	})
}

func mergeUpdateAllocChallengesEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagUpdateAllocationChallenge, withAllocChallengesMerged())
}

func withAllocChallengesMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *Allocation) (*Allocation, error) {
		a.OpenChallenges += b.OpenChallenges
		a.LatestClosedChallengeTxn = b.LatestClosedChallengeTxn
		a.SuccessfulChallenges += b.SuccessfulChallenges
		a.FailedChallenges += b.FailedChallenges

		return a, nil
	})
}

func (edb *EventDb) updateAllocationChallenges(allocs []Allocation) error {
	var (
		allocationIdList	[]string
		openChallengesList  []int64
		latestClosedChallengeTxnList []string
		successfulChallengesList []int64
		failedChallengeList []int64
	)

	for _, alloc := range allocs {
		allocationIdList = append(allocationIdList, alloc.AllocationID)
		openChallengesList = append(openChallengesList, alloc.OpenChallenges)
		latestClosedChallengeTxnList = append(latestClosedChallengeTxnList, alloc.LatestClosedChallengeTxn)
		successfulChallengesList = append(successfulChallengesList, alloc.SuccessfulChallenges)
		failedChallengeList = append(failedChallengeList, alloc.FailedChallenges)		
	}

	return CreateBuilder("allocations", "allocation_id", allocationIdList).
		AddUpdate("open_challenges", openChallengesList, "allocations.open_challenges - t.open_challenges").
		AddUpdate("latest_closed_challenge_txn", latestClosedChallengeTxnList).
		AddUpdate("successful_challenges", successfulChallengesList, "allocations.successful_challenges + t.successful_challenges").
		AddUpdate("failed_challenges", failedChallengeList, "allocations.failed_challenges + t.failed_challenges").Exec(edb).Error
}

func (edb *EventDb) addChallengesToAllocations(allocs []Allocation) error {
	var (
		allocationIdList    []string
		totalChallengesList []int64
		openChallengesList  []int64
	)

	for _, alloc := range allocs {
		allocationIdList = append(allocationIdList, alloc.AllocationID)
		totalChallengesList = append(totalChallengesList, alloc.TotalChallenges)
		openChallengesList = append(openChallengesList, alloc.OpenChallenges)
	}

	return CreateBuilder("allocations", "allocation_id", allocationIdList).
		AddUpdate("total_challenges", totalChallengesList, "allocations.total_challenges + t.total_challenges").
		AddUpdate("open_challenges", openChallengesList, "allocations.open_challenges + t.open_challenges").Exec(edb).Error
}

func mergeAddChallengesToAllocsEvents() *eventsMergerImpl[Allocation] {
	return newEventsMerger[Allocation](TagAddChallengeToAllocation, withAddChallengesToAllocMerged())
}

func withAddChallengesToAllocMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *Allocation) (*Allocation, error) {
		a.OpenChallenges += b.OpenChallenges
		a.TotalChallenges += b.TotalChallenges
		return a, nil
	})
}
