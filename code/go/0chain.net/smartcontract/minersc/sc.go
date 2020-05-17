package minersc

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"github.com/asaskevich/govalidator"
	"github.com/rcrowley/go-metrics"

	. "0chain.net/core/logging"
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

	const pfx = "smart_contracts.minersc."
	var scc = config.SmartContractConfig

	PhaseRounds[Start] = scc.GetInt64(pfx + "start_rounds")
	PhaseRounds[Contribute] = scc.GetInt64(pfx + "contribute_rounds")
	PhaseRounds[Share] = scc.GetInt64(pfx + "share_rounds")
	PhaseRounds[Publish] = scc.GetInt64(pfx + "publish_rounds")
	PhaseRounds[Wait] = scc.GetInt64(pfx + "wait_rounds")

	moveFunctions[Start] = msc.moveToContribute
	moveFunctions[Contribute] = msc.moveToShareOrPublish
	moveFunctions[Share] = msc.moveToShareOrPublish
	moveFunctions[Publish] = msc.moveToWait
	moveFunctions[Wait] = msc.moveToStart

	msc.smartContractFunctions["add_miner"] = msc.AddMiner
	msc.smartContractFunctions["add_sharder"] = msc.AddSharder
	msc.smartContractFunctions["update_settings"] = msc.UpdateSettings
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
	msc.SmartContractExecutionStats["update_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_settings"), nil)
	msc.SmartContractExecutionStats["viewchange_req"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "viewchange_req"), nil)
	msc.SmartContractExecutionStats["payFees"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "payFees"), nil)
	msc.SmartContractExecutionStats["feesPaid"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", msc.ID, "feesPaid"), nil, metrics.NewUniformSample(1024))
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", msc.ID, "mintedTokens"), nil, metrics.NewUniformSample(1024))
}

func (msc *MinerSmartContract) addMint(gn *globalNode, mint state.Balance) {
	gn.Minted += mint

	var mintStatsRaw, found = msc.SmartContractExecutionStats["mintedTokens"]
	if !found {
		panic("missing mintedTokens stat in miner SC")
	}
	var mintStats = mintStatsRaw.(metrics.Histogram)
	mintStats.Update(int64(mint))
}

//Execute implementing the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction, funcName string,
	input []byte, balances cstate.StateContextI) (string, error) {
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

func (msc *MinerSmartContract) getGlobalNode(balances cstate.StateContextI) (*globalNode, error) {
	gn := &globalNode{}
	gv, err := balances.GetTrieNode(GlobalNodeKey)
	if err == nil {
		err := gn.Decode(gv.Encode())
		if err != nil {
			return nil, err
		}
		return gn, nil
	}
	const pfx = "smart_contracts.minersc."
	var conf = config.SmartContractConfig
	gn.MinStake = state.Balance(conf.GetFloat64(pfx+"min_stake") * 1e10)
	gn.MaxStake = state.Balance(conf.GetFloat64(pfx+"max_stake") * 1e10)
	gn.MaxN = conf.GetInt(pfx + "max_n")
	gn.TPercent = conf.GetFloat64(pfx + "t_percent")
	gn.KPercent = conf.GetFloat64(pfx + "k_percent")

	gn.InterestRate = conf.GetFloat64("interest_rate")
	gn.RewardRate = conf.GetFloat64("reward_rate")
	gn.ShareRatio = conf.GetFloat64("share_ratio")
	gn.BlockReward = state.Balance(conf.GetFloat64("block_reward") * 1e10)
	gn.MaxCharge = conf.GetFloat64("max_charge")
	gn.Epoch = conf.GetInt64("epoch")
	gn.RewardDeclineRate = conf.GetFloat64("reward_decline_rate")
	gn.InterestDeclineRate = conf.GetFloat64("interest_decline_rate")
	gn.MaxMint = state.Balance(conf.GetFloat64("max_mint") * 1e10)

	return gn, nil
}

func (msc *MinerSmartContract) getUserNode(id string, balances cstate.StateContextI) (*UserNode, error) {
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
