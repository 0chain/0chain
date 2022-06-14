package event

import (
	"0chain.net/chaincore/currency"
	"gorm.io/gorm"
)

// Transaction model to save the transaction data
// swagger:model Transaction
type Transaction struct {
	gorm.Model
	Hash              string `gorm:"uniqueIndex"`
	BlockHash         string
	Version           string
	ClientId          string
	ToClientId        string
	TransactionData   string
	Value             currency.Coin
	Signature         string
	CreationDate      int64
	Fee               currency.Coin
	TransactionType   int
	TransactionOutput string
	OutputHash        string
	Status            int

	ReadMarkers []ReadMarker `gorm:"foreignKey:TransactionID;references:Hash"`
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
func (edb *EventDb) GetTransactionByClientId(clientID string, offset, limit int) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().Model(Transaction{}).Where(Transaction{ClientId: clientID}).Offset(offset).Limit(limit).Scan(&tr)
	return tr, res.Error
}

func (edb *EventDb) GetTransactionByBlockHash(blockHash string, offset, limit int) ([]Transaction, error) {
	tr := []Transaction{}
	res := edb.Store.Get().Model(Transaction{}).Where(Transaction{BlockHash: blockHash}).Offset(offset).Limit(limit).Scan(&tr)
	return tr, res.Error
}
