package storagesc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	name    = "storage"
)

type StorageSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ssc *StorageSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ssc.SmartContract = sc
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/allocations"] = ssc.GetAllocationsHandler
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
	ssc.SmartContractExecutionStats["read_redeem"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_redeem"), nil)
	ssc.SmartContractExecutionStats["commit_connection"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "commit_connection"), nil)
	ssc.SmartContractExecutionStats["new_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_allocation_request"), nil)
	ssc.SmartContractExecutionStats["add_blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_blobber"), nil)
	ssc.SmartContractExecutionStats["add_validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator"), nil)
	ssc.SmartContractExecutionStats["challenge_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_request"), nil)
	ssc.SmartContractExecutionStats["challenge_response"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_response"), nil)
}

func (ssc *StorageSmartContract) GetName() string {
	return name
}

func (ssc *StorageSmartContract) GetAddress() string {
	return ADDRESS
}

func (ssc *StorageSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return ssc.RestHandlers
}

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {
	if funcName == "read_redeem" {
		resp, err := sc.commitBlobberRead(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "commit_connection" {
		resp, err := sc.commitBlobberConnection(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "new_allocation_request" {
		resp, err := sc.newAllocationRequest(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "add_blobber" {
		resp, err := sc.addBlobber(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "add_validator" {
		resp, err := sc.addValidator(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "challenge_request" {
		resp, err := sc.addChallenge(t, balances.GetBlock(), input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "challenge_response" {
		resp, err := sc.verifyChallenge(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	return "", common.NewError("invalid_storage_function_name", "Invalid storage function called")
}
