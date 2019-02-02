package tokenpool

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
)

type ZcnLockingPool struct {
	ZcnPool            `json:"pool"`
	TokenLockInterface `json:"lock"`
}

func (p *ZcnLockingPool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *ZcnLockingPool) Decode(input []byte, tokelock TokenLockInterface) error {
	p.TokenLockInterface = tokelock
	err := json.Unmarshal(input, p)
	return err
}

func (p *ZcnLockingPool) GetBalance() state.Balance {
	return p.Balance
}

func (p *ZcnLockingPool) SetBalance(value state.Balance) {
	p.Balance = value
}

func (p *ZcnLockingPool) GetID() datastore.Key {
	return p.ID
}

func (p *ZcnLockingPool) DigPool(id datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error) {
	return p.ZcnPool.DigPool(id, txn)
}

func (p *ZcnLockingPool) FillPool(txn *transaction.Transaction) (*state.Transfer, string, error) {
	return p.ZcnPool.FillPool(txn)
}

func (p *ZcnLockingPool) TransferTo(op TokenPoolI, value state.Balance, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if p.IsLocked(txn) {
		return nil, "", common.NewError("pool-to-pool transfer failed", "pool is still locked")
	}
	return p.ZcnPool.TransferTo(op, value, txn)
}

func (p *ZcnLockingPool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if p.IsLocked(txn) {
		return nil, "", common.NewError("draining pool failed", "pool is still locked")
	}
	return p.ZcnPool.DrainPool(fromClientID, toClientID, value, txn)
}

func (p *ZcnLockingPool) EmptyPool(fromClientID, toClientID datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if p.IsLocked(txn) {
		return nil, "", common.NewError("emptying pool failed", "pool is still locked")
	}
	return p.ZcnPool.EmptyPool(fromClientID, toClientID, txn)
}
