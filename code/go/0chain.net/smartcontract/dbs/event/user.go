package event

import (
	"0chain.net/chaincore/state"
	"errors"
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

func (edb *EventDb) GetUser(id string) (*User, error) {
	var user User
	result := edb.Store.Get().
		Model(&User{}).
		Where(&User{UserID: id}).
		First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving user %v, error %v", id, result.Error)
	}

	return &user, nil
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

func (edb *EventDb) overwriteUser(user User) error {
	result := edb.Store.Get().
		Model(&User{}).
		Where(&User{UserID: user.UserID}).
		Updates(User{
			TransactionID: user.TransactionID,
			Balance:       user.Balance,
		})
	return result.Error
}

func (edb *EventDb) addOrOverwriteUser(user User) error {
	exists, err := user.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteUser(user)
	}

	result := edb.Store.Get().Create(&user)
	return result.Error
}

func (u *User) exists(edb *EventDb) (bool, error) {

	var user User
	result := edb.Get().
		Model(&User{}).
		Where(&User{UserID: u.UserID}).
		Take(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for user %v, error %v",
			u.UserID, result.Error)
	}

	return true, nil
}
