package event

import (
	"fmt"

	"0chain.net/chaincore/currency"
	"0chain.net/core/util"
	"gorm.io/gorm"
)

type User struct {
	ID      uint          `json:"-" gorm:"primarykey"`
	UserID  string        `json:"user_id" gorm:"uniqueIndex"`
	TxnHash string        `json:"txn"`
	Balance currency.Coin `json:"balance"`
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

func (edb *EventDb) overwriteUser(u User) error {
	return edb.Store.Get().Model(&User{}).
		Where("user_id = ?", u.UserID).
		Updates(map[string]interface{}{
			"txn_hash": u.TxnHash,
			"balance":  u.Balance,
			"round":    u.Round,
			"nonce":    u.Nonce,
		}).Error
}

func (edb *EventDb) addOrOverwriteUser(u User) error {
	exists, err := u.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteUser(u)
	}

	result := edb.Store.Get().Create(&u)
	return result.Error
}

func (edb *EventDb) GetUserFromId(userId string) (User, error) {
	user := User{}
	return user, edb.Store.Get().Model(&User{}).Where(User{UserID: userId}).Scan(&user).Error

}

func (edb *EventDb) CreateUser(usr *User) error {
	return edb.Store.Get().Create(usr).Error
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
