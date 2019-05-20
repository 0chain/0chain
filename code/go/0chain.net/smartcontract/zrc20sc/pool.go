package zrc20sc

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

type zrc20Pool struct {
	tokenpool.TokenPool
	tokenInfo
}

func (zcr *zrc20Pool) Encode() []byte {
	buff, _ := json.Marshal(zcr)
	return buff
}

func (zcr *zrc20Pool) Decode(input []byte) error {
	err := json.Unmarshal(input, zcr)
	return err
}

func (zrc *zrc20Pool) getKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + encryption.Hash(zrc.TokenName) + zrc.ID)
}

func (zrc *zrc20Pool) GetBalance() state.Balance {
	return zrc.Balance
}

func (zrc *zrc20Pool) SetBalance(value state.Balance) {
	zrc.Balance = value
}

func (zrc *zrc20Pool) GetID() datastore.Key {
	return zrc.ID
}

func (zrc *zrc20Pool) DigPool(id datastore.Key, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if zrc == nil {
		return nil, "", common.NewError("digging pool failed", "token info not set")
	}
	zcnUsed := (state.Balance(txn.Value) / zrc.ExchangeRate.ZCN) * zrc.ExchangeRate.ZCN
	if zcnUsed == 0 {
		return nil, "", common.NewError("digging pool failed", "insufficient funds to swap tokens")
	}
	newTokens := (zcnUsed / zrc.ExchangeRate.ZCN) * zrc.ExchangeRate.Other
	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, zcnUsed)
	zrc.ID = id
	zrc.Balance = newTokens
	zr := &zrc20PoolResponse{TokenPoolTransferResponse: tokenpool.TokenPoolTransferResponse{TxnHash: txn.Hash, FromClient: txn.ClientID, ToClient: txn.ToClientID, Value: zcnUsed, ToPool: zrc.ID}, FromToken: zrc.tokenInfo, FromPoolValue: zrc.Balance}
	return transfer, string(zr.encode()), nil
}

func (zrc *zrc20Pool) FillPool(txn *transaction.Transaction) (*state.Transfer, string, error) {
	if zrc == nil {
		return nil, "", common.NewError("filling pool failed", "token info not set")
	}
	zcnUsed := (state.Balance(txn.Value) / zrc.ExchangeRate.ZCN) * zrc.ExchangeRate.ZCN
	if zcnUsed == 0 {
		return nil, "", common.NewError("filling pool failed", "insufficient funds to swap tokens")
	}
	newTokens := (zcnUsed / zrc.ExchangeRate.ZCN) * zrc.ExchangeRate.Other
	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, zcnUsed)
	zrc.Balance += newTokens
	zr := &zrc20PoolResponse{TokenPoolTransferResponse: tokenpool.TokenPoolTransferResponse{TxnHash: txn.Hash, FromClient: txn.ClientID, ToClient: txn.ToClientID, Value: zcnUsed, ToPool: zrc.ID}, FromToken: zrc.tokenInfo, FromPoolValue: zrc.Balance}
	return transfer, string(zr.encode()), nil
}

func (zrc *zrc20Pool) TransferTo(op *zrc20Pool, value state.Balance, txn *transaction.Transaction) (*state.Transfer, string, error) {
	if zrc == nil || op == nil {
		return nil, "", common.NewError("pool-to-pool transfer failed", "one of the pools doesn't exist")
	}
	if value > zrc.Balance {
		return nil, "", common.NewError("pool-to-pool transfer failed", "value exceeds balance")
	}
	if zrc.TokenName != op.TokenName {
		return zrc.interPoolTransfer(op, value, txn)
	}
	op.Balance += value
	zrc.Balance -= value
	zr := &zrc20PoolResponse{TokenPoolTransferResponse: tokenpool.TokenPoolTransferResponse{FromPool: zrc.ID, ToPool: op.ID, Value: value}, FromToken: zrc.tokenInfo, FromPoolValue: zrc.Balance}
	return nil, string(zr.encode()), nil
}

func (zrc *zrc20Pool) interPoolTransfer(op *zrc20Pool, value state.Balance, txn *transaction.Transaction) (*state.Transfer, string, error) {
	otherUsed := (value / zrc.ExchangeRate.Other) * zrc.ExchangeRate.Other
	zcnWorth := (otherUsed / zrc.ExchangeRate.Other) * zrc.ExchangeRate.ZCN
	if zcnWorth == 0 {
		return nil, "", common.NewError("interpool transfer failed", "insufficent funds to exchange from this pool")
	}
	zcnOtherToken := (zcnWorth / op.ExchangeRate.ZCN) * op.ExchangeRate.ZCN
	if zcnOtherToken == 0 {
		return nil, "", common.NewError("interpool transfer failed", "insufficent funds to exchange to another pool")
	}
	leftOver := zcnWorth - zcnOtherToken
	transfer := state.NewTransfer(txn.ToClientID, txn.ClientID, leftOver)
	otherTransfered := (zcnOtherToken / op.ExchangeRate.ZCN) * op.ExchangeRate.Other
	zrc.Balance -= otherUsed
	op.Balance += otherTransfered
	zr := &zrc20PoolResponse{TokenPoolTransferResponse: tokenpool.TokenPoolTransferResponse{FromPool: zrc.ID, ToPool: op.ID}, FromToken: zrc.tokenInfo, ToToken: op.tokenInfo, FromPoolValue: otherUsed}
	return transfer, string(zr.encode()), nil
}

func (zrc *zrc20Pool) DrainPool(fromClientID, toClientID datastore.Key, value state.Balance) (*state.Transfer, string, error) {
	if zrc == nil {
		return nil, "", common.NewError("draining pool failed", "pool doesn't exist")
	}
	if value > zrc.GetBalance() {
		return nil, "", common.NewError("draining pool failed", "value exceeds balance")
	}
	otherUsed := (value / zrc.ExchangeRate.Other) * zrc.ExchangeRate.Other
	zcnUsed := (value / zrc.ExchangeRate.Other) * zrc.ExchangeRate.ZCN
	if zcnUsed == 0 {
		return nil, "", common.NewError("draining pool failed", "insufficient funds to swap tokens")
	}
	transfer := state.NewTransfer(fromClientID, toClientID, zcnUsed)
	zrc.Balance -= otherUsed
	zr := &zrc20PoolResponse{TokenPoolTransferResponse: tokenpool.TokenPoolTransferResponse{FromClient: toClientID, FromPool: zrc.ID, ToClient: fromClientID, Value: zcnUsed}, FromToken: zrc.tokenInfo, FromPoolValue: zrc.Balance}
	return transfer, string(zr.encode()), nil
}

func (zrc *zrc20Pool) EmptyPool(fromClientID, toClientID datastore.Key) (*state.Transfer, string, error) {
	if zrc == nil {
		return nil, "", common.NewError("emptying pool failed", "pool doesn't exist")
	}
	zcnUsed := (zrc.Balance / zrc.ExchangeRate.Other) * zrc.ExchangeRate.ZCN
	if zcnUsed == 0 {
		return nil, "", common.NewError("emptying pool failed", "insufficient funds to swap tokens")
	}
	transfer := state.NewTransfer(fromClientID, toClientID, zcnUsed)
	zrc.Balance = 0
	zr := &zrc20PoolResponse{TokenPoolTransferResponse: tokenpool.TokenPoolTransferResponse{FromClient: toClientID, FromPool: zrc.ID, ToClient: fromClientID, Value: zcnUsed}, FromToken: zrc.tokenInfo, FromPoolValue: zrc.Balance}
	return transfer, string(zr.encode()), nil
}
