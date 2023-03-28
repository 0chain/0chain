package event

import (
	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"gorm.io/gorm/clause"
)

// swagger:model Error
type Error struct {
	model.ImmutableModel
	TransactionID string
	Error         string
}

func (edb *EventDb) addError(err Error) error {
	return edb.Store.Get().Create(&err).Error
}

func (edb *EventDb) GetErrorByTransactionHash(transactionID string, limit common.Pagination) ([]Error, error) {
	var transactionErrors []Error
	return transactionErrors, edb.Store.Get().Model(&Error{}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Where(Error{TransactionID: transactionID}).Find(&transactionErrors).Error
}
