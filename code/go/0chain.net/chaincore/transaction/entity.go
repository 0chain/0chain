package transaction

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

/*TXN_TIME_TOLERANCE - the txn creation date should be within these many seconds before/after of current time */
var TXN_TIME_TOLERANCE int64

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
	Fee             int64            `json:"transaction_fee" msgpack:"f"`

	TransactionType   int    `json:"transaction_type" msgpack:"tt"`
	TransactionOutput string `json:"transaction_output,omitempty" msgpack:"o,omitempty"`
	OutputHash        string `json:"txn_output_hash" msgpack:"oh"`
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
	t.ComputeClientID()
}

/*ComputeClientID - compute the client id if there is a public key in the transaction */
func (t *Transaction) ComputeClientID() {
	if t.PublicKey != "" {
		if t.ClientID == "" {
			// Doing this is OK because the transaction signature has ClientID
			// that won't pass verification if some other client's public is put in
			co := client.NewClient()
			co.SetPublicKey(t.PublicKey)
			t.ClientID = co.ID
		}
	} else {
		if t.ClientID == "" {
			Logger.Error("invalid transaction", zap.String("txn", datastore.ToJSON(t).String()))
		}
	}
}

/*ValidateWrtTime - validate entityt w.r.t given time (as now) */
func (t *Transaction) ValidateWrtTime(ctx context.Context, ts common.Timestamp) error {
	return t.ValidateWrtTimeForBlock(ctx, ts, true)
}

/*ValidateWrtTimeForBlock - validate entityt w.r.t given time (as now) */
func (t *Transaction) ValidateWrtTimeForBlock(ctx context.Context, ts common.Timestamp, validateSignature bool) error {
	if t.Value < 0 {
		return common.InvalidRequest("value must be greater than or equal to zero")
	}
	// TODO: t.Fee needs to be compared to the minimum transaction fee once governance is implemented
	if t.Fee < 0 {
		return common.InvalidRequest("fee must be greater than or equal to zero")
	}
	err := config.ValidChain(datastore.ToString(t.ChainID))
	if err != nil {
		return err
	}
	if t.Hash == "" {
		return common.InvalidRequest("hash required for transaction")
	}
	if !common.WithinTime(int64(ts), int64(t.CreationDate), TXN_TIME_TOLERANCE) {
		return common.InvalidRequest(fmt.Sprintf("Transaction creation time not within tolerance: ts=%v txn.creation_date=%v", ts, t.CreationDate))
	}
	if t.ClientID == t.ToClientID {
		return common.InvalidRequest("from and to client should be different")
	}
	err = t.VerifyHash(ctx)
	if err != nil {
		return err
	}
	if validateSignature {
		err = t.VerifySignature(ctx)
		if err != nil {
			return err
		}
	}
	if t.OutputHash != "" {
		err = t.VerifyOutputHash(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

/*Validate - Entity implementation */
func (t *Transaction) Validate(ctx context.Context) error {
	return t.ValidateWrtTime(ctx, common.Now())
}

/*GetScore - score for write*/
func (t *Transaction) GetScore() int64 {
	return t.Fee
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

/*GetHashBytes - implement Hashable interface */
func (t *Transaction) GetHashBytes() []byte {
	return util.HashStringToBytes(t.Hash)
}

/*GetClient - get the Client object associated with the transaction */
func (t *Transaction) GetClient(ctx context.Context) (*client.Client, error) {
	co, err := client.GetClient(ctx, t.ClientID)
	if err != nil {
		return nil, err
	}
	return co, nil
}

/*HashData - data used to hash the transaction */
func (t *Transaction) HashData() string {
	hashdata := common.TimeToString(t.CreationDate) + ":" + t.ClientID + ":" + t.ToClientID + ":" + strconv.FormatInt(t.Value, 10) + ":" + encryption.Hash(t.TransactionData)
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
		return common.NewError("hash_mismatch", fmt.Sprintf("The hash of the data doesn't match with the provided hash: %v %v %v", t.Hash, t.ComputeHash(), t.HashData()))
	}
	return nil
}

/*VerifySignature - verify the transaction hash */
func (t *Transaction) VerifySignature(ctx context.Context) error {
	sigScheme, err := t.GetSignatureScheme(ctx)
	if err != nil {
		return err
	}
	correctSignature, err := sigScheme.Verify(t.Signature, t.Hash)
	if err != nil {
		return err
	}
	if !correctSignature {
		return common.NewError("invalid_signature", "Invalid Signature")
	}
	return nil
}

/*GetSignatureScheme - get the signature scheme associated with this transaction */
func (t *Transaction) GetSignatureScheme(ctx context.Context) (encryption.SignatureScheme, error) {
	var err error
	var co *client.Client
	if t.PublicKey == "" {
		co, err = t.GetClient(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		co = client.NewClient()
		co.ID = t.ClientID
		co.SetPublicKey(t.PublicKey)
	}
	return co.GetSignatureScheme(), nil
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
func (t *Transaction) Sign(signatureScheme encryption.SignatureScheme) (string, error) {
	t.Hash = t.ComputeHash()
	signature, err := signatureScheme.Sign(t.Hash)
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
	return summary
}

/*DebugTxn - is this a transaction that needs being debugged
- applicable only when running in test mode and the transaction_data string contains debug keyword somewhere in it
*/
func (t *Transaction) DebugTxn() bool {
	if !config.Development() {
		return false
	}
	return strings.Index(t.TransactionData, "debug") >= 0
}

/*ComputeOutputHash - compute the hash from the transaction output */
func (t *Transaction) ComputeOutputHash() string {
	if t.TransactionOutput == "" {
		return encryption.EmptyHash
	}
	return encryption.Hash(t.TransactionOutput)
}

/*VerifyOutputHash - Verify the hash of the transaction */
func (t *Transaction) VerifyOutputHash(ctx context.Context) error {
	if t.OutputHash != t.ComputeOutputHash() {
		Logger.Debug("verify output hash (hash mismatch)", zap.String("hash", t.OutputHash), zap.String("computed_hash", t.ComputeOutputHash()), zap.String("hash_data", t.TransactionOutput), zap.String("txn", datastore.ToJSON(t).String()))
		return common.NewError("hash_mismatch", fmt.Sprintf("The hash of the output doesn't match with the provided hash: %v %v %v", t.Hash, t.ComputeOutputHash(), t.TransactionOutput))
	}
	return nil
}

func SetTxnTimeout(timeout int64) {
	TXN_TIME_TOLERANCE = timeout
}
