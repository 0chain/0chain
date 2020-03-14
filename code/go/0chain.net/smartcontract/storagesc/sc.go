package storagesc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	// TODO (sfxdx): what is the real owner?
	owner   = "5be4e0abc645b04fa45895645568c448849ea379a938ca114b5f67a96258dbc2" // a random stub
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	name    = "storage"

	KB = 1024      // kilobyte
	MB = 1024 * KB // megabyte
	GB = 1024 * MB // gigabyte
)

type StorageSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ssc *StorageSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ssc.SmartContract = sc
	// sc configurations
	ssc.SmartContract.RestHandlers["/getConfig"] = ssc.getConfigHandler
	ssc.SmartContractExecutionStats["update_config"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_config"), nil)
	// reading / writing
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContractExecutionStats["read_redeem"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_redeem"), nil)
	ssc.SmartContractExecutionStats["commit_connection"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "commit_connection"), nil)
	// allocation
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/allocations"] = ssc.GetAllocationsHandler
	ssc.SmartContractExecutionStats["new_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_allocation_request"), nil)
	ssc.SmartContractExecutionStats["update_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_allocation_request"), nil)
	// challenge
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
	ssc.SmartContract.RestHandlers["/getchallenge"] = ssc.GetChallengeHandler
	ssc.SmartContractExecutionStats["challenge_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_request"), nil)
	ssc.SmartContractExecutionStats["challenge_response"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_response"), nil)
	// validator
	ssc.SmartContractExecutionStats["add_validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator"), nil)
	// blobber
	ssc.SmartContract.RestHandlers["/getblobbers"] = ssc.GetBlobbersHandler
	ssc.SmartContractExecutionStats["add_blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_blobber"), nil)
	ssc.SmartContractExecutionStats["update_blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_blobber"), nil)
	// read pool
	ssc.SmartContract.RestHandlers["/getReadPoolsStats"] = ssc.getReadPoolsStatsHandler
	ssc.SmartContractExecutionStats["new_read_pool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_read_pool"), nil)
	ssc.SmartContractExecutionStats["read_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_pool_lock"), nil)
	ssc.SmartContractExecutionStats["read_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_pool_unlock"), nil)
	// write pool
	ssc.SmartContract.RestHandlers["/getWritePoolStat"] = ssc.getWritePoolStatHandler
	ssc.SmartContractExecutionStats["write_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "write_pool_lock"), nil)
	ssc.SmartContractExecutionStats["write_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "write_pool_unlock"), nil)
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

func (sc *StorageSmartContract) Execute(t *transaction.Transaction,
	funcName string, input []byte, balances c_state.StateContextI) (
	resp string, err error) {

	switch funcName {

	case "read_redeem":
		if resp, err = sc.commitBlobberRead(t, input, balances); err != nil {
			return
		}
		challengesEnabled := config.SmartContractConfig.GetBool(
			"smart_contracts.storagesc.challenge_enabled")
		if challengesEnabled {
			err = sc.generateChallenges(t, balances.GetBlock(), input, balances)
			if err != nil {
				return "", err
			}
		}

	case "commit_connection":
		resp, err = sc.commitBlobberConnection(t, input, balances)
		if err != nil {
			return
		}

		challengesEnabled := config.SmartContractConfig.GetBool(
			"smart_contracts.storagesc.challenge_enabled")
		if challengesEnabled {
			err = sc.generateChallenges(t, balances.GetBlock(), input, balances)
			if err != nil {
				return "", err
			}
		}

	case "new_allocation_request":
		resp, err = sc.newAllocationRequest(t, input, balances)

	case "update_allocation_request":
		resp, err = sc.updateAllocationRequest(t, input, balances)

	case "add_blobber":
		resp, err = sc.addBlobber(t, input, balances)

	case "add_validator":
		resp, err = sc.addValidator(t, input, balances)

	case "blobber_health_check":
		resp, err = sc.blobberHealthCheck(t, input, balances)

	// read_pool

	case "new_read_pool":
		resp, err = sc.newReadPool(t, input, balances)
	case "read_pool_lock":
		resp, err = sc.readPoolLock(t, input, balances)
	case "read_pool_unlock":
		resp, err = sc.readPoolUnlock(t, input, balances)

	// write pool

	case "write_pool_lock":
		resp, err = sc.writePoolLock(t, input, balances)
	case "write_pool_unlock":
		resp, err = sc.writePoolUnlock(t, input, balances)

	// case "challenge_request":
	// 	resp, err := sc.addChallenge(t, balances.GetBlock(), input, balances)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	return resp, nil

	case "generate_challenges":
		challengesEnabled := config.SmartContractConfig.GetBool(
			"smart_contracts.storagesc.challenge_enabled")
		if challengesEnabled {
			err = sc.generateChallenges(t, balances.GetBlock(), input, balances)
			if err != nil {
				return
			}
		} else {
			return "Challenges disabled in the config", nil
		}
		return "Challenges generated", nil

	case "challenge_response":
		resp, err = sc.verifyChallenge(t, input, balances)

	// configurations

	case "update_config":
		resp, err = sc.updateConfig(t, input, balances)

	default:
		err = common.NewError("invalid_storage_function_name",
			"Invalid storage function called")
	}

	return
}
