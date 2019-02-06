package smartcontractinterface

import (
	"encoding/json"

	c_state "0chain.net/chain/state"
	"0chain.net/smartcontractstate"
	"0chain.net/transaction"
)

const Seperator = ":"

type SmartContract struct {
	DB smartcontractstate.SCStateI
	ID string
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error)
}
