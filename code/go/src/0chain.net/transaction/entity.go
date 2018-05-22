package transaction

import (
	"context"
	"time"

	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/encryption"
)

/*Transaction type for capturing the transaction data */
type Transaction struct {
	datastore.CollectionIDField
	Hash            string      `json:"hash"`
	ClientID        string      `json:"client_id"`
	ToClientID      string      `json:"to_client_id,omitempty"`
	ChainID         string      `json:"chain_id,omitempty"`
	TransactionData string      `json:"transaction_data"`
	Signature       string      `json:"signature"`
	CreationDate    common.Time `json:"creation_date"`
	Status          byte        `json:"status"`

	Client   *client.Client `json:"-"`
	ToClient *client.Client `json:"-"`
}

const (
	/*TXN_STATUS_FREE - transaction that is yet to be put into a block */
	TXN_STATUS_FREE = 0
	/*TXN_STATUS_PENDING - transaction that is yet being worked on by putting it into the block */
	TXN_STATUS_PENDING = 1
	/*TXN_STATUS_MINED - transaction that is already mined */
	TXN_STATUS_MINED = 2
	/*TXN_STATUS_CANCELLED - the transaction is cancelled via error reporting protocol */
	TXN_STATUS_CANCELLED = 3
)

/*GetEntityName - Entity implementation */
func (t *Transaction) GetEntityName() string {
	return "txn"
}

/*Validate - Entity implementation */
func (t *Transaction) Validate(ctx context.Context) error {
	err := chain.ValidChain(t.ChainID)
	if err != nil {
		return err
	}
	if t.ID == "" {
		if t.Hash == "" {
			return common.InvalidRequest("hash required for transaction")
		}
		t.ID = t.Hash
	}
	if t.ID != t.Hash {
		return common.NewError("id_hash_mismatch", "ID and Hash don't match")
	}

	err = t.VerifySignature(ctx)
	if err != nil {
		return err
	}
	return nil
}

/*ComputeProperties - Entity implementation */
func (t *Transaction) ComputeProperties() {
	if t.Hash != "" {
		t.ID = t.Hash
	}
}

/*Read - datastore read */
func (t *Transaction) Read(ctx context.Context, key string) error {
	return datastore.Read(ctx, key, t)
}

/*Write - datastore read */
func (t *Transaction) Write(ctx context.Context) error {
	return datastore.Write(ctx, t)
}

/*Delete - datastore read */
func (t *Transaction) Delete(ctx context.Context) error {
	return datastore.Delete(ctx, t)
}

var txnEntityCollection = &datastore.EntityCollection{CollectionName: "collection.txn", CollectionSize: 10000000, CollectionDuration: time.Hour}

/*GetCollectionName - override to partition by chain id */
func (t *Transaction) GetCollectionName() string {
	return txnEntityCollection.GetCollectionName(t.ChainID)
}

/*GetClient - get the Client object associated with the transaction */
func (t *Transaction) GetClient(ctx context.Context) (*client.Client, error) {
	co := &client.Client{}
	err := datastore.Read(ctx, t.ClientID, co)
	if err != nil {
		return nil, err
	}
	t.Client = co
	return co, nil
}

/*VerifySignature - verify the transaction hash */
func (t *Transaction) VerifySignature(ctx context.Context) error { //TODO
	co, err := t.GetClient(ctx)
	if err != nil {
		return err
	}
	_, err = co.Verify(t.Signature, t.Hash)
	if err != nil {
		return err
	}
	/*
		if msg != t.TransactionData {
			return common.NewError("hash_signature_mismatch", "Decrypted signature doesn't match the hash of the transaction")
		} */
	return nil
}

/*TransactionProvider - entity provider for client object */
func TransactionProvider() interface{} {
	c := &Transaction{}
	c.EntityCollection = txnEntityCollection
	c.Status = TXN_STATUS_FREE
	return c
}

/*Entity Buffer Size = 10240
* Timeout = 250 milliseconds
* Entity Chunk Size = 128
* Chunk Buffer Size = 32
* Chunk Workers = 8
 */
var TransactionEntityChannel = datastore.SetupWorkers(10240, 250*time.Millisecond, 128, 32, 8)

/*Sign - given a client and client's private key, sign this tranasction */
func (t *Transaction) Sign(client *client.Client, privateKey string) (string, error) {
	// TODO: The actual hash could be based on a combination of things (client_id, creation_date and txn data)
	t.Hash = encryption.Hash(t.TransactionData)
	return encryption.Sign(privateKey, t.Hash)
}
