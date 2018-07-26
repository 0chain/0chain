package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"

	"0chain.net/client"
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
	w.ClientID = encryption.Hash(w.PublicKeyBytes)
}

/*SetKeys - sets the keys for the wallet */
func (w *Wallet) SetKeys(publicKey string, privateKey string) error {
	key, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	w.PublicKeyBytes = key
	w.PublicKey = publicKey
	key, err = hex.DecodeString(privateKey)
	if err != nil {
		return err
	}
	w.PrivateKeyBytes = key
	w.PrivateKey = privateKey
	w.ClientID = encryption.Hash(w.PublicKeyBytes)
	return nil
}

/*Register - register a wallet using the server side api */
func (w *Wallet) Register(ctx context.Context) error {
	c := clientMetadataProvider.Instance().(*client.Client)
	c.PublicKey = w.PublicKey
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

/*CreateTransaction - create a transaction */
func (w *Wallet) CreateTransaction(toClient string) *transaction.Transaction {
	value := (1 + rand.Int63n(10)) * 10000000000
	msg := fmt.Sprintf("0chain zerochain zipcode Europe rightthing Oriental California honest accurate India network %v %v", rand.Int63(), value)
	return w.CreateSendTransaction(toClient, value, msg)
}

/*CreateSendTransaction - create a send transaction */
func (w *Wallet) CreateSendTransaction(toClient string, value int64, msg string) *transaction.Transaction {
	txn := transactionMetadataProvider.Instance().(*transaction.Transaction)
	txn.ClientID = w.ClientID
	txn.ToClientID = toClient
	txn.Value = value
	txn.TransactionData = msg
	txn.Sign(w.PrivateKeyBytes)
	return txn
}
