package tokenpool

import (
	"encoding/json"
	"time"

	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
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
	GetID() datastore.Key
	DigPool(id datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error)
	FillPool(txn *transaction.Transaction) (*state.Transfer, string, error)
	TransferTo(op *TokenPoolI, value state.Balance) (string, error)
	DrainPool(fromClientID, toClientID datastore.Key, value state.Balance, txn *transaction.Transaction) (*state.Transfer, string, error)
	EmptyPool(fromClientID, toClientID datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error)
}

type TokenPool struct {
	ID      datastore.Key `json:"id"`
	Balance state.Balance `json:"balance"`
}

type TokenLock struct {
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration"`
	Owner     datastore.Key `json:"owner"`
	// for future use
	// Leaser          datastore.Key   `json:"leaser"`
	// LockExecutors   []datastore.Key `json:"lock_executors"`
	// PayoutExecutors []datastore.Key `json:"payout_executors"`
}

func (l *TokenLock) Encode() []byte {
	buff, _ := json.Marshal(l)
	return buff
}

func (l *TokenLock) Decode(input []byte) error {
	err := json.Unmarshal(input, l)
	return err
}
