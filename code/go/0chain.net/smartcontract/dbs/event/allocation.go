package event

import (
	"fmt"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/common"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type Allocation struct {
	gorm.Model
	AllocationID             string        `json:"allocation_id" gorm:"uniqueIndex"`
	AllocationName           string        `json:"allocation_name" gorm:"column:allocation_name;size:64;"`
	TransactionID            string        `json:"transaction_id"`
	DataShards               int           `json:"data_shards"`
	ParityShards             int           `json:"parity_shards"`
	Size                     int64         `json:"size"`
	Expiration               int64         `json:"expiration"`
	Owner                    string        `json:"owner" gorm:"index:idx_aowner"`
	OwnerPublicKey           string        `json:"owner_public_key"`
	IsImmutable              bool          `json:"is_immutable"`
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
	//ref
	User  User                    `gorm:"foreignKey:Owner;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Terms []AllocationBlobberTerm `json:"terms" gorm:"foreignKey:AllocationID;references:AllocationID"`
}

func (alloc *Allocation) onUpdateChallenge(tx *gorm.DB, c *Challenge) error {
	vs := map[string]interface{}{
		"open_challenges":             gorm.Expr("allocations.open_challenges - 1"),
		"latest_closed_challenge_txn": gorm.Expr("?", c.ChallengeID),
	}

	if c.Passed {
		vs["successful_challenges"] = gorm.Expr("allocations.successful_challenges + 1")
	} else {
		vs["failed_challenges"] = gorm.Expr("allocations.failed_challenges + 1")
	}

	return tx.Model(&Allocation{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "allocation_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&Allocation{AllocationID: c.AllocationID}).Error
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

func (edb *EventDb) GetActiveAllocsBlobberCount() (int64, error) {
	var count int64
	err := edb.Store.Get().
		Raw("SELECT SUM(parity_shards) + SUM(data_shards) FROM allocations WHERE finalized = ? AND cancelled = ?",
			false, false).
		Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("error retrieving blobber allocations count, error: %v", err)
	}

	return count, nil
}

func (edb *EventDb) addAllocations(allocs []Allocation) error {
	return edb.Store.Get().Create(&allocs).Error
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
		"is_immutable",
		"read_price_min",
		"read_price_max",
		"write_price_min",
		"write_price_max",
		"challenge_completion_time",
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
	}

	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - update allocation slow",
				zap.Duration("duration", du),
				zap.Int("num", len(allocs)))
		}
	}()

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "allocation_id"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&allocs).Error
}

func (edb *EventDb) updateAllocationStakes(allocs []Allocation) error {
	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - update allocation stakes slow",
				zap.Any("duration", du),
				zap.Int("num", len(allocs)))
		}
	}()

	updateColumns := []string{
		"write_pool",
		"moved_to_challenge",
		"moved_back",
		"moved_to_validators",
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "allocation_id"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&allocs).Error
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
	// update allocation stat
	vs := map[string]interface{}{
		"used_size":          gorm.Expr("allocations.used_size + excluded.used_size"),
		"num_writes":         gorm.Expr("allocations.num_writes + excluded.num_writes"),
		"moved_to_challenge": gorm.Expr("allocations.moved_to_challenge + excluded.moved_to_challenge"),
		"moved_back":         gorm.Expr("allocations.moved_back + excluded.moved_back"),
		"write_pool":         gorm.Expr("allocations.write_pool - excluded.moved_to_challenge + excluded.moved_back"),
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "allocation_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&allocs).Error
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
