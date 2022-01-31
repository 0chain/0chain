package tokenpool

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type ZcnPool struct {
	TokenPool
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

func (p *ZcnPool) SetBalance(value state.Balance) {
	p.Balance = value
}

func (p *ZcnPool) GetID() datastore.Key {
	return p.ID
}

func (p *ZcnPool) DigPool(id datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if txn.Value < 0 {
		return nil, "", common.NewError("digging pool failed", "insufficient funds")
	}

	p.TokenPool.ID = id // Transaction Hash
	p.TokenPool.Balance = state.Balance(txn.Value)

	tpr := &TokenPoolTransferResponse{
		TxnHash:    txn.Hash,       // transaction hash
		FromClient: txn.ClientID,   // authorizer node id
		ToPool:     p.ID,           // transaction hash
		ToClient:   txn.ToClientID, // smart contracts address
		Value:      state.Balance(txn.Value),
	}

	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, state.Balance(txn.Value))
	return transfer, string(tpr.Encode()), nil
}

func (p *ZcnPool) FillPool(txn *transaction.Transaction) (*state.Transfer, string, error) {
	if txn.Value <= 0 {
		return nil, "", common.NewError("filling pool failed", "insufficient funds")
	}
	p.Balance += state.Balance(txn.Value)
	tpr := &TokenPoolTransferResponse{TxnHash: txn.Hash, FromClient: txn.ClientID, ToPool: p.ID, ToClient: txn.ToClientID, Value: state.Balance(txn.Value)}
	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, state.Balance(txn.Value))
	return transfer, string(tpr.Encode()), nil
}

// TransferTo ZcnPool to ZcnPool transfer
func (p *ZcnPool) TransferTo(op TokenPoolI, value state.Balance, _ interface{}) (*state.Transfer, string, error) {
	if value > p.Balance {
		return nil, "", common.NewError("pool-to-pool transfer failed", "value exceeds balance")
	}
	tpr := &TokenPoolTransferResponse{FromPool: p.ID, ToPool: op.GetID(), Value: value}
	op.SetBalance(op.GetBalance() + value)
	p.Balance -= value
	return nil, string(tpr.Encode()), nil
}

func (p *ZcnPool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance, _ interface{}) (*state.Transfer, string, error) {
	if value > p.Balance {
		return nil, "", common.NewError("draining pool failed", "value exceeds balance")
	}
	tpr := &TokenPoolTransferResponse{FromClient: fromClientID, ToClient: toClientID, Value: value, FromPool: p.ID}
	transfer := state.NewTransfer(fromClientID, toClientID, value)
	p.Balance -= value
	return transfer, string(tpr.Encode()), nil
}

func (p *ZcnPool) EmptyPool(fromClientID, toClientID datastore.Key, _ interface{}) (*state.Transfer, string, error) {
	if p.Balance == 0 {
		return nil, "", common.NewError("emptying pool failed", "pool already empty")
	}
	transfer := state.NewTransfer(fromClientID, toClientID, p.Balance)
	tpr := &TokenPoolTransferResponse{FromClient: fromClientID, ToClient: toClientID, Value: p.Balance, FromPool: p.ID}
	p.Balance = 0
	return transfer, string(tpr.Encode()), nil
}
