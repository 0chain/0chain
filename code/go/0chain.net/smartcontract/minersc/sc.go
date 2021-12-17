package minersc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"0chain.net/chaincore/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
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

func (ipsc *MinerSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return ipsc.SmartContract.HandlerStats(ctx, params)
}

func (ipsc *MinerSmartContract) GetExecutionStats() map[string]interface{} {
	return ipsc.SmartContractExecutionStats
}

func (msc *MinerSmartContract) GetRestPoints() map[string]sci.SmartContractRestHandler {
	return msc.RestHandlers
}

//setSC setting up smartcontract. implementing the interface
func (msc *MinerSmartContract) setSC(sc *sci.SmartContract, bcContext sci.BCContextI) {
	msc.SmartContract = sc
	msc.SmartContract.RestHandlers["/globalSettings"] = msc.getGlobalsHandler
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

	msc.SmartContract.RestHandlers["/getEvents"] = msc.GetEventsHandler

	msc.SmartContract.RestHandlers["/nodeStat"] = msc.nodeStatHandler
	msc.SmartContract.RestHandlers["/nodePoolStat"] = msc.nodePoolStatHandler
	msc.SmartContract.RestHandlers["/configs"] = msc.configHandler

	msc.bcContext = bcContext
	msc.SmartContractExecutionStats["add_miner"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_miner"), nil)
	msc.SmartContractExecutionStats["add_sharder"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_sharder"), nil)
	msc.SmartContractExecutionStats["miner_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "miner_health_check"), nil)
	msc.SmartContractExecutionStats["sharder_health_check"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "sharder_health_check"), nil)
	msc.SmartContractExecutionStats["update_global_settings"] = metrics.GetOrRegisterCounter(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_globals"), nil)
	msc.SmartContractExecutionStats["update_miner_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_miner_settings"), nil)
	msc.SmartContractExecutionStats["update_sharder_settings"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "update_sharder_settings"), nil)
	msc.SmartContractExecutionStats["payFees"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "payFees"), nil)
	msc.SmartContractExecutionStats["feesPaid"] = metrics.GetOrRegisterCounter("feesPaid", nil)
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.GetOrRegisterCounter("mintedTokens", nil)
}

func (msc *MinerSmartContract) addMint(gn *GlobalNode, mint state.Balance) {
	gn.Minted += mint

	var mintStatsRaw, found = msc.SmartContractExecutionStats["mintedTokens"]
	if found {
		var mintStats = mintStatsRaw.(metrics.Counter)
		mintStats.Inc(int64(mint))
	}
}

//Execute implementing the interface
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

func getGlobalNode(
	balances cstate.StateContextI,
) (gn *GlobalNode, err error) {
	gn = new(GlobalNode)
	var p util.Serializable
	p, err = balances.GetTrieNode(GlobalNodeKey)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		gn.readConfig()
		if err := gn.validate(); err != nil {
			return nil, fmt.Errorf("validating global node: %v", err)
		}
		return gn, nil
	}

	if err = gn.Decode(p.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
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
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return un, nil
}
