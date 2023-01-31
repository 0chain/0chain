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

func (edb *EventDb) updateUserCollectedRewards(rms []RewardMint) error {
	var ids []string
	var collectedRewards []int64
	for _, rm := range rms {
		ids = append(ids, rm.ClientID)
		collectedRewards = append(collectedRewards, rm.Amount)
	}

	return CreateBuilder("user", "user_id", ids).
		AddUpdate("collected_reward", collectedRewards, "users.collected_reward + t.amount").Exec(edb).Error
}

func (edb *EventDb) updateUserTotalStake(dpls []DelegatePoolLock, shouldIncrease bool) error {
	var ids []string
	var stakes []int64
	for _, dpl := range dpls {
		ids = append(ids, dpl.Client)
		stakes = append(stakes, dpl.Amount)
	}

	operation := "+"
	if !shouldIncrease {
		operation = "-"
	}

	return CreateBuilder("users", "user_id", ids).
		AddUpdate("total_stake", stakes, "users.total_stake "+operation+" t.amount").Exec(edb).Error
}

func (edb *EventDb) updateUserReadPoolTotal(rpls []ReadPoolLock, shouldIncrease bool) error {
	var ids []string
	var readpools []int64
	for _, rpl := range rpls {
		ids = append(ids, rpl.Client)
		readpools = append(readpools, rpl.Amount)
	}
	operation := "+"
	if !shouldIncrease {
		operation = "-"
	}

	return CreateBuilder("user", "user_id", ids).
		AddUpdate("read_pool_total", readpools, "users.read_pool_total "+operation+" t.amount").Exec(edb).Error
}

func (edb *EventDb) updateUserWritePoolTotal(wpls []WritePoolLock, shouldIncrease bool) error {
	var ids []string
	var writepools []int64
	for _, rpl := range wpls {
		ids = append(ids, rpl.Client)
		writepools = append(writepools, rpl.Amount)
	}

	operation := "+"
	if !shouldIncrease {
		operation = "-"
	}

	return CreateBuilder("user", "user_id", ids).
		AddUpdate("write_pool_total", writepools, "users.write_pool_total "+operation+" t.amount").Exec(edb).Error
}

func (edb *EventDb) updateUserPayedFees(txns []Transaction) error {
	var ids []string
	var fees []currency.Coin
	for _, t := range txns {
		ids = append(ids, t.ClientId)
		fees = append(fees, t.Fee)
	}

	return CreateBuilder("user", "user_id", ids).
		AddUpdate("payed_fees", fees, "users.payed_fees + t.fee").Exec(edb).Error
}

func mergeUpdateUserCollectedRewardsEvents() *eventsMergerImpl[RewardMint] {
	return newEventsMerger[RewardMint](TagUpdateUserCollectedRewards, withUniqueEventOverwrite())
}

func mergeIncreaseUserTotalStakeEvents() *eventsMergerImpl[DelegatePoolLock] {
	return newEventsMerger[DelegatePoolLock](TagLockStakePool, withUniqueEventOverwrite())
}
func mergeDecreaseUserTotalStakeEvents() *eventsMergerImpl[DelegatePoolLock] {
	return newEventsMerger[DelegatePoolLock](TagUnlockStakePool, withUniqueEventOverwrite())
}

func mergeIncreaseUserReadPoolTotalEvents() *eventsMergerImpl[ReadPoolLock] {
	return newEventsMerger[ReadPoolLock](TagLockReadPool, withUniqueEventOverwrite())
}

func mergeDecreaseUserReadPoolTotalEvents() *eventsMergerImpl[ReadPoolLock] {
	return newEventsMerger[ReadPoolLock](TagUnlockReadPool, withUniqueEventOverwrite())
}

func mergeIncreaseUserWritePoolTotalEvents() *eventsMergerImpl[WritePoolLock] {
	return newEventsMerger[WritePoolLock](TagLockWritePool, withUniqueEventOverwrite())
}

func mergeDecreaseUserWritePoolTotalEvents() *eventsMergerImpl[WritePoolLock] {
	return newEventsMerger[WritePoolLock](TagUnlockWritePool, withUniqueEventOverwrite())
}

func mergeUpdateUserPayedFeesEvents() *eventsMergerImpl[Transaction] {
	return newEventsMerger[Transaction](TagUpdateUserPayedFees, withUniqueEventOverwrite())
}

func mergeAddUsersEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagAddOrOverwriteUser, withUniqueEventOverwrite())
}
