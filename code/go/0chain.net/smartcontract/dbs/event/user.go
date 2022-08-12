package event

import (
	"fmt"

	"0chain.net/chaincore/currency"
	"0chain.net/core/util"
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

func (u *User) exists(edb *EventDb) (bool, error) {
	var user User
	err := edb.Store.Get().Model(&User{}).
		Where("user_id = ?", u.UserID).
		Take(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user's existence %v,"+
			" error %v", user, err)
	}

	return true, nil
}
