package httpclientutil

import (
	"0chain.net/core/common"
)

//Transaction entity that encapsulates the transaction related data and meta data
type Transaction struct {
	Hash              string           `json:"hash,omitempty"`
	Version           string           `json:"version,omitempty"`
	ClientID          string           `json:"client_id,omitempty"`
	PublicKey         string           `json:"public_key,omitempty"`
	ToClientID        string           `json:"to_client_id,omitempty"`
	ChainID           string           `json:"chain_id,omitempty"`
	TransactionData   string           `json:"transaction_data,omitempty"`
	Value             int64            `json:"transaction_value,omitempty"`
	Signature         string           `json:"signature,omitempty"`
	CreationDate      common.Timestamp `json:"creation_date,omitempty"`
	Fee               int64            `json:"transaction_fee,omitempty"`
	Nonce             int64            `json:"transaction_nonce,omitempty"`
	TransactionType   int              `json:"transaction_type,omitempty"`
	TransactionOutput string           `json:"transaction_output,omitempty"`
	OutputHash        string           `json:"txn_output_hash"`
}

const (
	//TxnTypeSend A transaction to send tokens to another account, state is maintained by account
	TxnTypeSend = 0

	//TxnTypeLockIn A transaction to lock tokens, state is maintained on the account and the parent lock in transaction
	TxnTypeLockIn = 2

	// Any txn type that refers to a parent txn should have an odd value

	//TxnTypeStorageWrite A transaction to write data to the blobber
	TxnTypeStorageWrite = 101

	//TxnTypeStorageRead A transaction to read data from the blobber
	TxnTypeStorageRead = 103
	//TxnTypeData A transaction to just store a piece of data on the block chain
	TxnTypeData = 10

	//TxnTypeStats A smart contract transaction type
	TxnTypeStats = 1000
)

//SmartContractTxnData Smart Contract Txn Data
type SmartContractTxnData struct {
	Name      string      `json:"name"`
	InputArgs interface{} `json:"input"`
}

// NewTransactionEntity creates a new transaction
func NewTransactionEntity(ID string, chainID string, pkey string) *Transaction {
	txn := &Transaction{}
	txn.Version = "1.0"
	txn.ClientID = ID //node.Self.ID
	txn.CreationDate = common.Now()
	txn.ChainID = chainID //chain.GetServerChain().ID
	txn.PublicKey = pkey  //node.Self.PublicKey
	return txn
}
