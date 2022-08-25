package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Transaction model to save the transaction data
// swagger:model Transaction
type Transaction struct {
	gorm.Model
	Hash              string `gorm:"uniqueIndex:idx_thash"`
	BlockHash         string `gorm:"index:idx_tblock_hash"`
	BlockRound        int64  `gorm:"index:idx_tblock_round"`
	Version           string
	ClientId          string `gorm:"index:idx_tclient_id"`
	ToClientId        string `gorm:"index:idx_tto_client_id"`
	TransactionData   string
	Value             currency.Coin
	Signature         string
	CreationDate      int64 `gorm:"index:idx_tcreation_date"`
	Fee               currency.Coin
	Nonce             int64
	TransactionType   int
	TransactionOutput string
	OutputHash        string
	Status            int

	//ref
	ReadMarkers []ReadMarker  `gorm:"foreignKey:TransactionID;references:Hash"`
	WriteMarker []WriteMarker `gorm:"foreignKey:TransactionID;references:Hash"`
}

func (edb *EventDb) addTransaction(transaction Transaction) error {
	res := edb.Store.Get().Create(&transaction)
	return res.Error
}

// GetTransactionByHash finds the transaction record by hash
func (edb *EventDb) GetTransactionByHash(hash string) (Transaction, error) {
	tr := Transaction{}
	res := edb.Store.Get().Model(Transaction{}).Where(Transaction{Hash: hash}).First(&tr)
	return tr, res.Error
}

// GetTransactionByClientId searches for transaction by clientID
func (edb *EventDb) GetTransactionByClientId(clientID string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.Get().Model(Transaction{}).Where(Transaction{ClientId: clientID}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "creation_date"},
		Desc:   limit.IsDescending,
	}).Scan(&tr)
	return tr, res.Error
}

// GetTransactionByToClientId searches for transaction by toClientID
func (edb *EventDb) GetTransactionByToClientId(toClientID string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.Get().Model(Transaction{}).Where(Transaction{ToClientId: toClientID}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "creation_date"},
		Desc:   limit.IsDescending,
	}).Scan(&tr)
	return tr, res.Error
}

func (edb *EventDb) GetTransactionByBlockHash(blockHash string, limit common.Pagination) ([]Transaction, error) {
	var tr []Transaction
	res := edb.Store.Get().Model(Transaction{}).Where(Transaction{BlockHash: blockHash}).Offset(limit.Offset).Limit(limit.Limit).Scan(&tr)
	return tr, res.Error
}

// GetTransactions finds the transaction
func (edb *EventDb) GetTransactions(limit common.Pagination) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().Model(&Transaction{}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "creation_date"},
		Desc:   limit.IsDescending,
	}).Find(&tr)

	return tr, res.Error
}

// GetTransactionByBlockNumbers finds the transaction record between two block numbers
func (edb *EventDb) GetTransactionByBlockNumbers(blockStart, blockEnd int, limit common.Pagination) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().
		Model(Transaction{}).
		Joins("INNER JOIN blocks on blocks.round >= ? AND blocks.round <= ? AND blocks.hash = transactions.block_hash", blockStart, blockEnd).
		Offset(limit.Limit).
		Limit(limit.Offset).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_date"},
			Desc:   limit.IsDescending,
		}).
		Scan(&tr)

	return tr, res.Error
}
