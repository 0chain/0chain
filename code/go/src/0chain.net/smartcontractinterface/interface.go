package smartcontractinterface

import (
	"encoding/json"

	"0chain.net/smartcontractstate"
	"0chain.net/transaction"
)

const Seperator = ":"

type SmartContract struct {
	DB smartcontractstate.SCDB
	ID string
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte) (string, error)
}
