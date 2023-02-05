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
	CollectedReward int64         `json:"collected_reward"`
	TotalStake      int64         `json:"total_stake"`
	ReadPoolTotal   int64         `json:"read_pool_total"`
	WritePoolTotal  int64         `json:"write_pool_total"`
	PayedFees       int64         `json:"payed_fees"`
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
	var collectedRewards []int64
	for _, u := range users {
		ids = append(ids, u.UserID)
		collectedRewards = append(collectedRewards, u.CollectedReward)
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("collected_reward", collectedRewards, "users.collected_rewards + t.collected_reward").Exec(edb).Error
}

func (edb *EventDb) updateUserTotalStake(dpls []DelegatePoolLock, shouldIncrease bool) error {
	var ids []string
	var stakes []int64
	for _, dpl := range dpls {
		ids = append(ids, dpl.Client)
		if shouldIncrease {
			stakes = append(stakes, dpl.Amount)
			continue
		}
		stakes = append(stakes, -dpl.Amount)
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("total_stake", stakes, "users.total_stake + t.total_stake").Exec(edb).Error
}

func (edb *EventDb) updateUserReadPoolTotal(rpls []ReadPoolLock, shouldIncrease bool) error {
	var ids []string
	var readpools []int64
	for _, rpl := range rpls {
		ids = append(ids, rpl.Client)
		if shouldIncrease {
			readpools = append(readpools, rpl.Amount)
			continue
		}
		readpools = append(readpools, -rpl.Amount)

	}
	return CreateBuilder("users", "user_id", ids).
		AddUpdate("read_pool_total", readpools, "users.read_pool_total + t.read_pool_total").Exec(edb).Error
}

func (edb *EventDb) updateUserWritePoolTotal(wpls []WritePoolLock, shouldIncrease bool) error {
	var ids []string
	var writepools []int64
	for _, wpl := range wpls {
		ids = append(ids, wpl.Client)
		if shouldIncrease {
			writepools = append(writepools, wpl.Amount)
			continue
		}
		writepools = append(writepools, -wpl.Amount)

	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("write_pool_total", writepools, "users.write_pool_total + t.write_pool_total").Exec(edb).Error
}

func (edb *EventDb) updateUserPayedFees(users []User) error {
	var ids []string
	var fees []int64
	for _, u := range users {
		ids = append(ids, u.UserID)
		fees = append(fees, u.PayedFees)
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("payed_fees", fees, "users.payed_fees + t.payed_fees").Exec(edb).Error
}

func mergeUpdateUserCollectedRewardsEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUpdateUserCollectedRewards, withUniqueEventOverwrite())
}

func mergeUpdateUserTotalStakeEvents() *eventsMergerImpl[DelegatePoolLock] {
	return newEventsMerger[DelegatePoolLock](TagLockStakePool, withUniqueEventOverwrite())
}

func mergeUpdateUserReadPoolTotalEvents() *eventsMergerImpl[ReadPoolLock] {
	return newEventsMerger[ReadPoolLock](TagLockReadPool, withUniqueEventOverwrite())
}

func mergeUpdateUserWritePoolTotalEvents() *eventsMergerImpl[WritePoolLock] {
	return newEventsMerger[WritePoolLock](TagLockWritePool, withUniqueEventOverwrite())
}

func mergeUpdateUserPayedFeesEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagUpdateUserPayedFees, withUniqueEventOverwrite())
}

func mergeAddUsersEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagAddOrOverwriteUser, withUniqueEventOverwrite())
}
