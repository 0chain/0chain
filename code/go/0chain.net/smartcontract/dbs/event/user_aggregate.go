package event

import (
	"time"

	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserAggregate struct {
	UserID          string `json:"user_id" gorm:"uniqueIndex"`
	Round           int64  `json:"round"`
	CollectedReward int64  `json:"collected_reward"`
	TotalStake      int64  `json:"total_stake"`
	ReadPoolTotal   int64  `json:"read_pool_total"`
	WritePoolTotal  int64  `json:"write_pool_total"`
	PayedFees       int64  `json:"payed_fees"`
	CreatedAt       time.Time
}

func (edb *EventDb) ReplicateUserAggregate(round int64, limit int, offset int) ([]UserAggregate, error) {
	var snapshots []UserAggregate
	var result *gorm.DB
	result = edb.Store.Get().
		Raw("SELECT * FROM user_aggregates WHERE round = ? ORDER BY user_id ASC LIMIT ? OFFSET ?", round, limit, offset).Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}
	return snapshots, nil
}

func (edb *EventDb) updateUserAggregate(round, pageAmount int64, gs *globalSnapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS temp_user_ids "+
		"ON COMMIT DROP AS SELECT user_id as id FROM users where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	var count int64
	r := edb.Store.Get().Raw("SELECT count(*) FROM temp_user_ids").Scan(&count)
	if r.Error != nil {
		logging.Logger.Error("getting user ids count", zap.Error(r.Error))
		return
	}
	if count == 0 {
		return
	}
	pageCount := count / edb.PageLimit()

	for i := int64(0); i <= pageCount; i++ {
		edb.calculateUserAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateUserAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from temp_user_ids ORDER BY id limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting user aggregate ids", zap.Int("num", len(ids)))

	var currentUsers []User
	result := edb.Store.Get().Model(&User{}).
		Where("users.user_id in (select id from temp_user_ids ORDER BY id limit ? offset ?)", limit, offset).
		Find(&currentUsers)

	if result.Error != nil {
		logging.Logger.Error("getting current users", zap.Error(result.Error))
		return
	}
	logging.Logger.Debug("user_snapshot", zap.Int("total_current_users", len(currentUsers)))

	if round <= edb.AggregatePeriod() && len(currentUsers) > 0 {
		if err := edb.addUserSnapshot(currentUsers); err != nil {
			logging.Logger.Error("saving users snapshots", zap.Error(err))
		}
	}

	oldUsers, err := edb.getUserSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting user snapshots", zap.Error(err))
		return
	}
	logging.Logger.Debug("user_snapshot", zap.Int("total_old_users", len(oldUsers)))

	var aggregates []UserAggregate
	for _, current := range currentUsers {
		old, found := oldUsers[current.UserID]
		if !found {
			continue
		}
		aggregate := UserAggregate{
			Round:  round,
			UserID: current.UserID,
		}
		aggregate.CollectedReward = (old.CollectedReward + current.CollectedReward) / 2
		aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
		aggregate.ReadPoolTotal = (old.ReadPoolTotal + current.ReadPoolTotal) / 2
		aggregate.WritePoolTotal = (old.WritePoolTotal + current.WritePoolTotal) / 2
		aggregate.PayedFees = (old.PayedFees + current.PayedFees) / 2

		aggregates = append(aggregates, aggregate)
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}
	logging.Logger.Debug("user_snapshot", zap.Int("aggregates", len(aggregates)))

	if len(currentUsers) > 0 {
		if err := edb.addUserSnapshot(currentUsers); err != nil {
			logging.Logger.Error("saving users snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("user_snapshot", zap.Int("current_users", len(currentUsers)))
}
