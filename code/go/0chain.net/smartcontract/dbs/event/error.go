package event

import "gorm.io/gorm"

type Error struct {
	gorm.Model
	TransactionID string
	Error         string
}

func (edb *EventDb) addError(err Error) error {
	return edb.Store.Get().Create(&err).Error
}
