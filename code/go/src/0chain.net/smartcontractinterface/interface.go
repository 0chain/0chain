package smartcontractinterface

import (
	"encoding/json"

	"0chain.net/smartcontractstate"
	"0chain.net/transaction"
)

const Seperator = ":"

type SmartContract struct {
	db *smartcontractstate.NodeDB
	t  *transaction.Transaction
}

type SmartContractTransactionData struct {
	FunctionName string          `json:"name"`
	InputData    json.RawMessage `json:"input"`
}

type SmartContractInterface interface {
	Execute(t *transaction.Transaction, funcName string, input []byte) (string, error)
}
