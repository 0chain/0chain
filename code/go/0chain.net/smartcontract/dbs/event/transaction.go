package event

import (
	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

// Transaction model to save the transaction data
// swagger:model Transaction
type Transaction struct {
	model.ImmutableModel
	Hash              string        `json:"hash" gorm:"uniqueIndex:idx_thash"`
	BlockHash         string        `json:"block_hash" gorm:"index:idx_tblock_hash"`
	Round             int64         `json:"round"`
	Version           string        `json:"version"`
	ClientId          string        `json:"client_id" gorm:"index:idx_tclient_id"`
	ToClientId        string        `json:"to_client_id" gorm:"index:idx_tto_client_id"`
	TransactionData   string        `json:"transaction_data"`
	Value             currency.Coin `json:"value"`
	Signature         string        `json:"signature"`
	CreationDate      int64         `json:"creation_date"  gorm:"index:idx_tcreation_date"`
	Fee               currency.Coin `json:"fee"`
	Nonce             int64         `json:"nonce"`
	TransactionType   int           `json:"transaction_type"`
	TransactionOutput string        `json:"transaction_output"`
	OutputHash        string        `json:"output_hash"`
	Status            int           `json:"status"`
}

func (edb *EventDb) addTransactions(txns []Transaction) error {
	return edb.Store.Get().Create(&txns).Error
}

func mergeAddTransactionsEvents() *eventsMergerImpl[Transaction] {
	return newEventsMerger[Transaction](TagAddTransactions, withUniqueEventOverwrite())
}

// GetTransactionByHash finds the transaction record by hash
func (edb *EventDb) GetTransactionByHash(hash string) (Transaction, error) {
	tr := Transaction{}
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Where(Transaction{Hash: hash}).
		First(&tr)
	return tr, res.Error
}

// GetTransactionByClientId searches for transaction by clientID
func (edb *EventDb) GetTransactionByClientId(clientID string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Where(Transaction{ClientId: clientID}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_date"},
			Desc:   limit.IsDescending,
		}).
		Scan(&tr)
	return tr, res.Error
}

// GetTransactionByToClientId searches for transaction by toClientID
func (edb *EventDb) GetTransactionByToClientId(toClientID string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Where(Transaction{ToClientId: toClientID}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_date"},
			Desc:   limit.IsDescending,
		}).Scan(&tr)
	return tr, res.Error
}

func (edb *EventDb) GetTransactionByBlockHash(blockHash string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Where(Transaction{BlockHash: blockHash}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Scan(&tr)
	return tr, res.Error
}

// GetTransactions finds the transaction
func (edb *EventDb) GetTransactions(limit common.Pagination) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_date"},
			Desc:   limit.IsDescending,
		}).Find(&tr)

	return tr, res.Error
}

// GetTransactionByBlockNumbers finds the transaction record between two block numbers
func (edb *EventDb) GetTransactionByBlockNumbers(blockStart, blockEnd int64, limit common.Pagination) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().
		Model(&Transaction{}).
		Where("round >= ? AND round < ?", blockStart, blockEnd).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "round"},
			Desc:   limit.IsDescending,
		}).
		Find(&tr)
	return tr, res.Error
}

func (edb *EventDb) GetTransactionsForBlocks(blockStart, blockEnd int64) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().
		Model(&Transaction{}).
		Where("round >= ? AND round < ?", blockStart, blockEnd).
		Order("round asc").
		Find(&tr)
	return tr, res.Error
}
