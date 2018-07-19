package wallet

import (
	"encoding/hex"
	"fmt"
	"math/rand"

	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/transaction"
)

/*Wallet - a struct representing the client's wallet */
type Wallet struct {
	PublicKey       string
	PrivateKey      string
	PrivateKeyBytes []byte
	PublicKeyBytes  []byte
	ClientID        string
}

/*Initialize - initialize a wallet with public/private keys */
func (w *Wallet) Initialize() {
	publicKey, privateKey, err := encryption.GenerateKeysBytes()
	if err != nil {
		panic(err)
	}
	w.PublicKeyBytes = publicKey
	w.PrivateKeyBytes = privateKey
	w.PublicKey = hex.EncodeToString(publicKey)
	w.PrivateKey = hex.EncodeToString(privateKey)
	w.ClientID = encryption.Hash(publicKey)
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
	txn.Value = rand.Int63n(100000)
	txn.TransactionData = fmt.Sprintf("0chain zerochain zipcode Europe rightthing Oriental California honest accurate India network %v %v", rand.Int63(), txn.Value)
	txn.Sign(w.PrivateKeyBytes)
	return txn
}
