package smartcontract

import (
	"context"
	"encoding/json"

	c_state "0chain.net/chain/state"
	"0chain.net/common"
	"0chain.net/faucetsc"
	. "0chain.net/logging"
	sci "0chain.net/smartcontractinterface"
	"0chain.net/smartcontractstate"
	"0chain.net/storagesc"
	"0chain.net/transaction"
	"go.uber.org/zap"
)

const (
	STORAGE_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	FAUCET_CONTRACT_ADDRESS  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
)

func getSmartContract(t *transaction.Transaction, ndb smartcontractstate.SCDB) sci.SmartContractInterface {
	switch t.ToClientID {
	case STORAGE_CONTRACT_ADDRESS:
		storageSC := &storagesc.StorageSmartContract{}
		storageSC.DB = smartcontractstate.NewSCState(ndb, STORAGE_CONTRACT_ADDRESS)
		storageSC.ID = STORAGE_CONTRACT_ADDRESS
		return storageSC
	case FAUCET_CONTRACT_ADDRESS:
		faucetSC := &faucetsc.FaucetSmartContract{}
		faucetSC.DB = smartcontractstate.NewSCState(ndb, FAUCET_CONTRACT_ADDRESS)
		faucetSC.ID = FAUCET_CONTRACT_ADDRESS
		return faucetSC
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
