package smartcontractinterface

import (
	"encoding/json"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/smartcontractstate"
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
	Execute(t *transaction.Transaction, b *block.Block, funcName string, input []byte, balances c_state.StateContextI) (string, error)
}
