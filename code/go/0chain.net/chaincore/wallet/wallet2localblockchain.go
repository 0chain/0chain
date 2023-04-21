package wallet

import (
	"fmt"
	"math/rand"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/currency"
)

var transactionMetadataProvider datastore.EntityMetadata

// var clientMetadataProvider datastore.EntityMetadata

/*SetupWallet - setup the wallet package */
func SetupWallet() {
	transactionMetadataProvider = datastore.GetEntityMetadata("txn")
}

/*CreateRandomSendTransaction - create a transaction */
func (w *Wallet) CreateRandomSendTransaction(toClient string, value currency.Coin, estimateFeeFunc func(t *transaction.Transaction) currency.Coin) *transaction.Transaction {
	msg := fmt.Sprintf("0chain zerochain zipcode Europe rightthing Oriental California honest accurate India network %v %v", rand.Int63(), value)
	return w.CreateSendTransaction(toClient, value, msg, estimateFeeFunc)
}

/*CreateSendTransaction - create a send transaction */
func (w *Wallet) CreateSendTransaction(toClient string, value currency.Coin, msg string,
	estimateFeeFunc func(t *transaction.Transaction) currency.Coin) *transaction.Transaction {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.PublicKey = w.PublicKey
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.TransactionData = msg

	isFeeEnabled := config.Configuration().ChainConfig.IsFeeEnabled()
	if isFeeEnabled {
		txn.Fee = estimateFeeFunc(txn)
	}
	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		panic(err)
	}
	return txn
}

// CreateSCTransaction creates a faucet refill transaction
func (w *Wallet) CreateSCTransaction(toClient string, value currency.Coin, msg string,
	estimateFeeFunc func(txn *transaction.Transaction) currency.Coin) (*transaction.Transaction, error) {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.PublicKey = w.PublicKey
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.TransactionData = msg
	txn.TransactionType = transaction.TxnTypeSmartContract
	if err := txn.ComputeProperties(); err != nil {
		return nil, err
	}

	isFeeEnabled := config.Configuration().ChainConfig.IsFeeEnabled()
	if isFeeEnabled {
		txn.Fee = estimateFeeFunc(txn)
	}
	txn.TransactionType = transaction.TxnTypeSmartContract
	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		return nil, err
	}
	return txn, nil
}

/*CreateRandomDataTransaction - creat a random data transaction */
func (w *Wallet) CreateRandomDataTransaction(fee currency.Coin) *transaction.Transaction {
	msg := fmt.Sprintf("storing some random data - 1234567890 abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ %v", rand.Int63())
	return w.CreateDataTransaction(msg, fee)
}

/*CreateDataTransaction - create a data transaction */
func (w *Wallet) CreateDataTransaction(msg string, fee currency.Coin) *transaction.Transaction {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.TransactionData = msg
	txn.TransactionType = transaction.TxnTypeData

	isFeeEnabled := config.Configuration().ChainConfig.IsFeeEnabled()
	if isFeeEnabled {
		txn.Fee = fee
	}
	if _, err := txn.Sign(w.SignatureScheme); err != nil {
		panic(err)
	}
	return txn
}
