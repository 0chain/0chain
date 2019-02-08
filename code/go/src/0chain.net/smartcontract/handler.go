package smartcontract

import (
	"context"
	"encoding/json"

	"0chain.net/block"
	c_state "0chain.net/chain/state"
	"0chain.net/common"
	"0chain.net/faucetsc"
	. "0chain.net/logging"
	sci "0chain.net/smartcontractinterface"
	"0chain.net/smartcontractstate"
	"0chain.net/storagesc"
	"0chain.net/transaction"
	"0chain.net/zrc20sc"
	"go.uber.org/zap"
)

const (
	STORAGE_CONTRACT_ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	FAUCET_CONTRACT_ADDRESS  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	ZRC20_CONTRACT_ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d5"
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
	case ZRC20_CONTRACT_ADDRESS:
		zrc20SC := &zrc20sc.ZRC20SmartContract{}
		zrc20SC.DB = smartcontractstate.NewSCState(ndb, ZRC20_CONTRACT_ADDRESS)
		zrc20SC.ID = ZRC20_CONTRACT_ADDRESS
		return zrc20SC
	}
	return nil
}

func ExecuteSmartContract(ctx context.Context, t *transaction.Transaction, b *block.Block, ndb smartcontractstate.SCDB, balances c_state.StateContextI) (string, error) {
	contractObj := getSmartContract(t, ndb)
	if contractObj != nil {
		var smartContractData sci.SmartContractTransactionData
		dataBytes := []byte(t.TransactionData)
		err := json.Unmarshal(dataBytes, &smartContractData)
		if err != nil {
			Logger.Error("Error while decoding the JSON from transaction", zap.Any("input", t.TransactionData), zap.Any("error", err))
			return "", err
		}
		transactionOutput, err := contractObj.Execute(t, b, smartContractData.FunctionName, []byte(smartContractData.InputData), balances)
		if err != nil {
			return "", err
		}
		return transactionOutput, nil
	}
	return "", common.NewError("invalid_smart_contract_address", "Invalid Smart Contract address")
}
