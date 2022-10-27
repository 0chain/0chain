package event

import (
	"time"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	UserID  string        `json:"user_id" gorm:"primarykey"`
	TxnHash string        `json:"txn_hash"`
	Balance currency.Coin `json:"balance"`
	Change  currency.Coin `json:"change"`
	Round   int64         `json:"round"`
	Nonce   int64         `json:"nonce"`
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
		logging.Logger.Debug("event db - upsert users ", zap.Any("duration", time.Since(ts)),
			zap.Int("num", len(users)))
	}()
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"txn_hash", "round", "nonce"}),
	}).Create(&users).Error
}

func mergeAddUsersEvents() *eventsMergerImpl[User] {
	return newEventsMerger[User](TagAddOrOverwriteUser, withUniqueEventOverwrite())
}

func (edb *EventDb) GetUserFromId(userId string) (User, error) {
	user := User{}
	return user, edb.Store.Get().Model(&User{}).Where(User{UserID: userId}).Scan(&user).Error

}
