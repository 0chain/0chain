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
	err = edb.Store.Get().Exec(`CREATE TEMPORARY TABLE IF NOT EXISTS user_snapshot_ids_temp
		ON COMMIT DROP AS SELECT t.id FROM UNNEST(?::[]text) AS t(id)`, pq.StringArray(ids),
	).Error
	if err != nil {
		return
	}
	
	err = edb.Store.Get().Exec(`SELECT us.* FROM user_snapshot_ids_temp tmp
		INNER JOIN user_snapshots us ON tmp.id = us.user_id`).Scan(&snapshots).Error
	return
}

func (edb *EventDb) AddOrOverwriteUserSnapshots(snapshots []*UserSnapshot) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:  []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}