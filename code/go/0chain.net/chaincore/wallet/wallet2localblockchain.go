package wallet

import (
	"context"
	"fmt"
	"math/rand"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

/*Register - register a wallet using the server side api */
func (w *Wallet) Register(ctx context.Context) error {
	c := clientMetadataProvider.Instance().(*client.Client)
	c.PublicKey = w.SignatureScheme.GetPublicKey()
	c.ID = w.ClientID
	_, err := client.PutClient(ctx, c)
	return err
}

var transactionMetadataProvider datastore.EntityMetadata
var clientMetadataProvider datastore.EntityMetadata

/*SetupWallet - setup the wallet package */
func SetupWallet() {
	transactionMetadataProvider = datastore.GetEntityMetadata("txn")
	clientMetadataProvider = datastore.GetEntityMetadata("client")
}

/*CreateRandomSendTransaction - create a transaction */
func (w *Wallet) CreateRandomSendTransaction(toClient string, fee int64) *transaction.Transaction {
	// value := rand.Int63n(100) * 1000000000
	// if value == 0 {
	// 	value = 100000000
	// }
	value := int64(1000000000)
	msg := fmt.Sprintf("0chain zerochain zipcode Europe rightthing Oriental California honest accurate India network %v %v", rand.Int63(), value)
	return w.CreateSendTransaction(toClient, value, msg, fee)
}

/*CreateSendTransaction - create a send transaction */
func (w *Wallet) CreateSendTransaction(toClient string, value int64, msg string, fee int64) *transaction.Transaction {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.TransactionData = msg
	if config.DevConfiguration.IsFeeEnabled {
		txn.Fee = fee
	}
	txn.Sign(w.SignatureScheme)
	return txn
}

/*CreateSendTransaction - create a send transaction */
func (w *Wallet) CreateSCTransaction(toClient string, value int64, msg string, fee int64) *transaction.Transaction {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.TransactionData = msg
	if config.DevConfiguration.IsFeeEnabled {
		txn.Fee = fee
	}
	txn.TransactionType = transaction.TxnTypeSmartContract
	txn.Sign(w.SignatureScheme)
	return txn
}

/*CreateRandomDataTransaction - creat a random data transaction */
func (w *Wallet) CreateRandomDataTransaction(fee int64) *transaction.Transaction {
	msg := fmt.Sprintf("storing some random data - 1234567890 abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ %v", rand.Int63())
	return w.CreateDataTransaction(msg, fee)
}

/*CreateDataTransaction - create a data transaction */
func (w *Wallet) CreateDataTransaction(msg string, fee int64) *transaction.Transaction {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.TransactionData = msg
	txn.TransactionType = transaction.TxnTypeData
	if config.DevConfiguration.IsFeeEnabled {
		txn.Fee = fee
	}
	txn.Sign(w.SignatureScheme)
	return txn
}
