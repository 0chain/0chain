package pool

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
)

type Pool struct {
	id      datastore.Key `json:"id"`
	balance state.Balance `json:"balance"`
}

func (p *Pool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *Pool) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

//Pool to pool transfer
func (p *Pool) TransferTo(op *Pool, value state.Balance) error {
	if value > p.balance {
		return common.NewError("pool transfer failed", "value exceeds balance")
	}
	op.balance += value
	p.balance -= value
	return nil
}

func (p *Pool) EmptyPool(fromClientID, toClientID datastore.Key) *state.Transfer {
	transfer := state.NewTransfer(fromClientID, toClientID, p.balance)
	p.balance = 0
	return transfer
}

func (p *Pool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance) (*state.Transfer, error) {
	if value > p.balance {
		return nil, common.NewError("pool drain failed", "value exceeds balance")
	}
	transfer := state.NewTransfer(fromClientID, toClientID, value)
	p.balance -= value
	return transfer, nil
}

func (p *Pool) FillPool(txn *transaction.Transaction) {
	p.balance += txn.Value
}

func DigPool(id datastore.Key, txn *transaction.Transaction) *Pool {
	p := &Pool{id, txn.Value}
	return p
}
