package tokenpool

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

type TokenPoolTransferResponse struct {
	TxnHash    datastore.Key `json:"txn_hash,omitempty"`
	FromPool   datastore.Key `json:"from_pool,omitempty"`
	ToPool     datastore.Key `json:"to_pool,omitempty"`
	Value      state.Balance `json:"value,omitempty"`
	FromClient datastore.Key `json:"from_client,omitempty"`
	ToClient   datastore.Key `json:"to_client,omitempty"`
}

func (p *TokenPoolTransferResponse) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *TokenPoolTransferResponse) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

type TokenPoolI interface {
	GetBalance() state.Balance
	SetBalance(value state.Balance)
	GetID() datastore.Key
	DigPool(id datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error)
	FillPool(txn *transaction.Transaction) (*state.Transfer, string, error)
	TransferTo(op TokenPoolI, value state.Balance, entity interface{}) (*state.Transfer, string, error)
	DrainPool(fromClientID, toClientID datastore.Key, value state.Balance, entity interface{}) (*state.Transfer, string, error)
	EmptyPool(fromClientID, toClientID datastore.Key, entity interface{}) (*state.Transfer, string, error)
}

type TokenPool struct {
	ID      datastore.Key `json:"id"`
	Balance state.Balance `json:"balance"`
}

type TokenLockInterface interface {
	IsLocked(entity interface{}) bool
	LockStats(entity interface{}) []byte
}
