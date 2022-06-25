package storagesc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/smartcontract"

	"github.com/rcrowley/go-metrics"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
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

func (ipsc *StorageSmartContract) GetCost(t *transaction.Transaction, funcName string, balances chainstate.StateContextI) (int, error) {
	conf, err := ipsc.getConfig(balances, true)
	if err != nil {
		return math.MaxInt32, err
	}
	if conf.Cost == nil {
		return math.MaxInt32, errors.New("can't get cost")
	}
	cost, ok := conf.Cost[funcName]
	if !ok {
		logging.Logger.Error("no cost given", zap.Any("funcName", funcName))
		return math.MaxInt32, errors.New("no cost given for " + funcName)
	}
	return cost, nil
}

func (ssc *StorageSmartContract) setSC(sc *sci.SmartContract, _ sci.BCContextI) {
	ssc.SmartContract = sc
	// sc configurations
	ssc.SmartContractExecutionStats["update_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_settings"), nil)
	// reading / writing
	ssc.SmartContractExecutionStats["read_redeem"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_redeem"), nil)
	ssc.SmartContractExecutionStats["commit_connection"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "commit_connection"), nil)
	// allocation
	ssc.SmartContractExecutionStats["new_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_allocation_request"), nil)
	ssc.SmartContractExecutionStats["update_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_allocation_request"), nil)
	ssc.SmartContractExecutionStats["finalize_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "finalize_allocation"), nil)
	ssc.SmartContractExecutionStats["cancel_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "cancel_allocation"), nil)
	ssc.SmartContractExecutionStats["free_allocation_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "free_allocation_request"), nil)
	ssc.SmartContractExecutionStats["free_update_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_free_storage"), nil)
	ssc.SmartContractExecutionStats["add_curator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_curator"), nil)
	ssc.SmartContractExecutionStats["curator_transfer_allocation"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "curator_transfer_allocation"), nil)
	// challenge
	ssc.SmartContractExecutionStats["challenge_request"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_request"), nil)
	ssc.SmartContractExecutionStats["challenge_response"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "challenge_response"), nil)
	ssc.SmartContractExecutionStats["generate_challenges"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "generate_challenges"), nil)
	ssc.SmartContractExecutionStats["generate_challenge"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "generate_challenge"), nil)
	// validator
	ssc.SmartContractExecutionStats["add_validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator (add/update SC function)"), nil)
	ssc.SmartContractExecutionStats["update_validator_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_validator_settings"), nil)
	// validators stat (not function calls)
	ssc.SmartContractExecutionStats[statAddValidator] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_validator"), nil)
	ssc.SmartContractExecutionStats[statUpdateValidator] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_validator"), nil)
	ssc.SmartContractExecutionStats[statNumberOfValidators] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "number of validators"), nil)
	// blobber
	ssc.SmartContractExecutionStats["add_blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "add_blobber (add/update/remove SC function)"), nil)
	ssc.SmartContractExecutionStats["update_blobber_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "update_blobber_settings"), nil)
	ssc.SmartContractExecutionStats["blobber_block_rewards"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "blobber_block_rewards"), nil)

	ssc.SmartContractExecutionStats["shut-down-blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "shut-down-blobber"), nil)
	ssc.SmartContractExecutionStats["kill-blobber"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "kill-blobber"), nil)
	ssc.SmartContractExecutionStats["blobber_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "blobber_health_check"), nil)
	ssc.SmartContractExecutionStats["validator-health-check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "validator-health-check"), nil)
	ssc.SmartContractExecutionStats["shut-down-validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "shut-down-validator"), nil)
	ssc.SmartContractExecutionStats["kill-validator"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "kill-validator"), nil)
	// blobber statistic (not function calls)
	ssc.SmartContractExecutionStats[statNumberOfBlobbers] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: number of blobbers"), nil)
	ssc.SmartContractExecutionStats[statAddBlobber] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: add bblober"), nil)
	ssc.SmartContractExecutionStats[statUpdateBlobber] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: update blobber"), nil)
	ssc.SmartContractExecutionStats[statRemoveBlobber] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stat: remove blobber"), nil)
	// read pool
	ssc.SmartContractExecutionStats["new_read_pool"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "new_read_pool"), nil)
	ssc.SmartContractExecutionStats["read_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_pool_lock"), nil)
	ssc.SmartContractExecutionStats["read_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "read_pool_unlock"), nil)
	// write pool
	ssc.SmartContractExecutionStats["write_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "write_pool_lock"), nil)
	ssc.SmartContractExecutionStats["write_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "write_pool_unlock"), nil)
	// stake pool
	ssc.SmartContractExecutionStats["stake_pool_lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stake_pool_lock"), nil)
	ssc.SmartContractExecutionStats["stake_pool_unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stake_pool_unlock"), nil)
	ssc.SmartContractExecutionStats["stake_pool_pay_interests"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "stake_pool_pay_interests"), nil)
	ssc.SmartContractExecutionStats["pay_reward"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ssc.ID, "pay_reward (add/update/remove SC function)"), nil)
}

func (ssc *StorageSmartContract) GetName() string {
	return name
}

func (ssc *StorageSmartContract) GetAddress() string {
	return ADDRESS
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

	case "commit_connection":
		resp, err = sc.commitBlobberConnection(t, input, balances)
		if err != nil {
			return
		}

	// allocations

	case "new_allocation_request":
		resp, err = sc.newAllocationRequest(t, input, balances, nil)
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
	case "update_blobber_settings":
		resp, err = sc.updateBlobberSettings(t, input, balances)
	case "update_validator_settings":
		resp, err = sc.updateValidatorSettings(t, input, balances)
	case "blobber_block_rewards":
		err = sc.blobberBlockRewards(balances)
	case "blobber_health_check":
		resp, err = sc.blobberHealthCheck(t, input, balances)
	case "shut-down-blobber":
		_, err = sc.shutDownBlobber(t, input, balances)
	case "kill-blobber":
		_, err = sc.killBlobber(t, input, balances)

	case "add_validator":
		resp, err = sc.addValidator(t, input, balances)
	case "validator-health-check":
		resp, err = sc.validatorHealthCheck(t, input, balances)
	case "shut-down-validator":
		_, err = sc.shutDownValidator(t, input, balances)
	case "kill-validator":
		_, err = sc.killValidator(t, input, balances)

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
	case "collect_reward":
		resp, err = sc.collectReward(t, input, balances)
	case "generate_challenge":
		challengesEnabled := config.SmartContractConfig.GetBool(
			"smart_contracts.storagesc.challenge_enabled")
		if challengesEnabled {
			err = sc.generateChallenge(t, balances.GetBlock(), input, balances)
			if err != nil {
				return
			}
		} else {
			return "OpenChallenges disabled in the config", nil
		}
		return "OpenChallenges generated", nil

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
