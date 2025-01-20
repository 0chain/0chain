package event

import (
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	model.UpdatableModel
	UserID    string        `json:"user_id" gorm:"uniqueIndex"`
	TxnHash   string        `json:"txn_hash"`
	Balance   currency.Coin `json:"balance"`
	Round     int64         `json:"round"`
	Nonce     int64         `json:"nonce"`
	MintNonce int64         `json:"mint_nonce"`
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
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"txn_hash", "round", "balance", "nonce"}),
	}).Create(&users).Error
}

// update or create users
func (edb *EventDb) updateUserMintNonce(users []User) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"mint_nonce": gorm.Expr("GREATEST(users.mint_nonce, EXCLUDED.mint_nonce)"),
		}),
	}).Create(&users).Error
}

func mergeUpdateUserCollectedRewardsEvents() *eventsMergerImpl[UserAggregate] {
	return newEventsMerger[UserAggregate](TagUpdateUserCollectedRewards, withCollectedRewardsMerged())
}

func withCollectedRewardsMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *UserAggregate) (*UserAggregate, error) {
		a.CollectedReward += b.CollectedReward
		return a, nil
	})
}

func mergeUserStakeEvents() *eventsMergerImpl[DelegatePoolLock] {
	return newEventsMerger[DelegatePoolLock](TagLockStakePool, withTotalStakeMerged())
}

func mergeUserUnstakeEvents() *eventsMergerImpl[DelegatePoolLock] {
	return newEventsMerger[DelegatePoolLock](TagUnlockStakePool, withTotalStakeMerged())
}

func withTotalStakeMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *DelegatePoolLock) (*DelegatePoolLock, error) {
		a.Amount += b.Amount
		return a, nil
	})
}

func mergeUserReadPoolLockEvents() *eventsMergerImpl[ReadPoolLock] {
	return newEventsMerger[ReadPoolLock](TagLockReadPool, withReadPoolMerged())
}

func mergeUserReadPoolUnlockEvents() *eventsMergerImpl[ReadPoolLock] {
	return newEventsMerger[ReadPoolLock](TagUnlockReadPool, withReadPoolMerged())
}

func withReadPoolMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *ReadPoolLock) (*ReadPoolLock, error) {
		a.Amount += b.Amount
		return a, nil
	})
}

func mergeUserWritePoolLockEvents() *eventsMergerImpl[WritePoolLock] {
	return newEventsMerger[WritePoolLock](TagLockWritePool, withWritePoolMerged())
}

func mergeUserWritePoolUnlockEvents() *eventsMergerImpl[WritePoolLock] {
	return newEventsMerger[WritePoolLock](TagUnlockWritePool, withWritePoolMerged())
}

func withWritePoolMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *WritePoolLock) (*WritePoolLock, error) {
		a.Amount += b.Amount
		return a, nil
	})
}

func mergeUpdateUserPayedFeesEvents() *eventsMergerImpl[UserAggregate] {
	return newEventsMerger[UserAggregate](TagUpdateUserPayedFees, withPayedFeesMerged())
}

func withPayedFeesMerged() eventMergeMiddleware {
	return withEventMerge(func(a, b *UserAggregate) (*UserAggregate, error) {
		a.PayedFees += b.PayedFees
		return a, nil
	})
}

func mergeAddUsersEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagAddOrOverwriteUser, withUniqueEventOverwrite())
}
