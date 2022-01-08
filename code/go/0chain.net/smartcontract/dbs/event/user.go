package event

import (
	"0chain.net/chaincore/state"
	"fmt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TransactionID string        `json:"transaction_id"`
	Balance       state.Balance `json:"balance"`
	UserID        string        `json:"user_id"`

	// Non sql fields
	Amount state.Balance `gorm:"-"`
}

func (edb *EventDb) updateIncreaseUserBalanceByAmount(u User) error {
	var user = User{UserID: u.UserID}

	exists, err := user.exists(edb)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("user %v not in database. Cannot update", u.ID)
	}

	updates := map[string]interface{}{
		"balance":        gorm.Expr("balance + ?", u.Amount),
		"transaction_id": u.TransactionID,
	}

	result := edb.Store.Get().
		Model(&User{}).
		Where(&User{UserID: u.UserID}).
		Updates(updates)
	return result.Error
}

func (edb *EventDb) updateDecreaseUserBalanceByAmount(u User) error {
	var user = User{UserID: u.UserID}

	exists, err := user.exists(edb)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("user %v not in database. Cannot update", u.ID)
	}

	updates := map[string]interface{}{
		"balance":        gorm.Expr("balance - ?", u.Amount),
		"transaction_id": u.TransactionID,
	}

	result := edb.Store.Get().
		Model(&User{}).
		Where(&User{UserID: u.UserID}).
		Updates(updates)
	return result.Error
}

func (u *User) exists(edb *EventDb) (bool, error) {

	var user User
	result := edb.Get().
		Model(&User{}).
		Where(&User{UserID: u.UserID}).
		Take(&user)

	if result.Error != nil {
		return false, fmt.Errorf("error searching for user %v, error %v",
			u.UserID, result.Error)
	}

	return true, nil
}
