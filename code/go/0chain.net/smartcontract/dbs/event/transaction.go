package event

import (
	"gorm.io/gorm"
)

// Transaction model to save the transaction data
type Transaction struct {
	gorm.Model
	Hash              string
	BlockHash         string
	Version           string
	ClientId          string
	ToClientId        string
	TransactionData   string
	Value             int64
	Signature         string
	CreationDate      int64
	Fee               int64
	TransactionType   int
	TransactionOutput string
	OutputHash        string
	Status            int
}

func (edb *EventDb) addTransaction(transaction Transaction) error {
	res := edb.Store.Get().Create(&transaction)
	return res.Error
}

// GetTransactionByHash finds the transaction record by hash
func (edb *EventDb) GetTransactionByHash(hash string) (Transaction, error) {
	tr := Transaction{}
	res := edb.Store.Get().Table("transactions").Where("hash = ?", hash).First(&tr)
	return tr, res.Error
}
