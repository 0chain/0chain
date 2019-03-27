package smartcontract

import (
	"context"
	"encoding/json"
	"net/url"
	"sync"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

//lock used to setup smartcontract rest handlers
var scLock = sync.Mutex{}

//ContractMap - stores the map of valid smart contracts mapping from its address to its interface implementation
var ContractMap = map[string]sci.SmartContractInterface{}


//ExecuteRestAPI - executes the rest api on the smart contract
func ExecuteRestAPI(ctx context.Context, scAdress string, restpath string, params url.Values, ndb smartcontractstate.SCDB) (interface{}, error) {
	_, sc := getSmartContract(scAdress, ndb)
	if sc != nil {
		//add bc context here
		handler, restpathok := sc.RestHandlers[restpath]
		if !restpathok {
			return nil, common.NewError("invalid_path", "Invalid path")
		}
		return handler(ctx, params)
	}
	return nil, common.NewError("invalid_sc", "Invalid Smart contract address")
}

func getSmartContract(scAddress string, ndb smartcontractstate.SCDB) (sci.SmartContractInterface, *sci.SmartContract) {
	contracti, ok := ContractMap[scAddress]
	if ok {
		scLock.Lock()
		sc := sci.NewSC(smartcontractstate.NewSCState(ndb, scAddress), scAddress)

		bc := &BCContext{} 
		contracti.SetSC(sc, bc)
		scLock.Unlock()
		return contracti, sc
	}
	return nil, nil
}

//ExecuteSmartContract - executes the smart contract in the context of the given transaction
func ExecuteSmartContract(ctx context.Context, t *transaction.Transaction, ndb smartcontractstate.SCDB, balances c_state.StateContextI) (string, error) {
	contractObj, _ := getSmartContract(t.ToClientID, ndb)
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
