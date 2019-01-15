package pool

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
)

type ZcnPool struct {
	Pool
}

func (p *ZcnPool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *ZcnPool) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

func (p *ZcnPool) GetBalance() state.Balance {
	return p.Balance
}

func (p *ZcnPool) GetID() datastore.Key {
	return p.ID
}

func (p *ZcnPool) DigPool(id datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if txn.Value <= 0 {
		return nil, "", common.NewError("digging pool failed", "insufficent funds")
	}
	p.Pool.ID = id
	p.Pool.Balance = state.Balance(txn.Value)
	pr := &PoolTransferResponse{TxnHash: txn.Hash, FromClient: txn.ClientID, ToPool: p.ID, ToClient: txn.ToClientID}
	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, state.Balance(txn.Value))
	return transfer, string(pr.Encode()), nil
}

func (p *ZcnPool) FillPool(txn *transaction.Transaction) (*state.Transfer, string, error) {
	if txn.Value <= 0 {
		return nil, "", common.NewError("filling pool failed", "insufficent funds")
	}
	p.Balance += state.Balance(txn.Value)
	pr := &PoolTransferResponse{TxnHash: txn.Hash, FromClient: txn.ClientID, ToPool: p.ID, ToClient: txn.ToClientID}
	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, state.Balance(txn.Value))
	return transfer, string(pr.Encode()), nil
}

//ZcnPool to ZcnPool transfer
func (p *ZcnPool) TransferTo(op *ZcnPool, value state.Balance) (string, error) {
	if value > p.Balance {
		return "", common.NewError("pool-to-pool transfer failed", "value exceeds balance")
	}
	pr := &PoolTransferResponse{FromPool: p.ID, ToPool: op.ID, Value: value}
	op.Balance += value
	p.Balance -= value
	return string(pr.Encode()), nil
}

func (p *ZcnPool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance) (*state.Transfer, string, error) {
	if value > p.Balance {
		return nil, "", common.NewError("draining pool failed", "value exceeds balance")
	}
	pr := &PoolTransferResponse{FromClient: fromClientID, ToClient: toClientID, Value: value, FromPool: p.ID}
	transfer := state.NewTransfer(fromClientID, toClientID, value)
	p.Balance -= value
	return transfer, string(pr.Encode()), nil
}

func (p *ZcnPool) EmptyPool(fromClientID, toClientID datastore.Key) (*state.Transfer, string, error) {
	if p.Balance == 0 {
		return nil, "", common.NewError("emptying pool failed", "pool already empty")
	}
	transfer := state.NewTransfer(fromClientID, toClientID, p.Balance)
	pr := &PoolTransferResponse{FromClient: fromClientID, ToClient: toClientID, Value: p.Balance, FromPool: p.ID}
	p.Balance = 0
	return transfer, string(pr.Encode()), nil
}
