package event

import (
	"time"

	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	model.UpdatableModel
	UserID          string        `json:"user_id" gorm:"uniqueIndex"`
	BucketID        int64         `json:"bucket_id" gorm:"not null,default:0"`
	TxnHash         string        `json:"txn_hash"`
	Balance         currency.Coin `json:"balance"`
	Change          currency.Coin `json:"change"`
	Round           int64         `json:"round"`
	Nonce           int64         `json:"nonce"`
	CollectedReward currency.Coin `json:"collected_reward"`
	TotalStake      currency.Coin `json:"total_stake"`
	ReadPoolTotal   currency.Coin `json:"read_pool_total"`
	WritePoolTotal  currency.Coin `json:"write_pool_total"`
	PayedFees       currency.Coin `json:"payed_fees"`
}

func (edb *EventDb) GetUser(userID string) (*User, error) {
	var user User
	err := edb.Store.Get().Model(&User{}).
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil && err == gorm.ErrRecordNotFound {
		return nil, util.ErrValueNotPresent
	}

	return &user, nil
}

// update or create users
func (edb *EventDb) addOrUpdateUsers(users []User) error {
	ts := time.Now()
	defer func() {
		logging.Logger.Debug("event db - upsert users ", zap.Duration("duration", time.Since(ts)),
			zap.Int("num", len(users)))
	}()
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"txn_hash", "round", "balance", "nonce"}),
	}).Create(&users).Error
}

func (edb *EventDb) updateUserCollectedRewards(users []User) error {
	var ids []string
	var collectedRewards []uint64
	for _, u := range users {
		ids = append(ids, u.UserID)
		collectedRewards = append(collectedRewards, uint64(u.CollectedReward))
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("collected_reward", collectedRewards, "users.collected_reward + t.collected_reward").Exec(edb).Error
}

func (edb *EventDb) updateUserTotalStake(users []User, shouldIncrease bool) error {
	var ids []string
	var stakes []uint64
	for _, u := range users {
		ids = append(ids, u.UserID)
		stakes = append(stakes, uint64(u.TotalStake))
	}

	operation := "+"
	if !shouldIncrease {
		operation = "-"
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("total_stake", stakes, "users.total_stake "+operation+" t.total_stake").Exec(edb).Error
}

func (edb *EventDb) updateUserReadPoolTotal(users []User, shouldIncrease bool) error {
	var ids []string
	var readpools []uint64
	for _, u := range users {
		ids = append(ids, u.UserID)
		readpools = append(readpools, uint64(u.ReadPoolTotal))
	}
	operation := "+"
	if !shouldIncrease {
		operation = "-"
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("read_pool_total", readpools, "users.read_pool_total "+operation+" t.read_pool_total").Exec(edb).Error
}

func (edb *EventDb) updateUserWritePoolTotal(users []User, shouldIncrease bool) error {
	var ids []string
	var writepools []uint64
	for _, u := range users {
		ids = append(ids, u.UserID)
		writepools = append(writepools, uint64(u.WritePoolTotal))
	}

	operation := "+"
	if !shouldIncrease {
		operation = "-"
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("write_pool_total", writepools, "users.write_pool_total "+operation+" t.write_pool_total").Exec(edb).Error
}

func (edb *EventDb) updateUserPayedFees(users []User) error {
	var ids []string
	var fees []uint64
	for _, u := range users {
		ids = append(ids, u.UserID)
		fees = append(fees, uint64(u.PayedFees))
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("payed_fees", fees, "users.payed_fees + t.payed_fees").Exec(edb).Error
}

func mergeUpdateUserCollectedRewardsEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUpdateUserCollectedRewards, withUniqueEventOverwrite())
}

func mergeIncreaseUserTotalStakeEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagLockStakePool, withUniqueEventOverwrite())
}
func mergeDecreaseUserTotalStakeEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUnlockStakePool, withUniqueEventOverwrite())
}

func mergeIncreaseUserReadPoolTotalEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagLockReadPool, withUniqueEventOverwrite())
}

func mergeDecreaseUserReadPoolTotalEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUnlockReadPool, withUniqueEventOverwrite())
}

func mergeIncreaseUserWritePoolTotalEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagLockWritePool, withUniqueEventOverwrite())
}

func mergeDecreaseUserWritePoolTotalEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUnlockWritePool, withUniqueEventOverwrite())
}

func mergeUpdateUserPayedFeesEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUpdateUserPayedFees, withUniqueEventOverwrite())
}

func mergeAddUsersEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagAddOrOverwriteUser, withUniqueEventOverwrite())
}
