package event

import (
	"time"

	"gorm.io/gorm/clause"
)


type UserSnapshot struct {
	UserID          string `json:"user_id" gorm:"uniqueIndex"`
	Round           int64  `json:"round"`
	CollectedReward int64  `json:"collected_reward"`
	TotalStake      int64  `json:"total_stake"`
	ReadPoolTotal   int64  `json:"read_pool_total"`
	WritePoolTotal  int64  `json:"write_pool_total"`
	PayedFees       int64  `json:"payed_fees"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (edb *EventDb) GetUserSnapshotsByIds(ids []string) (snapshots []UserSnapshot, err error) {
	err = edb.Store.Get().Where("user_id in (?)", ids).Find(&snapshots).Error
	return
}

func (edb *EventDb) AddOrOverwriteUserSnapshots(snapshots []UserSnapshot) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:  []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}