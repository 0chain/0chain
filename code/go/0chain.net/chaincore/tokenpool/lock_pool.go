package tokenpool

import (
	"encoding/json"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

//go:generate msgp -tests=false -io=false -v

type ZcnLockingPool struct {
	ZcnPool            `json:"pool"`
	TokenLockInterface `json:"lock"`
}

func (p *ZcnLockingPool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *ZcnLockingPool) Decode(input []byte, tokenlock TokenLockInterface) error {
	p.TokenLockInterface = tokenlock
	err := json.Unmarshal(input, p)
	return err
}

func (p *ZcnLockingPool) GetBalance() currency.Coin {
	return p.Balance
}

func (p *ZcnLockingPool) SetBalance(value currency.Coin) {
	p.Balance = value
}

func (p *ZcnLockingPool) GetID() string {
	return p.ID
}

func (p *ZcnLockingPool) DigPool(id string, txn *transaction.Transaction) (*state.Transfer, string, error) {
	return p.ZcnPool.DigPool(id, txn)
}

func (p *ZcnLockingPool) FillPool(txn *transaction.Transaction) (*state.Transfer, string, error) {
	return p.ZcnPool.FillPool(txn)
}

func (p *ZcnLockingPool) TransferTo(op TokenPoolI, value currency.Coin, entity interface{}) (*state.Transfer, string, error) {
	if p.IsLocked(entity) {
		return nil, "", common.NewError("pool-to-pool transfer failed", "pool is still locked")
	}
	return p.ZcnPool.TransferTo(op, value, entity)
}

func (p *ZcnLockingPool) DrainPool(fromClientID, toClientID string, value currency.Coin, entity interface{}) (*state.Transfer, string, error) {
	if p.IsLocked(entity) {
		return nil, "", common.NewError("draining pool failed", "pool is still locked")
	}
	return p.ZcnPool.DrainPool(fromClientID, toClientID, value, entity)
}

func (p *ZcnLockingPool) EmptyPool(fromClientID, toClientID string, entity interface{}) (*state.Transfer, string, error) {
	if p.IsLocked(entity) {
		return nil, "", common.NewError("emptying pool failed", "pool is still locked")
	}
	return p.ZcnPool.EmptyPool(fromClientID, toClientID, entity)
}
