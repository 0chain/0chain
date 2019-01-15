package tokenpool

import (
	"encoding/json"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
)

type ZcnLockingPool struct {
	ZcnPool
	TokenLock
}

func (p *ZcnLockingPool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *ZcnLockingPool) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

func (p *ZcnLockingPool) GetBalance() state.Balance {
	return p.Balance
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

func (p *ZcnLockingPool) TransferTo(op *ZcnLockingPool, value state.Balance) (string, error) {
	if p.Locked() {
		return common.NewError("pool-to-pool transfer failed", "pool is still locked").Error(), nil
	}
	return p.ZcnPool.TransferTo(&op.ZcnPool, value)
}

func (p *ZcnLockingPool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance) (*state.Transfer, string, error) {
	if p.Locked() {
		return nil, common.NewError("draining pool failed", "pool is still locked").Error(), nil
	}
	return p.ZcnPool.DrainPool(fromClientID, toClientID, value)
}

func (p *ZcnLockingPool) EmptyPool(fromClientID, toClientID datastore.Key) (*state.Transfer, string, error) {
	if p.Locked() {
		return nil, common.NewError("emptying pool failed", "pool is still locked").Error(), nil
	}
	return p.ZcnPool.EmptyPool(fromClientID, toClientID)
}

func (p *ZcnLockingPool) Locked() bool {
	return time.Now().Sub(p.StartTime) < p.Duration
}
