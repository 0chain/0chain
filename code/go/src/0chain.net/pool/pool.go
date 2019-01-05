package pool

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
)

type PoolTransferResponse struct {
	TxnHash    datastore.Key `json:"txn_hash,omitempty"`
	FromPool   datastore.Key `json:"from_pool,omitempty"`
	ToPool     datastore.Key `json:"to_pool,omitempty"`
	Value      state.Balance `json:"value,omitempty"`
	FromClient datastore.Key `json:"from_client,omitempty"`
	ToClient   datastore.Key `json:"to_client,omitempty"`
}

func (p *PoolTransferResponse) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *PoolTransferResponse) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

type pool struct {
	id      datastore.Key `json:"id"`
	balance state.Balance `json:"balance"`
}

func (p *pool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *pool) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

func (p *pool) GetBalance() state.Balance {
	return p.balance
}

func (p *pool) GetID() datastore.Key {
	return p.id
}

//pool to pool transfer
func (p *pool) TransferTo(op *pool, value state.Balance) (string, error) {
	if value > p.balance {
		return "", common.NewError("pool transfer failed", "value exceeds balance")
	}
	pr := &PoolTransferResponse{FromPool: p.id, ToPool: op.id, Value: value}
	op.balance += value
	p.balance -= value
	return string(pr.Encode()), nil
}

func (p *pool) EmptyPool(fromClientID, toClientID datastore.Key) (*state.Transfer, string) {
	transfer := state.NewTransfer(fromClientID, toClientID, p.balance)
	pr := &PoolTransferResponse{FromClient: fromClientID, ToClient: toClientID, Value: p.balance}
	p.balance = 0
	return transfer, string(pr.Encode())
}

func (p *pool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance) (*state.Transfer, string, error) {
	if value > p.balance {
		return nil, "", common.NewError("pool drain failed", "value exceeds balance")
	}
	pr := &PoolTransferResponse{FromClient: fromClientID, ToClient: toClientID, Value: value}
	transfer := state.NewTransfer(fromClientID, toClientID, value)
	p.balance -= value
	return transfer, string(pr.Encode()), nil
}

func (p *pool) FillPool(txn *transaction.Transaction) string {
	p.balance += state.Balance(txn.Value)
	pr := &PoolTransferResponse{TxnHash: txn.Hash, FromClient: txn.ClientID, ToPool: p.id}
	return string(pr.Encode())
}

func DigPool(id datastore.Key, txn *transaction.Transaction) *pool {
	p := &pool{id, state.Balance(txn.Value)}
	return p
}
