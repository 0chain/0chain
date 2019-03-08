package smartcontract

import (
	"context"
	"encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var ContractMap = map[string]sci.SmartContractInterface{}

func ExecuteRestAPI(ctx context.Context, scAdress string, restpath string, params url.Values, ndb smartcontractstate.SCDB) (interface{}, error) {
	contracti, ok := ContractMap[scAdress]
	if ok {
		sc := sci.NewSC(smartcontractstate.NewSCState(ndb, scAdress), scAdress)
		bc := &BCContext{} 
		contracti.SetSC(sc, bc)
		handler, restpathok := sc.RestHandlers[restpath]
		if !restpathok {
			return nil, common.NewError("invalid_path", "Invalid path")
		}
		return handler(ctx, params)
	}
	return nil, common.NewError("invalid_sc", "Invalid Smart contract address")
}

func getSmartContract(t *transaction.Transaction, ndb smartcontractstate.SCDB) sci.SmartContractInterface {
	contracti, ok := ContractMap[t.ToClientID]
	if ok {
		bc := &BCContext{}
		contracti.SetSC(sci.NewSC(smartcontractstate.NewSCState(ndb, t.ToClientID), t.ToClientID), bc)
		return contracti
	}
	return nil
}

func ExecuteSmartContract(ctx context.Context, t *transaction.Transaction, ndb smartcontractstate.SCDB, balances c_state.StateContextI) (string, error) {
	contractObj := getSmartContract(t, ndb)
	if contractObj != nil {
		var smartContractData sci.SmartContractTransactionData
		dataBytes := []byte(t.TransactionData)
		err := json.Unmarshal(dataBytes, &smartContractData)
		if err != nil {
			Logger.Error("Error while decoding the JSON from transaction", zap.Any("input", t.TransactionData), zap.Any("error", err))
			return "", err
		}
		transactionOutput, err := contractObj.Execute(t, smartContractData.FunctionName, []byte(smartContractData.InputData), balances)
		if err != nil {
			return "", err
		}
		return transactionOutput, nil
	}
	return "", common.NewError("invalid_smart_contract_address", "Invalid Smart Contract address")
}
