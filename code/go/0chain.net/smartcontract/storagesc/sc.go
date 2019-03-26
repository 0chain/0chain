package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

type StorageSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ssc *StorageSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ssc.SmartContract = sc
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
}

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {
	if funcName == "read_redeem" {
		resp, err := sc.commitBlobberRead(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "commit_connection" {
		resp, err := sc.commitBlobberConnection(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "new_allocation_request" {
		resp, err := sc.newAllocationRequest(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "add_blobber" {
		resp, err := sc.addBlobber(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "add_validator" {
		resp, err := sc.addValidator(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "challenge_request" {
		resp, err := sc.addChallenge(t, balances.GetBlock(), input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "challenge_response" {
		resp, err := sc.verifyChallenge(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	return "", common.NewError("invalid_storage_function_name", "Invalid storage function called")
}
