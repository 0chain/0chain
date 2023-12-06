package event

import (
	"time"

	"github.com/lib/pq"
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
	if len(ids) == 0 {
		return snapshots, nil
	}

	err = edb.Store.Get().Raw(`
			SELECT us.* 
			FROM user_snapshots us 
			WHERE us.user_id IN (SELECT t.id FROM UNNEST(?::text[]) AS t(id))`,
		pq.StringArray(ids)).Scan(&snapshots).Debug().Error
	return
}

func (edb *EventDb) AddOrOverwriteUserSnapshots(snapshots []*UserSnapshot) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}
