package smartcontract

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

// ContractMap - stores the map of valid smart contracts mapping from its address to its interface implementation
var ContractMap = map[string]sci.SmartContractInterface{}

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

func ExecuteWithStats(smcoi sci.SmartContractInterface, txn *transaction.Transaction, balances c_state.StateContextI) (string, error) {
	ts := time.Now()
	inter, err := smcoi.Execute(txn.Clone(), txn.FunctionName, txn.InputData, balances)
	if err == nil {
		if tm := smcoi.GetExecutionStats()[txn.FunctionName]; tm != nil {
			if timer, ok := tm.(metrics.Timer); ok {
				timer.Update(time.Since(ts))
			}
		}
	}
	return inter, err
}

// ExecuteSmartContract - executes the smart contract in the context of the given transaction
func ExecuteSmartContract(txn *transaction.Transaction, balances c_state.StateContextI) (string, error) {
	contractObj := getSmartContract(txn.ToClientID)
	if contractObj != nil {
		transactionOutput, err := ExecuteWithStats(contractObj, txn, balances)
		if err != nil {
			return "", err
		}
		return transactionOutput, nil
	}
	return "", common.NewError("invalid_smart_contract_address", "Invalid Smart Contract address")
}

func EstimateTransactionCost(t *transaction.Transaction, scData sci.SmartContractTransactionData, balances c_state.StateContextI) (int, error) {
	contractObj := getSmartContract(t.ToClientID)
	if contractObj == nil {
		return 0, errors.New("estimate transaction cost - invalid to client id")
	}

	table, err := contractObj.GetCostTable(balances)
	if err != nil {
		return math.MaxInt, err
	}
	cost, ok := table[strings.ToLower(scData.FunctionName)]
	if !ok {
		//TODO figure out what to do with such transactions, do not return err now for backward compatibility
		//return math.MaxInt, errors.New("no cost found for function")
		return math.MaxInt, nil
	}
	return cost, nil
}

func GetTransactionCostTable(balances c_state.StateContextI) map[string]map[string]int {
	res := make(map[string]map[string]int)
	for addr, sc := range ContractMap {
		table, err := sc.GetCostTable(balances)
		if err != nil {
			logging.Logger.Error("no cost found", zap.String("addr", addr), zap.Error(err))
		}
		res[addr] = table
	}
	return res
}
