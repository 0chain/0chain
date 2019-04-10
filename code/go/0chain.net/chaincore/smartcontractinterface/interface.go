package smartcontractinterface

import (
	"context"
	"encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/chaincore/transaction"
)

const Seperator = ":"

type SmartContractRestHandler func(ctx context.Context, params url.Values) (interface{}, error)

type SmartContract struct {
	DB                          smartcontractstate.SCStateI
	ID                          string
	RestHandlers                map[string]SmartContractRestHandler
	SmartContractExecutionStats map[string]interface{}
}

func NewSC(db smartcontractstate.SCStateI, id string) *SmartContract {
	restHandlers := make(map[string]SmartContractRestHandler)
	scExecStats := make(map[string]interface{})
	return &SmartContract{DB: db, ID: id, RestHandlers: restHandlers, SmartContractExecutionStats: scExecStats}
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error)
	SetSC(sc *SmartContract, bc BCContextI)
	GetRestPoints() map[string]SmartContractRestHandler
}

/*BCContextI interface for smart contracts to access blockchain.
These functions should not modify blockchain states in anyway.
*/
type BCContextI interface {
	GetNodepoolInfo() interface{}
}
