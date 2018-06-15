package transaction

import (
	"context"
	"fmt"
	"time"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
)

func init() {
	//memorystore.AddPool("txndb", memorystore.DefaultPool) //TODO: This is temporary
	memorystore.AddPool("txndb", memorystore.NewPool("redis_txns", 6479))
}

/*Transaction type for capturing the transaction data */
type Transaction struct {
	datastore.HashIDField
	datastore.CollectionMemberField
	datastore.VersionField

	ClientID  datastore.Key `json:"client_id" msgpack:"cid,omitempty"`
	PublicKey string        `json:"-" msgpack:"puk,omitempty"`

	ToClientID      datastore.Key    `json:"to_client_id,omitempty" msgpack:"tcid,omitempty"`
	ChainID         datastore.Key    `json:"chain_id,omitempty" msgpack:"chid"`
	TransactionData string           `json:"transaction_data" msgpack:"d"`
	Value           int64            `json:"transaction_value" msgpack:"v"` // The value associated with this transaction
	Signature       string           `json:"signature" msgpack:"s"`
	CreationDate    common.Timestamp `json:"creation_date" msgpack:"ts"`
	Status          byte             `json:"status" msgpack:"st"`
	BlockID         datastore.Key    `json:"block_id,omitempty" msgpack:"bid"` // This is the block that finalized this transaction

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

var transactionEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (t *Transaction) GetEntityMetadata() datastore.EntityMetadata {
	return transactionEntityMetadata
}

/*ComputeProperties - Entity implementation */
func (t *Transaction) ComputeProperties() {
	if datastore.IsEmpty(t.ChainID) {
		t.ChainID = datastore.ToKey(config.GetMainChainID())
	}
	if t.PublicKey != "" {
		if t.ClientID == "" {
			// Doing this is OK because the transaction signature has ClientID
			// that won't pass verification if some other client's public is put in
			t.ClientID = encryption.Hash(t.PublicKey)
		}
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

	err = t.VerifyHash(ctx)
	if err != nil {
		return err
	}

	err = t.VerifySignature(ctx)
	if err != nil {
		return err
	}
	return nil
}

/*Read - store read */
func (t *Transaction) Read(ctx context.Context, key datastore.Key) error {
	return t.GetEntityMetadata().GetStore().Read(ctx, key, t)
}

/*Write - store read */
func (t *Transaction) Write(ctx context.Context) error {
	return t.GetEntityMetadata().GetStore().Write(ctx, t)
}

/*Delete - store read */
func (t *Transaction) Delete(ctx context.Context) error {
	return t.GetEntityMetadata().GetStore().Delete(ctx, t)
}

var txnEntityCollection *datastore.EntityCollection

/*GetCollectionName - override to partition by chain id */
func (t *Transaction) GetCollectionName() string {
	return txnEntityCollection.GetCollectionName(t.ChainID)
}

/*GetHash - return the hash of the transaction */
func (t *Transaction) GetHash() string {
	return t.Hash
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
func (t *Transaction) VerifySignature(ctx context.Context) error {
	var err error
	co := datastore.GetEntityMetadata("client").Instance().(*client.Client)
	if t.PublicKey == "" {
		co, err = t.GetClient(ctx)
		if err != nil {
			return err
		}
	} else {
		co.ID = t.ClientID
		co.PublicKey = t.PublicKey
		t.PublicKey = ""
	}
	correctSignature, err := co.Verify(t.Signature, t.Hash)
	if err != nil {
		return err
	}
	if !correctSignature {
		return common.NewError("invalid_signature", "Invalid Signature")
	}
	return nil
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	c := &Transaction{}
	c.Version = "1.0"
	c.EntityCollection = txnEntityCollection
	c.Status = TXN_STATUS_FREE
	c.CreationDate = common.Now()
	c.ChainID = datastore.ToKey(config.GetMainChainID())
	return c
}

var TransactionEntityChannel chan datastore.QueuedEntity

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	transactionEntityMetadata = datastore.MetadataProvider()
	transactionEntityMetadata.Name = "txn"
	transactionEntityMetadata.MemoryDB = "txndb"
	transactionEntityMetadata.Provider = Provider
	transactionEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("txn", transactionEntityMetadata)
	txnEntityCollection = &datastore.EntityCollection{CollectionName: "collection.txn", CollectionSize: 60000000, CollectionDuration: time.Hour}

	/*Entity Buffer Size = 10240
	* Timeout = 250 milliseconds
	* Entity Chunk Size = 128
	* Chunk Buffer Size = 32
	* Chunk Workers = 8
	 */
	var chunkingOptions = datastore.ChunkingOptions{
		EntityMetadata:   transactionEntityMetadata,
		EntityBufferSize: 10240,
		MaxHoldupTime:    250 * time.Millisecond,
		NumChunkCreators: 1,
		ChunkSize:        128,
		ChunkBufferSize:  64,
		NumChunkStorers:  16,
	}
	TransactionEntityChannel = memorystore.SetupWorkers(common.GetRootContext(), &chunkingOptions)
}

/*Sign - given a client and client's private key, sign this tranasction */
func (t *Transaction) Sign(client *client.Client, privateKey string) (string, error) {
	t.Hash = t.ComputeHash()
	signature, err := encryption.Sign(privateKey, t.Hash)
	if err != nil {
		return signature, err
	}
	t.Signature = signature
	return signature, nil
}
