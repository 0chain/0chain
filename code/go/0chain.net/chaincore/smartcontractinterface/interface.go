package smartcontractinterface

import (
	c_state "0chain.net/smartcontract/common"
	"context"
	"encoding/json"
	"net/url"

	"0chain.net/chaincore/transaction"
)

const Seperator = ":"

type SmartContractRestHandler func(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error)

type SmartContract struct {
	ID                          string
	SmartContractExecutionStats map[string]interface{}
}

func NewSC(id string) *SmartContract {
	scExecStats := make(map[string]interface{})
	return &SmartContract{ID: id, SmartContractExecutionStats: scExecStats}
}

// SmartContractTransactionData is passed in Transaction.TransactionData
// InputData may contain Public Key in some cases
// FunctionName is user to invoke SC API function
type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error)
	GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error)
	GetExecutionStats() map[string]interface{}
	GetName() string
	GetAddress() string
	GetCostTable(balances c_state.StateContextI) (map[string]int, error)
}

/*
BCContextI interface for smart contracts to access blockchain.
These functions should not modify blockchain states in anyway.
*/
type BCContextI interface {
	GetNodepoolInfo() interface{}
}
