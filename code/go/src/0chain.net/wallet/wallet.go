package wallet

import (
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/transaction"
)

/*Wallet - a struct representing the client's wallet */
type Wallet struct {
	PublicKey  string
	PrivateKey string
	ClientID   string
}

/*Initialize - initialize a wallet with public/private keys */
func (w *Wallet) Initialize() {
	w.PublicKey, w.PrivateKey = encryption.GenerateKeys()
	w.ClientID = encryption.Hash(w.PublicKey)
}

var transactionMetadataProvider datastore.EntityMetadata

/*SetupWallet - setup the wallet package */
func SetupWallet() {
	transactionMetadataProvider = datastore.GetEntityMetadata("txn")
}

/*CreateTransaction - create a transaction */
func (w *Wallet) CreateTransaction(toClient string) *transaction.Transaction {
	// TODO: once we introduce the state, the transactions should be carefully created ensuring valid state
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.TransactionData = "0chain zerochain zipcode Europe rightthing Oriental California honest accurate India network"
	txn.Value = 10
	txn.Sign(w.PrivateKey)
	return txn
}
