package event

import (
	"strings"
	"time"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

// Transaction model to save the transaction data
// swagger:model Transaction
type Transaction struct {
	model.ImmutableModel
	Hash              string        `json:"hash" gorm:"uniqueIndex:idx_thash;index:idx_tround_thash, priority:2"`
	BlockHash         string        `json:"block_hash" gorm:"index:idx_tblock_hash"`
	Round             int64         `json:"round" gorm:"index:idx_tround;index:idx_tround_thash, priority:1"`
	Version           string        `json:"version"`
	ClientId          string        `json:"client_id" gorm:"index:idx_tclient_id"`
	ToClientId        string        `json:"to_client_id" gorm:"index:idx_tto_client_id"`
	TransactionData   string        `json:"transaction_data"`
	Value             currency.Coin `json:"value"`
	Signature         string        `json:"signature"`
	CreationDate      int64         `json:"creation_date"`
	Fee               currency.Coin `json:"fee"`
	Nonce             int64         `json:"nonce"`
	TransactionType   int           `json:"transaction_type"`
	TransactionOutput string        `json:"transaction_output"`
	OutputHash        string        `json:"output_hash"`
	Status            int           `json:"status"`
}

type TransactionErrors struct {
	TransactionOutput string `json:"transaction_output"`
	Count             int    `json:"count"`
}

func (edb *EventDb) addTransactions(txns []Transaction) error {
	return edb.Store.Get().Create(&txns).Error
}

func mergeAddTransactionsEvents() *eventsMergerImpl[Transaction] {
	return newEventsMerger[Transaction](TagAddTransactions, withUniqueEventOverwrite())
}

// GetTransactionByHash finds the transaction record by hash
// Used Index: idx_thash
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
// Used Index: idx_tclient_id
func (edb *EventDb) GetTransactionByClientId(clientID string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Joins("JOIN blocks ON blocks.hash = transactions.block_hash").
		Where("blocks.is_finalised = ?", true).
		Where(Transaction{ClientId: clientID}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.round"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.hash"},
			Desc:   limit.IsDescending,
		}).
		Scan(&tr)
	return tr, res.Error
}

// GetTransactionByToClientId searches for transaction by toClientID
// Used Index: idx_tto_client_id
func (edb *EventDb) GetTransactionByToClientId(toClientID string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Joins("JOIN blocks ON blocks.hash = transactions.block_hash").
		Where("blocks.is_finalised = ?", true).
		Where(Transaction{ToClientId: toClientID}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.round"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.hash"},
			Desc:   limit.IsDescending,
		}).
		Scan(&tr)
	return tr, res.Error
}

// GetTransactionByBlockHash finds the transaction record by block hash
// Used Index: idx_tblock_hash
func (edb *EventDb) GetTransactionByBlockHash(blockHash string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Joins("JOIN blocks ON blocks.hash = transactions.block_hash").
		Where("blocks.is_finalised = ?", true).
		Where(Transaction{BlockHash: blockHash}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.round"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.hash"},
			Desc:   limit.IsDescending,
		}).
		Scan(&tr)
	return tr, res.Error
}

// GetTransactions finds the transaction
func (edb *EventDb) GetTransactions(limit common.Pagination) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.
		Get().
		Model(&Transaction{}).
		Joins("JOIN blocks ON blocks.hash = transactions.block_hash").
		Where("blocks.is_finalised = ?", true).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.round"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.hash"},
			Desc:   limit.IsDescending,
		}).
		Find(&tr)

	return tr, res.Error
}

// GetTransactionByBlockNumbers finds the transaction record between two block numbers
func (edb *EventDb) GetTransactionByBlockNumbers(blockStart, blockEnd int64, limit common.Pagination) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().
		Model(&Transaction{}).
		Joins("JOIN blocks ON blocks.hash = transactions.block_hash").
		Where("blocks.is_finalised = ?", true).
		Where("round >= ? AND round < ?", blockStart, blockEnd).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.round"},
			Desc:   limit.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "transactions.hash"},
			Desc:   limit.IsDescending,
		}).
		Find(&tr)
	return tr, res.Error
}

// GetTransactionsForBlocks finds the transaction record between two block numbers
func (edb *EventDb) GetTransactionsForBlocks(blockStart, blockEnd int64) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().
		Model(&Transaction{}).
		Joins("JOIN blocks ON blocks.hash = transactions.block_hash").
		Where("blocks.is_finalised = ?", true).
		Where("round >= ? AND round < ?", blockStart, blockEnd).
		Order("round asc").
		Order("hash desc").
		Find(&tr)
	return tr, res.Error
}

func (edb *EventDb) UpdateTransactionErrors() error {
	db := edb.Get()

	// created_at for last day from now
	lastDay := time.Now().AddDate(0, 0, -1)
	// convert to string
	lastDayString := lastDay.Format("2006-01-02 15:04:05")

	// clean up the transaction error table
	err := db.Exec("TRUNCATE TABLE transaction_errors").Error
	if err != nil {
		return err
	}

	if dbTxn := db.Exec("INSERT INTO transaction_errors (transaction_output, count) "+
		"SELECT transaction_output, count(*) as count FROM transactions WHERE status = ? and created_at > ?"+
		"GROUP BY transaction_output", 2, lastDayString); dbTxn.Error != nil {

		logging.Logger.Error("Error while inserting transactions in transaction error table", zap.Any("error", dbTxn.Error))
		return dbTxn.Error
	}

	return nil
}

func (edb *EventDb) GetTransactionErrors() (map[string][]TransactionErrors, error) {
	var txnErrors []TransactionErrors

	err := edb.Get().Model(&TransactionErrors{}).Find(&txnErrors).Order("count desc")

	if err.Error != nil {
		return nil, err.Error
	}

	transactionErrors := categorizeOnSubstring(txnErrors)

	return transactionErrors, nil
}

func categorizeOnSubstring(input []TransactionErrors) map[string][]TransactionErrors {
	categorized := make(map[string][]TransactionErrors)

	for _, err := range input {
		// Find the index of the first colon in the transaction output
		colonIndex := strings.Index(err.TransactionOutput, ":")

		if colonIndex != -1 {
			// Extract the substring before the first colon
			category := err.TransactionOutput[:colonIndex]

			// Append the error to the corresponding category in the map
			categorized[category] = append(categorized[category], err)
		} else {
			categorized[err.TransactionOutput] = append(categorized[err.TransactionOutput], err)
		}
	}

	return categorized
}
