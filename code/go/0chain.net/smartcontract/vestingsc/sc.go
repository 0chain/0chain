package vestingsc

import (
	"fmt"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	owner   = "76d4cbf1a4de7cc9d002895048655d60227af6c77d86dab9beed1193725bdd13"
	ADDRESS = "2bba5b05949ea59c80aed3ac3474d7379d3be737e8eb5a968c52295e48333ead"
)

type RestPoints = map[string]smartcontractinterface.SmartContractRestHandler

type VestingSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (vsc *VestingSmartContract) GetName() string {
	return "vesting"
}

func (vsc *VestingSmartContract) GetAddress() string {
	return ADDRESS
}

func (vsc *VestingSmartContract) GetRestPoints() RestPoints {
	return vsc.RestHandlers
}

func (vsc *VestingSmartContract) SetSC(sc *smartcontractinterface.SmartContract,
	bcContext smartcontractinterface.BCContextI) {

	vsc.SmartContract = sc

	// information (statistics) and configurations
	vsc.SmartContract.RestHandlers["/getPoolInfo"] = vsc.getPoolInfo
	vsc.SmartContract.RestHandlers["/getConfig"] = vsc.getConfig

	// update vesting pool config
	vsc.SmartContractExecutionStats["update_config"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "update"), nil)

	// create the vesting pool
	vsc.SmartContractExecutionStats["create"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "create"), nil)

	// tokens lock/unlock
	vsc.SmartContractExecutionStats["lock"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "lock"), nil)
	vsc.SmartContractExecutionStats["unlock"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "unlock"), nil)

	// add/replace/delete {start,duration,friquency,amount,[destinations]}
	vsc.SmartContractExecutionStats["add"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "add"), nil)
	vsc.SmartContractExecutionStats["replace"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "replace"), nil)
	vsc.SmartContractExecutionStats["delete"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "delete"), nil)
}

func (vsc *VestingSmartContract) Execute(t *transaction.Transaction,
	function string, input []byte, balances chainstate.StateContextI) (
	resp string, err error) {

	switch function {

	case "trigger":
		resp, err = vsc.trigger(t, input, balances)

	case "create":
		resp, err = vcs.create(t, input, balances)
	case "lock":
		resp, err = vcs.lock(t, input, balances)
	case "unlock":
		resp, err = vcs.unlock(t, input, balances)

	case "add":
		resp, err = vcs.add(t, input, balances)
	case "replace":
		resp, err = vcs.replace(t, input, balances)
	case "delete":
		resp, err = vcs.delete(t, input, balances)

	case "update_config":
		resp, err = vcs.updateConfig(t, input, balances)

	default:
		err = common.NewError("vesting_sc_failed",
			fmt.Sprintf("no function with %q name", function))
	}
	return
}
