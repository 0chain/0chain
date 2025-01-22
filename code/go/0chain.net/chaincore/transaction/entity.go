package transaction

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"0chain.net/core/config"
	"0chain.net/core/viper"

	"encoding/json"

	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

/*TXN_TIME_TOLERANCE - the txn creation date should be within these many seconds before/after of current time */

var TXN_TIME_TOLERANCE int64

var transactionCount uint64 = 0

// ErrTxnMissingPublicKey is returned if the transaction does not have ClientID and public key
var (
	ErrTxnMissingPublicKey = errors.New("transaction missing public key")
	ErrTxnInvalidPublicKey = errors.New("transaction public key is invalid")
	ErrTxnInsufficientFee  = errors.New("insufficient transaction fee")
)

func SetupTransactionDB(redisTxnsHost string, redisTxnsPort int) {
	if len(redisTxnsHost) > 0 && redisTxnsPort > 0 {
		memorystore.AddPool("txndb", memorystore.NewPool(redisTxnsHost, redisTxnsPort))
	} else {
		//inside docker
		memorystore.AddPool("txndb", memorystore.NewPool(os.Getenv("REDIS_TXNS"), 6379))
	}
}

// swagger:model Transaction - transaction data
type Transaction struct {

	// Hash of the transaction
	datastore.HashIDField

	datastore.CollectionMemberField `json:"-" msgpack:"-"`

	// Version of the transaction
	//
	// required: true
	datastore.VersionField

	// a compination of smart contract address and function name
	*SmartContractData `json:"-" msgpack:"-"`

	// ClientID of the client issuing the transaction
	//
	// required: true
	ClientID string `json:"client_id" msgpack:"cid,omitempty"`

	// Public key of the client issuing the transaction
	//
	// required: true
	PublicKey string `json:"public_key,omitempty" msgpack:"puk,omitempty"`

	// ToClientID - the client id of the recipient, the other party in the transaction. It can be a client id or the address of a smart contract
	//
	// required: true
	ToClientID string `json:"to_client_id,omitempty" msgpack:"tcid,omitempty"`

	// ChainID - the chain id of the transaction
	//
	// required: true
	ChainID string `json:"chain_id,omitempty" msgpack:"chid"`

	// TransactionData - the data associated with the transaction
	//
	// required: true
	TransactionData string `json:"transaction_data" msgpack:"d"`

	// Value - a numeric value associated with this transaction. Its role is determined by the smart contract function
	//
	// required: true
	Value currency.Coin `json:"transaction_value" msgpack:"v"`

	// Signature - Issuer signature of the transaction
	//
	// required: true
	Signature string `json:"signature" msgpack:"s"`

	// CreationDate - the time when the transaction was created
	//
	// required: true
	CreationDate common.Timestamp `json:"creation_date" msgpack:"ts"`

	// Fee - the fee associated with the transaction
	//
	// required: true
	Fee currency.Coin `json:"transaction_fee" msgpack:"f"`

	// Nonce - the nonce associated with the transaction
	//
	// required: true
	Nonce int64 `json:"transaction_nonce" msgpack:"n"`

	// TransactionType - the type of the transaction.
	//	Possible values are:
	//		- 0: TxnTypeSend - A transaction to send tokens to another account, state is maintained by account.
	//		- 10: TxnTypeData - A transaction to just store a piece of data on the block chain.
	//		- 1000: TxnTypeSmartContract - A smart contract transaction type.
	// required: true
	TransactionType int `json:"transaction_type" msgpack:"tt"`

	// TransactionOutput - the output of the transaction
	//
	// required: true
	TransactionOutput string `json:"transaction_output,omitempty" msgpack:"o,omitempty"`

	// OutputHash - the hash of the transaction output
	//
	// required: true
	OutputHash string `json:"txn_output_hash" msgpack:"oh"`

	// Status - the status of the transaction
	//
	// required: true
	Status int `json:"transaction_status" msgpack:"sot"`

	IsExempt bool `json:"-" msgpack:"-"`
}

type FeeStats struct {
	MaxFees  currency.Coin `json:"max_fees"`
	MeanFees currency.Coin `json:"mean_fees"`
	MinFees  currency.Coin `json:"min_fees"`
}

var transactionEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (t *Transaction) GetEntityMetadata() datastore.EntityMetadata {
	return transactionEntityMetadata
}

/*ComputeProperties - Entity implementation */
func (t *Transaction) ComputeProperties() error {
	t.EntityCollection = txnEntityCollection
	if t.ChainID == "" {
		t.ChainID = datastore.ToKey(config.GetServerChainID())
	}
	t.SmartContractData = &SmartContractData{}
	if t.TransactionType == TxnTypeSmartContract {
		if err := json.Unmarshal([]byte(t.TransactionData), t.SmartContractData); err != nil {
			logging.Logger.Debug("transaction data", zap.Any("data", t.TransactionData))
			return fmt.Errorf("invalid smart contract data: %v", err)
		}
	}
	return t.ComputeClientID()
}

// swagger:model SmartContractData represents the smart contract data
type SmartContractData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

func (t *Transaction) ValidateNonce() error {
	if t.Nonce <= 0 {
		return errors.New("invalid transaction nonce")
	}

	return nil
}

// ValidateFee - Validate fee
func (t *Transaction) ValidateFee(txnExempted map[string]bool, minTxnFee currency.Coin) error {
	if t.TransactionData != "" {
		if _, ok := txnExempted[t.FunctionName]; ok {
			return nil
		}
	}
	if t.Fee < minTxnFee {
		return ErrTxnInsufficientFee
	}
	return nil
}

/*ComputeClientID - compute the client id if there is a public key in the transaction */
func (t *Transaction) ComputeClientID() error {
	if t.PublicKey == "" {
		logging.Logger.Error("invalid transaction",
			zap.Error(ErrTxnMissingPublicKey),
			zap.String("txn", datastore.ToJSON(t).String()))
		return ErrTxnMissingPublicKey
	}

	if t.ClientID != "" {
		return encryption.VerifyPublicKeyClientID(t.PublicKey, t.ClientID)
	}

	// Doing this is OK because the transaction signature has ClientID
	// that won't pass verification if some other client's public is put in
	id, err := client.GetIDFromPublicKey(t.PublicKey)
	if err != nil {
		logging.Logger.Error("invalid transaction public key",
			zap.String("public key", t.PublicKey),
			zap.Error(err))
		return ErrTxnInvalidPublicKey
	}

	t.ClientID = id
	return nil
}

/*ValidateWrtTime - validate entityt w.r.t given time (as now) */
func (t *Transaction) ValidateWrtTime(ctx context.Context, ts common.Timestamp) error {
	return t.ValidateWrtTimeForBlock(ctx, ts, true)
}

/*ValidateWrtTimeForBlock - validate entityt w.r.t given time (as now) */
func (t *Transaction) ValidateWrtTimeForBlock(ctx context.Context, ts common.Timestamp, validateSignature bool) error {
	if !encryption.IsHash(t.ToClientID) && t.ToClientID != "" {
		return common.InvalidRequest("to client id must be a hexadecimal hash")
	}

	// TODO: t.Fee needs to be compared to the minimum transaction fee once governance is implemented

	err := config.ValidChain(t.ChainID)
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
	if !encryption.IsHash(t.ToClientID) {
		return errors.New("invalid to client id")
	}
	return t.ValidateWrtTime(ctx, common.Now())
}

/*GetScore - score for write*/

func (t *Transaction) GetScore() (int64, error) {
	if config.Configuration().ChainConfig.IsFeeEnabled() {
		if t.IsExempt {
			// high score for exempt transactions
			return 100 * 1e10, nil
		}

		return t.Fee.Int64()
	}
	return 0, nil
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
	s := strings.Builder{}
	s.WriteString(common.TimeToString(t.CreationDate))
	s.WriteString(":")
	s.WriteString(strconv.FormatInt(t.Nonce, 10))
	s.WriteString(":")
	s.WriteString(t.ClientID)
	s.WriteString(":")
	s.WriteString(t.ToClientID)
	s.WriteString(":")
	s.WriteString(strconv.FormatUint(uint64(t.Value), 10))
	s.WriteString(":")
	s.WriteString(encryption.Hash(t.TransactionData))
	return s.String()
}

/*ComputeHash - compute the hash from the various components of the transaction */
func (t *Transaction) ComputeHash() string {
	return encryption.Hash(t.HashData())
}

/*VerifyHash - Verify the hash of the transaction */
func (t *Transaction) VerifyHash(ctx context.Context) error {
	if t.Hash != t.ComputeHash() {
		logging.Logger.Debug("verify hash (hash mismatch)",
			zap.String("hash", t.Hash),
			zap.String("computed_hash", t.ComputeHash()),
			zap.String("hash_data", t.HashData()),
			zap.String("txn", datastore.ToJSON(t).String()))
		return common.NewError("hash_mismatch",
			fmt.Sprintf("The hash of the data doesn't match with the provided hash: %v %v %v",
				t.Hash, t.ComputeHash(), t.HashData()))
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

	co, err := client.GetClientFromCache(t.ClientID)
	if err != nil {
		co = client.NewClient()
		co.ID = t.ClientID
		if err := co.SetPublicKey(t.PublicKey); err != nil {
			return nil, err
		}
		if err := client.PutClientCache(co); err != nil {
			return nil, err
		}
	}

	if co.SigScheme == nil {
		if t.PublicKey == "" {
			return nil, errors.New("get signature scheme failed, empty public key in transaction")
		}

		co.ID = t.ClientID
		if err := co.SetPublicKey(t.PublicKey); err != nil {
			return nil, err
		}
		if err := client.PutClientCache(co); err != nil {
			return nil, err
		}
	}

	return co.SigScheme, nil
}

func (t *Transaction) GetPublicKeyStr(ctx context.Context) (string, error) {
	if t.PublicKey != "" {
		return t.PublicKey, nil
	}

	co, err := client.GetClient(ctx, t.ClientID)
	if err != nil {
		return "", err
	}

	return co.PublicKey, nil
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	t := &Transaction{}
	t.Version = "1.0"
	t.CreationDate = common.Now()
	t.ChainID = datastore.ToKey(config.GetServerChainID())
	t.EntityCollection = txnEntityCollection
	t.SmartContractData = &SmartContractData{}
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

/*Sign - given a client and client's private key, sign this transaction */
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

/*
DebugTxn - is this a transaction that needs being debugged
- applicable only when running in test mode and the transaction_data string contains debug keyword somewhere in it
*/
func (t *Transaction) DebugTxn() bool {
	return config.Development() && viper.GetBool("logging.verbose")
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
		logging.Logger.Error("verify output hash (hash mismatch)", zap.String("hash", t.OutputHash), zap.String("computed_hash", t.ComputeOutputHash()), zap.String("hash_data", t.TransactionOutput), zap.String("txn", datastore.ToJSON(t).String()))
		return common.NewError("hash_mismatch", fmt.Sprintf("The hash of the output doesn't match with the provided hash: %v %v %v %v", t.Hash, t.OutputHash, t.ComputeOutputHash(), t.TransactionOutput))
	}
	return nil
}

// Clone returns a clone of the transaction instance
func (t *Transaction) Clone() *Transaction {
	clone := &Transaction{
		HashIDField:       t.HashIDField,
		VersionField:      t.VersionField,
		ClientID:          t.ClientID,
		PublicKey:         t.PublicKey,
		ToClientID:        t.ToClientID,
		ChainID:           t.ChainID,
		TransactionData:   t.TransactionData,
		Value:             t.Value,
		Signature:         t.Signature,
		CreationDate:      t.CreationDate,
		Fee:               t.Fee,
		Nonce:             t.Nonce,
		TransactionType:   t.TransactionType,
		TransactionOutput: t.TransactionOutput,
		OutputHash:        t.OutputHash,
		Status:            t.Status,
		IsExempt:          t.IsExempt,
	}

	if t.SmartContractData != nil {
		scData := &SmartContractData{}
		*scData = *t.SmartContractData
		clone.SmartContractData = scData
	}

	if ent := t.CollectionMemberField.EntityCollection; ent != nil {
		clone.CollectionMemberField.EntityCollection = &datastore.EntityCollection{
			CollectionName:     ent.CollectionName,
			CollectionSize:     ent.CollectionSize,
			CollectionDuration: ent.CollectionDuration,
		}
	}
	return clone
}

func SetTxnTimeout(timeout int64) {
	TXN_TIME_TOLERANCE = timeout
}

func GetTransactionCount() uint64 {
	return atomic.LoadUint64(&transactionCount)
}
func IncTransactionCount() uint64 {
	return atomic.AddUint64(&transactionCount, 1)
}
