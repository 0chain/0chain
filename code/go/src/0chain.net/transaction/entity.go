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
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"go.uber.org/zap"
)

/*TXN_TIME_TOLERANCE - the txn creation date should be within 5 seconds before/after of current time */
const TXN_TIME_TOLERANCE = 10

var TransactionCount = 0

func SetupTransactionDB() {
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

	TransactionType int `json:"transaction_type" msgpack:"tt"`

	// a parent transaction introdcues certain state and this state is managed as new transactions are created referencing to this parent transaction
	ParentTransactionHash datastore.Key `json:"parent_txn_hash,omitempty" msgpack:"ptid,omitempty"`
}

var transactionEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (t *Transaction) GetEntityMetadata() datastore.EntityMetadata {
	return transactionEntityMetadata
}

/*ComputeProperties - Entity implementation */
func (t *Transaction) ComputeProperties() {
	t.EntityCollection = txnEntityCollection
	if datastore.IsEmpty(t.ChainID) {
		t.ChainID = datastore.ToKey(config.GetServerChainID())
	}
	if t.PublicKey != "" {
		if t.ClientID == "" {
			// Doing this is OK because the transaction signature has ClientID
			// that won't pass verification if some other client's public is put in
			t.ClientID = encryption.Hash(t.PublicKey)
		}
	} else {
		if t.ClientID == "" {
			Logger.Error("invalid transaction", zap.String("txn", datastore.ToJSON(t).String()))
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
	if !common.Within(int64(t.CreationDate), TXN_TIME_TOLERANCE) {
		return common.InvalidRequest("Transaction creation time not within tolerance")
	}
	err = t.VerifyHash(ctx)
	if err != nil {
		return err
	}

	err = t.VerifySignature(ctx)
	if err != nil {
		return err
	}
	if t.TransactionType%2 == 1 {
		if t.ParentTransactionHash == "" {
			return common.InvalidRequest("Transaction type requires a parent transaction")
		}
	} else {
		//Do we want to support multi-level transactions?
		if t.ParentTransactionHash != "" {
			return common.InvalidRequest("a parent transaction can't have a parent transaction")
		}
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
	return co, nil
}

/*HashData - data used to hash the transaction */
func (t *Transaction) HashData() string {
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", t.CreationDate, t.ClientID, t.ToClientID, t.Value, t.TransactionData)
	return hashdata
}

/*ComputeHash - compute the hash from the various components of the transaction */
func (t *Transaction) ComputeHash() string {
	return encryption.Hash(t.HashData())
}

/*VerifyHash - Verify the hash of the transaction */
func (t *Transaction) VerifyHash(ctx context.Context) error {
	if t.Hash != t.ComputeHash() {
		Logger.Debug("verify hash (hash mismatch)", zap.String("hash", t.Hash), zap.String("computed_hash", t.ComputeHash()), zap.String("hash_data", t.HashData()), zap.String("txn", datastore.ToJSON(t).String()))
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
	co.PublicKeyBytes.SetBytesFromString(co.PublicKey)
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
	t := &Transaction{}
	t.Version = "1.0"
	t.CreationDate = common.Now()
	t.ChainID = datastore.ToKey(config.GetServerChainID())
	t.EntityCollection = txnEntityCollection
	return t
}

var TransactionEntityChannel chan datastore.QueuedEntity

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	transactionEntityMetadata = datastore.MetadataProvider()
	transactionEntityMetadata.Name = "txn"
	transactionEntityMetadata.DB = "txndb"
	transactionEntityMetadata.Provider = Provider
	transactionEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("txn", transactionEntityMetadata)
	txnEntityCollection = &datastore.EntityCollection{CollectionName: "collection.txn", CollectionSize: 60000000, CollectionDuration: time.Hour}

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
func (t *Transaction) Sign(privateKey string) (string, error) {
	t.Hash = t.ComputeHash()
	signature, err := encryption.Sign(privateKey, t.Hash)
	if err != nil {
		return signature, err
	}
	t.Signature = signature
	return signature, nil
}

/*GetSummary - get the transaction summary */
func (t *Transaction) GetSummary() *TransactionSummary {
	summary := datastore.GetEntityMetadata("txn_summary").Instance().(*TransactionSummary)
	summary.Hash = t.Hash
	summary.CreationDate = t.CreationDate
	return summary
}

/*GenerateChildTransactions - generate child transactions for this parent transaction */
func (t *Transaction) GenerateChildTransactions(ctx context.Context) []*Transaction {
	//TODO: putting a transaction into a block may need to accompany associated child transactions
	// E.g: When a storage allocation lockin transaction is processed, it may need to add additional child transactions to the block one for each blobber
	return nil
}
