package event

import (
	"errors"

	"0chain.net/chaincore/state"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	// Dependency: PR #759. (Default name of the forign key is `transactionID` unless struct name is changed.)
	TransactionID string `json:"TransactionID" gorm:"transactionID"`

	UserID  string        `json:"UserID" gorm:"userID"`
	Balance state.Balance `json:"Balance" gorm:"balance"`
	// Dependency: PR #765. (Change int64 to Nounce type.)
	Nonce int64 `json:"Nonce" gorm:"nonce"`
}

func (edb *EventDb) GetUser(userID string) (User, error) {
	var user User
	if edb.Store == nil {
		return user, errors.New("event database is nil")
	}
	result := edb.Store.Get().Where("userID = ?", userID).Find(&user)
	return user, result.Error
}
