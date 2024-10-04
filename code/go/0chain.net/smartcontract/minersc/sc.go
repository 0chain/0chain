package minersc

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"0chain.net/chaincore/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"

	"github.com/rcrowley/go-metrics"
)

const (
	//ADDRESS address of minersc
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
	name    = "miner"
)

var (
	PhaseRounds = make(map[Phase]int64)
	phaseFuncs  = make(map[Phase]phaseFunctions)

	lockPhaseFunctions = map[Phase]*sync.Mutex{
		Start:      {},
		Contribute: {},
		Publish:    {},
	}

	lockSmartContractExecute = map[string]*sync.Mutex{
		"add_miner":          {},
		"add_sharder":        {},
		"sharder_keep":       {},
		"contributeMpk":      {},
		"shareSignsOrShares": {},
	}
)

// MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*sci.SmartContract
	bcContext sci.BCContextI

	mutexMinerMPK          sync.RWMutex
	smartContractFunctions map[string]smartContractFunction
}

func NewMinerSmartContract() sci.SmartContractInterface {
	var mscCopy = &MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
		bcContext:     &smartcontract.BCContext{},
	}
	mscCopy.initSC()
	mscCopy.setSC(mscCopy.SmartContract, mscCopy.bcContext)
	return mscCopy
}

func (msc *MinerSmartContract) GetName() string {
	return name
}

func (msc *MinerSmartContract) GetAddress() string {
	return ADDRESS
}

func (msc *MinerSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return msc.SmartContract.HandlerStats(ctx, params)
}

func (msc *MinerSmartContract) GetExecutionStats() map[string]interface{} {
	return msc.SmartContractExecutionStats
}

func (msc *MinerSmartContract) GetCostTable(balances cstate.StateContextI) (map[string]int, error) {
	node, err := getGlobalNode(balances)
	if err != nil {
		return map[string]int{}, err
	}
	nb := node.MustBase()
	if nb.Cost == nil {
		return map[string]int{}, err
	}
	return nb.Cost, nil
}

// setSC setting up smartcontract. implementing the interface
func (msc *MinerSmartContract) setSC(sc *sci.SmartContract, bcContext sci.BCContextI) {
	msc.SmartContract = sc

	msc.bcContext = bcContext
	msc.SmartContractExecutionStats["add_miner"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_miner"), nil)
	msc.SmartContractExecutionStats["add_sharder"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_sharder"), nil)
	msc.SmartContractExecutionStats["collect_reward"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "collect_reward"), nil)
	msc.SmartContractExecutionStats["miner_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "miner_health_check"), nil)
	msc.SmartContractExecutionStats["sharder_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "sharder_health_check"), nil)
	msc.SmartContractExecutionStats["update_globals"] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_globals"), nil)
	msc.SmartContractExecutionStats["update_miner_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_miner_settings"), nil)
	msc.SmartContractExecutionStats["update_sharder_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_sharder_settings"), nil)
	msc.SmartContractExecutionStats["payFees"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "payFees"), nil)
	msc.SmartContractExecutionStats["feesPaid"] = metrics.GetOrRegisterCounter("feesPaid", nil)
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.GetOrRegisterCounter("mintedTokens", nil)
}

// Execute implementing the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction,
	funcName string, input []byte, balances cstate.StateContextI) (
	string, error) {

	gn, err := getGlobalNode(balances)
	if err != nil {
		return "", common.NewError("failed_to_get_global_node", err.Error())
	}
	if lock, ok := lockSmartContractExecute[funcName]; ok {
		lock.Lock()
		defer lock.Unlock()
	}
	scFunc, found := msc.smartContractFunctions[funcName]
	if !found {
		return common.NewErrorf("failed execution", "no miner smart contract method with name: %v", funcName).Error(), nil
	}
	return scFunc(t, input, gn, balances)
}

func getGlobalNode(
	balances cstate.CommonStateContextI,
) (gn *GlobalNode, err error) {
	gn = new(GlobalNode)
	err = balances.GetTrieNode(GlobalNodeKey, gn)
	if err != nil {
		return nil, err
	}

	return gn, nil
}

func InitConfig(balances cstate.StateContextI) error {
	gn := new(GlobalNode)
	if err := balances.GetTrieNode(GlobalNodeKey, gn); err != nil {
		if err != util.ErrValueNotPresent {
			return fmt.Errorf("failed to get global node: %w", err)
		}
		if err := gn.readConfig(balances); err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}
		if err := gn.validate(); err != nil {
			return fmt.Errorf("failed to validate global node: %w", err)
		}
		if _, err := balances.InsertTrieNode(GlobalNodeKey, gn); err != nil {
			return fmt.Errorf("failed to insert global node: %w", err)
		}
	}

	if err := initGlobalSettings(balances); err != nil {
		return fmt.Errorf("failed to initialize global settings: %w", err)
	}

	return nil
}

// GetGlobalNode returns the global node config
func GetGlobalNode(balances cstate.CommonStateContextI) (*GlobalNode, error) {
	return getGlobalNode(balances)
}
