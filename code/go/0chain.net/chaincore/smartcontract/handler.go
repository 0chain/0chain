package smartcontract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//ContractMap - stores the map of valid smart contracts mapping from its address to its interface implementation
var ContractMap = map[string]sci.SmartContractInterface{}

//ExecuteRestAPI - executes the rest api on the smart contract
func ExecuteRestAPI(ctx context.Context, scAdress string, restpath string, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	scI := getSmartContract(scAdress)
	if scI != nil {
		//add bc context here
		handler, restpathok := scI.GetRestPoints()[restpath]
		if !restpathok {
			return nil, common.NewError("invalid_path", "Invalid path")
		}
		return handler(ctx, params, balances)
	}
	return nil, common.NewError("invalid_sc", "Invalid Smart contract address")
}

func ExecuteStats(ctx context.Context, scAdress string, params url.Values, w http.ResponseWriter) {
	scI := getSmartContract(scAdress)
	if scI != nil {
		i, err := scI.GetHandlerStats(ctx, params)
		if err != nil {
			logging.Logger.Warn("unexpected error", zap.Error(err))
		}
		fmt.Fprintf(w, "%v", i)
		return
	}
	fmt.Fprintf(w, "invalid_sc: Invalid Smart contract address")
}

func getSmartContract(scAddress string) sci.SmartContractInterface {
	contracti, ok := ContractMap[scAddress]
	if ok {
		return contracti
	}
	return nil
}

func GetSmartContract(scAddress string) sci.SmartContractInterface {
	return getSmartContract(scAddress)
}

func ExecuteWithStats(smcoi sci.SmartContractInterface, t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {
	ts := time.Now()
	inter, err := smcoi.Execute(t, funcName, input, balances)
	if err == nil {
		if tm := smcoi.GetExecutionStats()[funcName]; tm != nil {
			if timer, ok := tm.(metrics.Timer); ok {
				timer.Update(time.Since(ts))
			}
		}
	}
	return inter, err
}

//ExecuteSmartContract - executes the smart contract in the context of the given transaction
func ExecuteSmartContract(ctx context.Context, t *transaction.Transaction, balances c_state.StateContextI) (string, error) {
	contractObj := getSmartContract(t.ToClientID)
	if contractObj != nil {
		var smartContractData sci.SmartContractTransactionData
		dataBytes := []byte(t.TransactionData)
		err := json.Unmarshal(dataBytes, &smartContractData)
		if err != nil {
			logging.Logger.Error("Error while decoding the JSON from transaction", zap.Any("input", t.TransactionData), zap.Any("error", err))
			return "", err
		}
		// transactionOutput, err := contractObj.ExecuteWithStats(t, smartContractData.FunctionName, []byte(smartContractData.InputData), balances)
		transactionOutput, err := ExecuteWithStats(contractObj, t, smartContractData.FunctionName, smartContractData.InputData, balances)
		if err != nil {
			return "", err
		}
		return transactionOutput, nil
	}
	return "", common.NewError("invalid_smart_contract_address", "Invalid Smart Contract address")
}
