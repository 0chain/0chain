package event

import (
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model UserSnapshot
type UserSnapshot struct {
	Round int64 `json:"round"`
	AggregateValues
}

func (edb *EventDb) getUserSnapshots(limit, offset int64) (map[string]UserSnapshot, error) {
	var snapshots []UserSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM user_snapshots WHERE user_id in (select id from temp_user_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]UserSnapshot, len(snapshots))
	logging.Logger.Debug("get_user_snapshot", zap.Int("snapshots selected", len(snapshots)))
	logging.Logger.Debug("get_user_snapshot", zap.Int64("snapshots rows selected", result.RowsAffected))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.UserID] = snapshot
	}

	result = edb.Store.Get().Where("user_id IN (select id from temp_user_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&UserSnapshot{})
	logging.Logger.Debug("get_user_snapshot", zap.Int64("deleted rows", result.RowsAffected))
	return mapSnapshots, result.Error
}

func (edb *EventDb) addUserSnapshot(users []User) error {
	var snapshots []UserSnapshot
	for _, user := range users {
		snap := UserSnapshot{
			Round: user.Round,
			AggregateValues: AggregateValues{
				UserID:          user.UserID,
				CollectedReward: user.CollectedReward,
				TotalStake:      user.TotalStake,
				ReadPoolTotal:   user.ReadPoolTotal,
				WritePoolTotal:  user.WritePoolTotal,
				PayedFees:       user.PayedFees,
			},
		}
		snapshots = append(snapshots, snap)
	}

	return edb.Store.Get().Create(&snapshots).Error
}
