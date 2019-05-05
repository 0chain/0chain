package zrc20sc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	Seperator = smartcontractinterface.Seperator
	ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d5"
	name      = "zrc20"
)

type ZRC20SmartContract struct {
	*smartcontractinterface.SmartContract
}

func (zrc *ZRC20SmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	zrc.SmartContract = sc
	zrc.SmartContractExecutionStats["createToken"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zrc.ID, "createToken"), nil)
	zrc.SmartContractExecutionStats["digPool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zrc.ID, "digPool"), nil)
	zrc.SmartContractExecutionStats["fillPool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zrc.ID, "fillPool"), nil)
	zrc.SmartContractExecutionStats["transferTo"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zrc.ID, "transferTo"), nil)
	zrc.SmartContractExecutionStats["drainPool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zrc.ID, "drainPool"), nil)
	zrc.SmartContractExecutionStats["emptyPool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", zrc.ID, "emptyPool"), nil)
}

func (zrc *ZRC20SmartContract) GetName() string {
	return name
}

func (zrc *ZRC20SmartContract) GetAddress() string {
	return ADDRESS
}

func (zrc *ZRC20SmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return zrc.RestHandlers
}

func (zrc *ZRC20SmartContract) createToken(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (string, error) {
	newRequest := &tokenNode{}
	err := newRequest.Decode(inputData)
	if err != nil {
		return common.NewError("bad request", "token cannot be created, request not formated correctly").Error(), nil
	}

	if !newRequest.validate() {
		return common.NewError("bad request", "token cannot be created, request is not filled out correctly").Error(), nil
	}
	token, _ := zrc.getTokenNode(newRequest.TokenName, balances)
	if token != nil {
		return common.NewError("bad request", "token already exists").Error(), nil
	}
	newRequest.Available = newRequest.TotalSupply
	balances.InsertTrieNode(newRequest.getKey(zrc.ID), newRequest)
	return string(newRequest.Encode()), nil
}

func (zrc *ZRC20SmartContract) digPool(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (string, error) {
	var newRequest zrc20TransferRequest
	zrcPool := &zrc20Pool{}
	err := newRequest.decode(inputData)
	if err != nil {
		return err.Error(), nil
	}
	token, err := zrc.getTokenNode(newRequest.FromToken, balances)
	if err != nil {
		return err.Error(), nil
	}
	zrcPool.tokenInfo = token.tokenInfo
	transfer, resp, err := zrcPool.DigPool(t.ClientID, t)
	if err != nil {
		return err.Error(), nil
	}
	tokensRequested := (transfer.Amount / token.ExchangeRate.ZCN) * token.ExchangeRate.Other
	if tokensRequested > token.Available {
		return common.NewError("digging pool failed", "tokens requested exceeds availble tokens").Error(), nil
	}
	balances.AddTransfer(transfer)
	token.Available -= tokensRequested
	balances.InsertTrieNode(token.getKey(zrc.ID), token)
	balances.InsertTrieNode(zrcPool.getKey(zrc.ID), zrcPool)
	return resp, nil
}

func (zrc *ZRC20SmartContract) fillPool(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (string, error) {
	var newRequest zrc20TransferRequest
	err := newRequest.decode(inputData)
	if err != nil {
		return err.Error(), nil
	}
	token, err := zrc.getTokenNode(newRequest.FromToken, balances)
	if err != nil {
		return err.Error(), nil
	}
	zrcPool, err := zrc.getPool(newRequest.FromToken, newRequest.TokenPoolTransferResponse.ToPool, balances)
	if err != nil {
		return err.Error(), nil
	}
	transfer, resp, err := zrcPool.FillPool(t)
	if err != nil {
		return err.Error(), nil
	}
	tokensRequested := (transfer.Amount / token.ExchangeRate.ZCN) * token.ExchangeRate.Other
	if tokensRequested > token.Available {
		return common.NewError("filling pool failed", "tokens requested exceeds availble tokens").Error(), nil
	}
	balances.AddTransfer(transfer)
	token.Available -= tokensRequested
	balances.InsertTrieNode(token.getKey(zrc.ID), token)
	balances.InsertTrieNode(zrcPool.getKey(zrc.ID), zrcPool)
	return resp, nil
}

func (zrc *ZRC20SmartContract) transferTo(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (string, error) {
	var newRequest zrc20TransferRequest
	err := newRequest.decode(inputData)
	if err != nil {
		return err.Error(), nil
	}
	zrcPool, err := zrc.getPool(newRequest.FromToken, newRequest.FromPool, balances)
	if err != nil {
		return err.Error(), nil
	}
	otherPool, err := zrc.getPool(newRequest.ToToken, newRequest.ToPool, balances)
	if err != nil {
		return err.Error(), nil
	}
	transfer, resp, err := zrcPool.TransferTo(otherPool, newRequest.Value, t)
	if err != nil {
		return err.Error(), nil
	}
	if transfer.Amount > 0 {
		balances.AddTransfer(transfer)
	}
	balances.InsertTrieNode(zrcPool.getKey(zrc.ID), zrcPool)
	balances.InsertTrieNode(otherPool.getKey(zrc.ID), otherPool)
	return resp, nil
}

func (zrc *ZRC20SmartContract) drainPool(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (string, error) {
	var newRequest zrc20TransferRequest
	err := newRequest.decode(inputData)
	if err != nil {
		return common.NewError("bad request", "token cannot be created, request not formated correctly").Error(), nil
	}
	token, err := zrc.getTokenNode(newRequest.FromToken, balances)
	if err != nil {
		return err.Error(), nil
	}
	zrcPool, err := zrc.getPool(newRequest.FromToken, newRequest.FromPool, balances)
	if err != nil {
		return err.Error(), nil
	}
	transfer, resp, err := zrcPool.DrainPool(zrc.ID, newRequest.ToClient, state.Balance(t.Value))
	if err != nil {
		return err.Error(), nil
	}
	tokensPutBack := (transfer.Amount / token.ExchangeRate.ZCN) * token.ExchangeRate.Other
	balances.AddTransfer(transfer)
	token.Available += tokensPutBack
	balances.InsertTrieNode(token.getKey(zrc.ID), token)
	balances.InsertTrieNode(zrcPool.getKey(zrc.ID), zrcPool)
	return resp, nil
}

func (zrc *ZRC20SmartContract) emptyPool(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (string, error) {
	var newRequest zrc20TransferRequest
	err := newRequest.decode(inputData)
	if err != nil {
		return common.NewError("bad request", "token cannot be created, request not formated correctly").Error(), nil
	}
	token, err := zrc.getTokenNode(newRequest.FromToken, balances)
	if err != nil {
		return err.Error(), nil
	}
	zrcPool, err := zrc.getPool(newRequest.FromToken, newRequest.FromPool, balances)
	if err != nil {
		return err.Error(), nil
	}
	transfer, resp, err := zrcPool.EmptyPool(zrc.ID, newRequest.ToClient)
	if err != nil {
		return err.Error(), nil
	}
	tokensPutBack := (transfer.Amount / token.ExchangeRate.ZCN) * token.ExchangeRate.Other
	balances.AddTransfer(transfer)
	token.Available += tokensPutBack
	balances.InsertTrieNode(token.getKey(zrc.ID), token)
	balances.DeleteTrieNode(zrcPool.getKey(zrc.ID))
	return resp, nil
}

func (zrc *ZRC20SmartContract) getPool(tokenName string, id datastore.Key, balances c_state.StateContextI) (*zrc20Pool, error) {
	zrcPool := &zrc20Pool{}
	zrcPool.ID = id
	zrcPool.TokenName = tokenName
	poolBytes, err := balances.GetTrieNode(zrcPool.getKey(zrc.ID))
	if err != nil {
		return nil, err
	}
	if poolBytes == nil {
		return nil, common.NewError("zrc20sc get pool", "pool doesn't exist")
	}
	err = zrcPool.Decode(poolBytes.Encode())
	if err != nil {
		return nil, err
	}
	return zrcPool, nil
}

func (zrc *ZRC20SmartContract) getTokenNode(tokenName string, balances c_state.StateContextI) (*tokenNode, error) {
	token := &tokenNode{}
	token.TokenName = tokenName
	tokenBytes, err := balances.GetTrieNode(token.getKey(zrc.ID))
	if err != nil {
		return nil, err
	}
	if tokenBytes == nil {
		return nil, common.NewError("zrc20sc get node", "token node doesn't exist")
	}
	err = token.Decode(tokenBytes.Encode())
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (zrc *ZRC20SmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	switch funcName {
	case "createToken":
		return zrc.createToken(t, inputData, balances)
	case "digPool":
		return zrc.digPool(t, inputData, balances)
	case "fillPool":
		return zrc.fillPool(t, inputData, balances)
	case "transferTo":
		return zrc.transferTo(t, inputData, balances)
	case "drainPool":
		return zrc.drainPool(t, inputData, balances)
	case "emptyPool":
		return zrc.emptyPool(t, inputData, balances)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
