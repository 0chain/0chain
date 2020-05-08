package minersc

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/asaskevich/govalidator"
	"github.com/rcrowley/go-metrics"
)

const (
	//ADDRESS address of minersc
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
	owner   = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
	name    = "miner"
)

var (
	PhaseRounds = make(map[int]int64)
	phaseFuncs  = make(map[int]phaseFunctions)

	lockPhaseFunctions = map[int]*sync.Mutex{
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

//MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*sci.SmartContract
	bcContext sci.BCContextI

	mutexMinerMPK          sync.RWMutex
	smartContractFunctions map[string]smartContractFunction
	callbackPhase          func(int)
}

func (msc *MinerSmartContract) InitSC() {
	if msc.smartContractFunctions == nil {
		msc.smartContractFunctions = make(map[string]smartContractFunction)
	}
	phaseFuncs[Start] = msc.createDKGMinersForContribute
	phaseFuncs[Contribute] = msc.widdleDKGMinersForShare
	phaseFuncs[Publish] = msc.createMagicBlockForWait

	PhaseRounds[Start] = config.SmartContractConfig.GetInt64("smart_contracts.minersc.start_rounds")
	PhaseRounds[Contribute] = config.SmartContractConfig.GetInt64("smart_contracts.minersc.contribute_rounds")
	PhaseRounds[Share] = config.SmartContractConfig.GetInt64("smart_contracts.minersc.share_rounds")
	PhaseRounds[Publish] = config.SmartContractConfig.GetInt64("smart_contracts.minersc.publish_rounds")
	PhaseRounds[Wait] = config.SmartContractConfig.GetInt64("smart_contracts.minersc.wait_rounds")

	moveFunctions[Start] = msc.moveToContribute
	moveFunctions[Contribute] = msc.moveToShareOrPublish
	moveFunctions[Share] = msc.moveToShareOrPublish
	moveFunctions[Publish] = msc.moveToWait
	moveFunctions[Wait] = msc.moveToStart

	msc.smartContractFunctions["add_miner"] = msc.AddMiner
	msc.smartContractFunctions["add_sharder"] = msc.AddSharder
	msc.smartContractFunctions["payFees"] = msc.payFees
	msc.smartContractFunctions["addToDelegatePool"] = msc.addToDelegatePool
	msc.smartContractFunctions["deleteFromDelegatePool"] = msc.deleteFromDelegatePool
	msc.smartContractFunctions["releaseFromDelegatePool"] = msc.releaseFromDelegatePool
	msc.smartContractFunctions["contributeMpk"] = msc.contributeMpk
	msc.smartContractFunctions["shareSignsOrShares"] = msc.shareSignsOrShares
	msc.smartContractFunctions["sharder_keep"] = msc.sharderKeep

}

func (msc *MinerSmartContract) GetName() string {
	return name
}

func (msc *MinerSmartContract) GetAddress() string {
	return ADDRESS
}

func (msc *MinerSmartContract) GetRestPoints() map[string]sci.SmartContractRestHandler {
	return msc.RestHandlers
}

//SetSC setting up smartcontract. implementing the interface
func (msc *MinerSmartContract) SetSC(sc *sci.SmartContract, bcContext sci.BCContextI) {
	msc.SmartContract = sc
	msc.SmartContract.RestHandlers["/getNodepool"] = msc.GetNodepoolHandler
	msc.SmartContract.RestHandlers["/getUserPools"] = msc.GetUserPoolsHandler
	msc.SmartContract.RestHandlers["/getPoolsStats"] = msc.GetPoolStatsHandler
	msc.SmartContract.RestHandlers["/getMinerList"] = msc.GetMinerListHandler
	msc.SmartContract.RestHandlers["/getSharderList"] = msc.GetSharderListHandler
	msc.SmartContract.RestHandlers["/getSharderKeepList"] = msc.GetSharderKeepListHandler
	msc.SmartContract.RestHandlers["/getPhase"] = msc.GetPhaseHandler
	msc.SmartContract.RestHandlers["/getDkgList"] = msc.GetDKGMinerListHandler
	msc.SmartContract.RestHandlers["/getMpksList"] = msc.GetMinersMpksListHandler
	msc.SmartContract.RestHandlers["/getGroupShareOrSigns"] = msc.GetGroupShareOrSignsHandler
	msc.SmartContract.RestHandlers["/getMagicBlock"] = msc.GetMagicBlockHandler
	msc.bcContext = bcContext
	msc.SmartContractExecutionStats["add_miner"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_miner"), nil)
	msc.SmartContractExecutionStats["add_sharder"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_sharder"), nil)
	msc.SmartContractExecutionStats["viewchange_req"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "viewchange_req"), nil)
	msc.SmartContractExecutionStats["payFees"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "payFees"), nil)
	msc.SmartContractExecutionStats["feesPaid"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", msc.ID, "feesPaid"), nil, metrics.NewUniformSample(1024))
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", msc.ID, "mintedTokens"), nil, metrics.NewUniformSample(1024))
}

//Execute implementing the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction, funcName string,
	input []byte, balances c_state.StateContextI) (string, error) {
	gn, err := msc.getGlobalNode(balances)
	if err != nil {
		return "", err
	}
	if lock, ok := lockSmartContractExecute[funcName]; ok {
		lock.Lock()
		defer lock.Unlock()
	}
	scFunc, found := msc.smartContractFunctions[funcName]
	if !found {
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
	return scFunc(t, input, gn, balances)
}

func getHostnameAndPort(burl string) (string, int, error) {
	hostName := ""
	port := 0

	//ToDo: does rudimentary checks. Add more checks
	u, err := url.Parse(burl)
	if err != nil {
		return hostName, port, errors.New(burl + " is not a valid url. " + err.Error())
	}

	if u.Scheme != "http" { //|| u.scheme == "https"  we don't support
		return hostName, port, errors.New(burl + " is not a valid url. It does not have scheme http")
	}

	sp := u.Port()
	if sp == "" {
		return hostName, port, errors.New(burl + " is not a valid url. It does not have port number")
	}

	p, err := strconv.Atoi(sp)
	if err != nil {
		return hostName, port, errors.New(burl + " is not a valid url. " + err.Error())
	}

	hostName = u.Hostname()

	if govalidator.IsDNSName(hostName) || govalidator.IsIPv4(hostName) {
		return hostName, p, nil
	}

	Logger.Info("Both IsDNSName and IsIPV4 returned false for " + hostName)
	return "", 0, errors.New(burl + " is not a valid url. It not a valid IP or valid DNS name")
}

func (msc *MinerSmartContract) getGlobalNode(balances c_state.StateContextI) (*globalNode, error) {
	gn := &globalNode{}
	gv, err := balances.GetTrieNode(GlobalNodeKey)
	if err == nil {
		err := gn.Decode(gv.Encode())
		if err != nil {
			return nil, err
		}
		return gn, nil
	}
	gn.InterestRate = config.SmartContractConfig.GetFloat64("smart_contracts.minersc.interest_rate")
	gn.MinStake = config.SmartContractConfig.GetInt64("smart_contracts.minersc.min_stake")
	gn.MaxStake = config.SmartContractConfig.GetInt64("smart_contracts.minersc.max_stake")
	gn.MaxN = config.SmartContractConfig.GetInt("smart_contracts.minersc.max_n")
	gn.TPercent = config.SmartContractConfig.GetFloat64("smart_contracts.minersc.t_percent")
	gn.KPercent = config.SmartContractConfig.GetFloat64("smart_contracts.minersc.k_percent")
	return gn, nil
}

func (msc *MinerSmartContract) getUserNode(id string, balances c_state.StateContextI) (*UserNode, error) {
	un := NewUserNode()
	un.ID = id
	us, err := balances.GetTrieNode(un.GetKey())
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}
	if us == nil {
		return un, nil
	}
	un.Decode(us.Encode())
	return un, err
}

func (msc *MinerSmartContract) SetCallbackPhase(CallbackPhase func(int)) {
	msc.callbackPhase = CallbackPhase
}
