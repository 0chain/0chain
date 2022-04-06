package event

import "gorm.io/gorm"

// swagger:model Error
type Error struct {
	gorm.Model
	TransactionID string
	Error         string
}

func (edb *EventDb) addError(err Error) error {
	return edb.Store.Get().Create(&err).Error
}

func (edb *EventDb) GetErrorByTransactionHash(transactionID string) ([]Error, error) {
	var transactionErrors []Error
	return transactionErrors, edb.Store.Get().Model(&Error{}).Where(Error{TransactionID: transactionID}).Find(&transactionErrors).Error
}
