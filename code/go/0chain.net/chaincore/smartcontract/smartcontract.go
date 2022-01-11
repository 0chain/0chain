package smartcontract

import (
	"context"
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
//var ContractMap = map[string]sci.SmartContractInterface{}

//ExecuteRestAPI - executes the rest api on the smart contract
func ExecuteRestAPI(ctx context.Context, scAdress string, restpath string, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	scI, err := GetSmartContract(scAdress)
	if err != nil {
		return nil, err
	}

	//add bc context here
	handler, restpathok := scI.GetRestPoints()[restpath]
	if !restpathok {
		return nil, common.NewError("invalid_path", "Invalid path")
	}
	return handler(ctx, params, balances)
}

func ExecuteStats(ctx context.Context, scAdress string, params url.Values, w http.ResponseWriter) {
	scI, err := GetSmartContract(scAdress)
	if err != nil {
		logging.Logger.Error("execute stats failed in getting smart contract", zap.Error(err))

		fmt.Fprintf(w, "invalid_sc: Invalid Smart contract address")
		return
	}

	i, err := scI.GetHandlerStats(ctx, params)
	if err != nil {
		logging.Logger.Warn("unexpected error", zap.Error(err))
		fmt.Fprintf(w, "unexpected error, failed to get stats handler")
		return
	}

	fmt.Fprintf(w, "%v", i)
	return
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
func ExecuteSmartContract(t *transaction.Transaction, scData *sci.SmartContractTransactionData, balances c_state.StateContextI) (string, error) {
	contractObj, err := GetSmartContract(t.ToClientID)
	if err != nil {
		return "", common.NewError("invalid_smart_contract_address", err.Error())
	}

	// transactionOutput, err := contractObj.ExecuteWithStats(t, smartContractData.FunctionName, []byte(smartContractData.InputData), balances)
	transactionOutput, err := ExecuteWithStats(contractObj, t, scData.FunctionName, scData.InputData, balances)
	if err != nil {
		return "", err
	}
	return transactionOutput, nil
}
