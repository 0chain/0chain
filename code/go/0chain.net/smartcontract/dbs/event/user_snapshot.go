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

	tx := edb.Store.Get().Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	err = tx.Exec(`CREATE TEMPORARY TABLE IF NOT EXISTS user_snapshot_ids_temp
        AS SELECT t.id FROM UNNEST(?::text[]) AS t(id)`, pq.StringArray(ids),
	).Debug().Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Raw(`SELECT us.* FROM user_snapshot_ids_temp tmp
        INNER JOIN user_snapshots us ON tmp.id = us.user_id`).Scan(&snapshots).Debug().Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Exec(`DROP TABLE IF EXISTS user_snapshot_ids_temp`).Debug().Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return snapshots, nil
}

func (edb *EventDb) AddOrOverwriteUserSnapshots(snapshots []*UserSnapshot) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}
