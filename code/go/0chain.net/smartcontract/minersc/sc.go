package minersc

import (
	"0chain.net/smartcontract"
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

//MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*sci.SmartContract
	bcContext sci.BCContextI

	mutexMinerMPK          sync.RWMutex
	smartContractFunctions map[string]smartContractFunction
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
	msc.SmartContract.RestHandlers["/getMinerList"] = msc.GetMinerListHandler
	msc.SmartContract.RestHandlers["/getSharderList"] = msc.GetSharderListHandler
	msc.SmartContract.RestHandlers["/getSharderKeepList"] = msc.GetSharderKeepListHandler
	msc.SmartContract.RestHandlers["/getPhase"] = msc.GetPhaseHandler
	msc.SmartContract.RestHandlers["/getDkgList"] = msc.GetDKGMinerListHandler
	msc.SmartContract.RestHandlers["/getMpksList"] = msc.GetMinersMpksListHandler
	msc.SmartContract.RestHandlers["/getGroupShareOrSigns"] = msc.GetGroupShareOrSignsHandler
	msc.SmartContract.RestHandlers["/getMagicBlock"] = msc.GetMagicBlockHandler

	msc.SmartContract.RestHandlers["/nodeStat"] = msc.nodeStatHandler
	msc.SmartContract.RestHandlers["/nodePoolStat"] = msc.nodePoolStatHandler
	msc.SmartContract.RestHandlers["/configs"] = msc.configsHandler

	msc.bcContext = bcContext
	msc.SmartContractExecutionStats["add_miner"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_miner"), nil)
	msc.SmartContractExecutionStats["add_sharder"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_sharder"), nil)
	msc.SmartContractExecutionStats["miner_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "miner_health_check"), nil)
	msc.SmartContractExecutionStats["sharder_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "sharder_health_check"), nil)
	msc.SmartContractExecutionStats["update_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_settings"), nil)
	msc.SmartContractExecutionStats["payFees"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "payFees"), nil)
	msc.SmartContractExecutionStats["feesPaid"] = metrics.GetOrRegisterCounter("feesPaid", nil)
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.GetOrRegisterCounter("mintedTokens", nil)
}

func (msc *MinerSmartContract) addMint(gn *GlobalNode, mint state.Balance) {
	gn.Minted += mint

	var mintStatsRaw, found = msc.SmartContractExecutionStats["mintedTokens"]
	if !found {
		panic("missing mintedTokens stat in miner SC")
	}
	var mintStats = mintStatsRaw.(metrics.Counter)
	mintStats.Inc(int64(mint))
}

//Execute implementing the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction,
	funcName string, input []byte, balances cstate.StateContextI) (
	string, error) {

	gn, err := msc.getGlobalNode(balances)
	if err != nil {
		return "", common.NewError("failed_to_get_global_node", err.Error())
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

func (msc *MinerSmartContract) getGlobalNode(balances cstate.StateContextI) (
	gn *GlobalNode, err error) {

	gn = new(GlobalNode)
	var p util.Serializable
	p, err = balances.GetTrieNode(GlobalNodeKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}

	if err == nil {
		if err = gn.Decode(p.Encode()); err != nil {
			return nil, fmt.Errorf(
				"invalid state: decoding global node: %v", err)
		}
		return gn, nil
	}

	err = nil // reset the value not present error

	const pfx = "smart_contracts.minersc."
	var conf = config.SmartContractConfig
	gn.MinStake = state.Balance(conf.GetFloat64(pfx+"min_stake") * 1e10)
	gn.MaxStake = state.Balance(conf.GetFloat64(pfx+"max_stake") * 1e10)
	gn.MaxN = conf.GetInt(pfx + "max_n")
	gn.MinN = conf.GetInt(pfx + "min_n")
	gn.TPercent = conf.GetFloat64(pfx + "t_percent")
	gn.KPercent = conf.GetFloat64(pfx + "k_percent")
	gn.MaxS = conf.GetInt(pfx + "max_s")
	gn.MinS = conf.GetInt(pfx + "min_s")
	gn.MaxDelegates = conf.GetInt(pfx + "max_delegates")
	gn.RewardRoundFrequency = conf.GetInt64(pfx + "reward_round_frequency")

	// check bounds
	if gn.MinN < 1 {
		return nil, fmt.Errorf("min_n is too small: %d", gn.MinN)
	}
	if gn.MaxN < gn.MinN {
		return nil, fmt.Errorf("max_n is less than min_n: %d < %d",
			gn.MaxN, gn.MinN)
	}

	if gn.MinS < 1 {
		return nil, fmt.Errorf("min_s is too small: %d", gn.MinS)
	}
	if gn.MaxS < gn.MinS {
		return nil, fmt.Errorf("max_s is less than min_s: %d < %d",
			gn.MaxS, gn.MinS)
	}

	if gn.MaxDelegates <= 0 {
		return nil, fmt.Errorf("max_delegates is too small: %d", gn.MaxDelegates)
	}

	gn.InterestRate = conf.GetFloat64(pfx + "interest_rate")
	gn.RewardRate = conf.GetFloat64(pfx + "reward_rate")
	gn.ShareRatio = conf.GetFloat64(pfx + "share_ratio")
	gn.BlockReward = state.Balance(conf.GetFloat64(pfx+"block_reward") * 1e10)
	gn.MaxCharge = conf.GetFloat64(pfx + "max_charge")
	gn.Epoch = conf.GetInt64(pfx + "epoch")
	gn.RewardDeclineRate = conf.GetFloat64(pfx + "reward_decline_rate")
	gn.InterestDeclineRate = conf.GetFloat64(pfx + "interest_decline_rate")
	gn.MaxMint = state.Balance(conf.GetFloat64(pfx+"max_mint") * 1e10)

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
	err = un.Decode(us.Encode())
	if err != nil {
		return nil, smartcontract.NewError(smartcontract.DecodingErr, err)
	}
	return un, nil
}
