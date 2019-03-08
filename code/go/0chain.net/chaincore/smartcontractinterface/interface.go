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
	DB           smartcontractstate.SCStateI
	ID           string
	RestHandlers map[string]SmartContractRestHandler
}

func NewSC(db smartcontractstate.SCStateI, id string) *SmartContract {
	restHandlers := make(map[string]SmartContractRestHandler)
	return &SmartContract{DB: db, ID: id, RestHandlers: restHandlers}
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error)
	SetSC(sc *SmartContract, bc BCContextI)
}

type BCContextI interface {
	GetNodepoolInfo() interface{}
}
