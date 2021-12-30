package storagesc

import (
	"context"
	"fmt"
	"net/url"

	"0chain.net/chaincore/smartcontract"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	name    = "storage"

	KB = 1024      // kilobyte
	MB = 1024 * KB // megabyte
	GB = 1024 * MB // gigabyte
)

type StorageSmartContract struct {
	*sci.SmartContract
}

func NewStorageSmartContract() sci.SmartContractInterface {
	var sscCopy = &StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	sscCopy.setSC(sscCopy.SmartContract, &smartcontract.BCContext{})
	return sscCopy
}

func (ipsc *StorageSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return ipsc.SmartContract.HandlerStats(ctx, params)
}

func (ipsc *StorageSmartContract) GetExecutionStats() map[string]interface{} {
	return ipsc.SmartContractExecutionStats
}

func (ssc *StorageSmartContract) setSC(sc *sci.SmartContract, bcContext sci.BCContextI) {
	ssc.SmartContract = sc
	ssc.SmartContract.RestHandlers["/get_mpt_key"] = ssc.GetMptKey
	// sc configurations
	ssc.SmartContract.RestHandlers["/getConfig"] = ssc.getConfigHandler
	ssc.SmartContractExecutionStats["update_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_settings"), nil)
	// reading / writing
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContractExecutionStats["read_redeem"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_redeem"), nil)
	ssc.SmartContractExecutionStats["commit_connection"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "commit_connection"), nil)
	// allocation
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/allocations"] = ssc.GetAllocationsHandler
	ssc.SmartContract.RestHandlers["/allocation_min_lock"] = ssc.GetAllocationMinLockHandler
	ssc.SmartContractExecutionStats["new_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_allocation_request"), nil)
	ssc.SmartContractExecutionStats["update_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_allocation_request"), nil)
	ssc.SmartContractExecutionStats["finalize_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "finalize_allocation"), nil)
	ssc.SmartContractExecutionStats["cancel_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "cancel_allocation"), nil)
	ssc.SmartContractExecutionStats["free_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "free_allocation_request"), nil)
	ssc.SmartContractExecutionStats["free_update_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_free_storage"), nil)
	ssc.SmartContractExecutionStats["add_curator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_curator"), nil)
	ssc.SmartContractExecutionStats["curator_transfer_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "curator_transfer_allocation"), nil)
	// challenge
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
	ssc.SmartContract.RestHandlers["/getchallenge"] = ssc.GetChallengeHandler
	ssc.SmartContractExecutionStats["challenge_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_request"), nil)
	ssc.SmartContractExecutionStats["challenge_response"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_response"), nil)
	ssc.SmartContractExecutionStats["generate_challenges"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "generate_challenges"), nil)
	// validator
	ssc.SmartContractExecutionStats["add_validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator (add/update SC function)"), nil)
	// validators stat (not function calls)
	ssc.SmartContractExecutionStats[statAddValidator] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator"), nil)
	ssc.SmartContractExecutionStats[statUpdateValidator] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_validator"), nil)
	ssc.SmartContractExecutionStats[statNumberOfValidators] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "number of validators"), nil)
	// blobber
	ssc.SmartContract.RestHandlers["/getblobbers"] = ssc.GetBlobbersHandler
	ssc.SmartContract.RestHandlers["/getBlobber"] = ssc.GetBlobberHandler
	ssc.SmartContractExecutionStats["add_blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_blobber (add/update/remove SC function)"), nil)
	ssc.SmartContractExecutionStats["update_blobber_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_blobber_settings"), nil)
	ssc.SmartContractExecutionStats["pay_blobber_block_rewards"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "pay_blobber_block_rewards"), nil)
	// blobber statistic (not function calls)
	ssc.SmartContractExecutionStats[statNumberOfBlobbers] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: number of blobbers"), nil)
	ssc.SmartContractExecutionStats[statAddBlobber] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: add bblober"), nil)
	ssc.SmartContractExecutionStats[statUpdateBlobber] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: update blobber"), nil)
	ssc.SmartContractExecutionStats[statRemoveBlobber] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: remove blobber"), nil)
	// read pool
	ssc.SmartContract.RestHandlers["/getReadPoolStat"] = ssc.getReadPoolStatHandler
	ssc.SmartContract.RestHandlers["/getReadPoolAllocBlobberStat"] = ssc.getReadPoolAllocBlobberStatHandler
	ssc.SmartContractExecutionStats["new_read_pool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_read_pool"), nil)
	ssc.SmartContractExecutionStats["read_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_pool_lock"), nil)
	ssc.SmartContractExecutionStats["read_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_pool_unlock"), nil)
	// write pool
	ssc.SmartContract.RestHandlers["/getWritePoolStat"] = ssc.getWritePoolStatHandler
	ssc.SmartContract.RestHandlers["/getWritePoolAllocBlobberStat"] = ssc.getWritePoolAllocBlobberStatHandler
	ssc.SmartContractExecutionStats["write_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "write_pool_lock"), nil)
	ssc.SmartContractExecutionStats["write_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "write_pool_unlock"), nil)
	// stake pool
	ssc.SmartContract.RestHandlers["/getStakePoolStat"] = ssc.getStakePoolStatHandler
	ssc.SmartContract.RestHandlers["/getUserStakePoolStat"] = ssc.getUserStakePoolStatHandler
	ssc.SmartContractExecutionStats["stake_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stake_pool_lock"), nil)
	ssc.SmartContractExecutionStats["stake_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stake_pool_unlock"), nil)
	ssc.SmartContractExecutionStats["stake_pool_pay_interests"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stake_pool_pay_interests"), nil)
	// challenge pool
	ssc.SmartContract.RestHandlers["/getChallengePoolStat"] = ssc.getChallengePoolStatHandler
}

func (ssc *StorageSmartContract) GetName() string {
	return name
}

func (ssc *StorageSmartContract) GetAddress() string {
	return ADDRESS
}

func (ssc *StorageSmartContract) GetRestPoints() map[string]sci.SmartContractRestHandler {
	return ssc.RestHandlers
}

// stat not belongs to SC function calls

const (
	statAddValidator       = "stat: add validator"
	statUpdateValidator    = "stat: update validator"
	statNumberOfValidators = "stat: number of validators"
	statNumberOfBlobbers   = "stat: number of blobbers"
	statAddBlobber         = "stat: add blobber"
	statUpdateBlobber      = "stat: update blobber"
	statRemoveBlobber      = "stat: remove blobber"
)

func (ssc *StorageSmartContract) statIncr(name string) {
	var (
		metric interface{}
		count  metrics.Counter
		ok     bool
	)
	if metric, ok = ssc.SmartContractExecutionStats[name]; !ok {
		return
	}
	if count, ok = metric.(metrics.Counter); !ok {
		return
	}
	count.Inc(1)
}

func (ssc *StorageSmartContract) statDecr(name string) {
	var (
		metric interface{}
		count  metrics.Counter
		ok     bool
	)
	if metric, ok = ssc.SmartContractExecutionStats[name]; !ok {
		return
	}
	if count, ok = metric.(metrics.Counter); !ok {
		return
	}
	count.Dec(1)
}

// functions execution

func (sc *StorageSmartContract) Execute(t *transaction.Transaction,
	funcName string, input []byte, balances chainstate.StateContextI) (
	resp string, err error) {

	switch funcName {

	// read/write markers

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

	// allocations

	case "new_allocation_request":
		resp, err = sc.newAllocationRequest(t, input, balances)
	case "update_allocation_request":
		resp, err = sc.updateAllocationRequest(t, input, balances)
	case "finalize_allocation":
		resp, err = sc.finalizeAllocation(t, input, balances)
	case "cancel_allocation":
		resp, err = sc.cancelAllocationRequest(t, input, balances)

	// free allocations

	case "add_free_storage_assigner":
		resp, err = sc.addFreeStorageAssigner(t, input, balances)
	case "free_allocation_request":
		resp, err = sc.freeAllocationRequest(t, input, balances)
	case "free_update_allocation":
		resp, err = sc.updateFreeStorageRequest(t, input, balances)
	case "curator_transfer_allocation":
		resp, err = sc.curatorTransferAllocation(t, input, balances)

	//curator
	case "add_curator":
		resp, err = sc.addCurator(t, input, balances)
	case "remove_curator":
		resp, err = sc.removeCurator(t, input, balances)

	// blobbers

	case "add_blobber":
		resp, err = sc.addBlobber(t, input, balances)
	case "add_validator":
		resp, err = sc.addValidator(t, input, balances)
	case "blobber_health_check":
		resp, err = sc.blobberHealthCheck(t, input, balances)
	case "update_blobber_settings":
		resp, err = sc.updateBlobberSettings(t, input, balances)
	case "pay_blobber_block_rewards":
		err = sc.payBlobberBlockRewards(balances)

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

		// stake pool

	case "stake_pool_lock":
		resp, err = sc.stakePoolLock(t, input, balances)
	case "stake_pool_unlock":
		resp, err = sc.stakePoolUnlock(t, input, balances)
	case "stake_pool_pay_interests":
		resp, err = sc.stakePoolPayInterests(t, input, balances)

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

	case "update_settings":
		resp, err = sc.updateSettings(t, input, balances)

	case "commit_settings_changes":
		resp, err = sc.commitSettingChanges(t, input, balances)

	default:
		err = common.NewErrorf("invalid_storage_function_name",
			"Invalid storage function '%s' called", funcName)
	}

	return
}
