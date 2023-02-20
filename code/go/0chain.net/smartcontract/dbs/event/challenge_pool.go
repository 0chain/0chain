package event

import (
	"0chain.net/smartcontract/common"
	"gorm.io/gorm/clause"
)

type ChallengePool struct {
	ID           string `gorm:"primarykey"`
	AllocationID string `gorm:"uniqueIndex"`
	Balance      int64  `json:"balance"`
	StartTime    int64  `json:"start_time"`
	Expiration   int64  `json:"expiration"`
	Finalized    bool   `json:"finalized"`
}

func (edb *EventDb) addOrUpdateChallengePools(cps []ChallengePool) error {
	updateFields := []string{"balance", "start_time", "expiration", "finalized"}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(updateFields), // column needed to be updated
	}).Create(&cps).Error
}

func mergeAddChallengePoolsEvents() *eventsMergerImpl[ChallengePool] {
	return newEventsMerger[ChallengePool](TagAddOrUpdateChallengePool, withUniqueEventOverwrite())
}

func (edb *EventDb) GetChallengePool(allocationID string, pagination common.Pagination) (*ChallengePool, error) {
	var cp ChallengePool
	return &cp, edb.Store.Get().Model(&ChallengePool{}).
		Where("allocation_id = ?", allocationID).
		Offset(pagination.Offset).Limit(pagination.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
		}).
		Take(&cp).Error
}
