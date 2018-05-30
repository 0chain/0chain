package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
)

/*Transaction type for capturing the transaction data */
type Transaction struct {
	memorystore.CollectionIDField
	Hash            string           `json:"hash"`
	ClientID        datastore.Key    `json:"client_id"`
	ToClientID      datastore.Key    `json:"to_client_id,omitempty"`
	ChainID         datastore.Key    `json:"chain_id,omitempty"`
	TransactionData string           `json:"transaction_data"`
	Value           int64            `json:"transaction_value"` // The value associated with this transaction
	Signature       string           `json:"signature"`
	CreationDate    common.Timestamp `json:"creation_date"`
	Status          byte             `json:"status"`
	BlockID         datastore.Key    `json:"block_id,omitempty"` // This is the block that finalized this transaction

	Client   *client.Client `json:"-"`
	ToClient *client.Client `json:"-"`
}

const (
	/*TXN_STATUS_FREE - transaction that is yet to be put into a block */
	TXN_STATUS_FREE = 0
	/*TXN_STATUS_PENDING - transaction that is yet being worked on by putting it into the block */
	TXN_STATUS_PENDING = 1
	/*TXN_STATUS_MINED - transaction that is already mined */
	TXN_STATUS_FINALIZED = 2
	/*TXN_STATUS_CANCELLED - the transaction is cancelled via error reporting protocol */
	TXN_STATUS_CANCELLED = 3
)

/*GetEntityName - Entity implementation */
func (t *Transaction) GetEntityName() string {
	return "txn"
}

/*ComputeProperties - Entity implementation */
func (t *Transaction) ComputeProperties() {
	if t.Hash != "" {
		t.ID = datastore.ToKey(t.Hash)
	}
	if datastore.IsEmpty(t.ChainID) {
		t.ChainID = datastore.ToKey(config.GetMainChainID())
	}
}

/*Validate - Entity implementation */
func (t *Transaction) Validate(ctx context.Context) error {
	err := config.ValidChain(datastore.ToString(t.ChainID))
	if err != nil {
		return err
	}
	if t.Hash == "" {
		return common.InvalidRequest("hash required for transaction")
	}
	if t.ID != datastore.ToKey(t.Hash) {
		return common.NewError("id_hash_mismatch", "ID and Hash don't match")
	}

	if t.ChainID != datastore.ToKey(config.GetServerChainID()) {
		return config.ErrSupportedChain
	}

	err = t.VerifyHash(ctx)
	if err == nil {
		err = t.VerifySignature(ctx)
	}
	if err != nil {
		return err
	}
	return nil
}

/*Read - store read */
func (t *Transaction) Read(ctx context.Context, key datastore.Key) error {
	return memorystore.Read(ctx, key, t)
}

/*Write - store read */
func (t *Transaction) Write(ctx context.Context) error {
	return memorystore.Write(ctx, t)
}

/*Delete - store read */
func (t *Transaction) Delete(ctx context.Context) error {
	return memorystore.Delete(ctx, t)
}

var txnEntityCollection *memorystore.EntityCollection

/*GetCollectionName - override to partition by chain id */
func (t *Transaction) GetCollectionName() string {
	return txnEntityCollection.GetCollectionName(t.ChainID)
}

/*GetClient - get the Client object associated with the transaction */
func (t *Transaction) GetClient(ctx context.Context) (*client.Client, error) {
	co := &client.Client{}
	err := co.Read(ctx, t.ClientID)
	if err != nil {
		return nil, err
	}
	t.Client = co
	return co, nil
}

/*ComputeHash - compute the hash from the various components of the transaction */
func (t *Transaction) ComputeHash() string {
	hashdata := fmt.Sprintf("%v:%v:%v:%v", t.ClientID, t.CreationDate, t.Value, t.TransactionData)
	return encryption.Hash(hashdata)
}

/*VerifyHash - Verify the hash of the transaction */
func (t *Transaction) VerifyHash(ctx context.Context) error {
	if t.Hash != t.ComputeHash() {
		return common.NewError("hash_mismatch", fmt.Sprintf("The hash of the data doesn't match with the provided hash"))
	}
	return nil
}

/*VerifySignature - verify the transaction hash */
func (t *Transaction) VerifySignature(ctx context.Context) error { //TODO
	co, err := t.GetClient(ctx)
	if err != nil {
		return err
	}
	correctSignature, err := co.Verify(t.Signature, t.Hash)
	if err != nil {
		return err
	}
	if !correctSignature {
		return errors.New("Not signed correctly")
	}
	/*
		if msg != t.TransactionData {
			return common.NewError("hash_signature_mismatch", "Decrypted signature doesn't match the hash of the transaction")
		} */
	return nil
}

/*Provider - entity provider for client object */
func Provider() interface{} {
	c := &Transaction{}
	c.EntityCollection = txnEntityCollection
	c.Status = TXN_STATUS_FREE
	c.CreationDate = common.Now()
	c.ChainID = datastore.ToKey(config.GetMainChainID())
	return c
}

var TransactionEntityChannel chan memorystore.MemoryEntity

/*SetupEntity - setup the entity */
func SetupEntity() {
	memorystore.RegisterEntityProvider("txn", Provider)
	txnEntityCollection = &memorystore.EntityCollection{CollectionName: "collection.txn", CollectionSize: 10000000, CollectionDuration: time.Hour}

	/*Entity Buffer Size = 10240
	* Timeout = 250 milliseconds
	* Entity Chunk Size = 128
	* Chunk Buffer Size = 32
	* Chunk Workers = 8
	 */
	var collectionOptions = memorystore.CollectionOptions{
		EntityBufferSize: 10240,
		MaxHoldupTime:    250 * time.Millisecond,
		NumChunkCreators: 1,
		ChunkSize:        128,
		ChunkBufferSize:  32,
		NumChunkStorers:  8,
	}
	TransactionEntityChannel = memorystore.SetupWorkers(common.GetRootContext(), &collectionOptions)
}

/*Sign - given a client and client's private key, sign this tranasction */
func (t *Transaction) Sign(client *client.Client, privateKey string) (string, error) {
	t.Hash = t.ComputeHash()
	return encryption.Sign(privateKey, t.Hash)
}

/*GetWeight - get the weight/score of this transction */
func (t *Transaction) GetWeight() float64 {
	// TODO: For now all transactions weigh the same
	return 1.0
}
