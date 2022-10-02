package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	UserID  string        `json:"user_id" gorm:"primarykey"`
	TxnHash string        `json:"txn"`
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

func (edb *EventDb) addOrOverwriteUser(u User) error {
	result := edb.Store.Get().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"txn_hash": u.TxnHash,
			"balance":  u.Balance,
			"round":    u.Round,
			"nonce":    u.Nonce,
		}),
	}).Create(&u)

	return result.Error
}

func (edb *EventDb) GetUserFromId(userId string) (User, error) {
	user := User{}
	return user, edb.Store.Get().Model(&User{}).Where(User{UserID: userId}).Scan(&user).Error

}
