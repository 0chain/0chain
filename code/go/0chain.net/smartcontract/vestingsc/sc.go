package vestingsc

import (
	"fmt"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

const ADDRESS = "2bba5b05949ea59c80aed3ac3474d7379d3be737e8eb5a968c52295e48333ead"

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
	vsc.SmartContract.RestHandlers["/getConfig"] = vsc.getConfigHandler
	vsc.SmartContract.RestHandlers["/getPoolInfo"] = vsc.getPoolInfoHandler
	vsc.SmartContract.RestHandlers["/getClientPools"] = vsc.getClientPoolsHandler

	// update vesting pool config
	vsc.SmartContractExecutionStats["update_config"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "update"), nil)

	// add/delete {start,duration,friquency,amount,[destinations]}
	vsc.SmartContractExecutionStats["add"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "add"), nil)
	vsc.SmartContractExecutionStats["delete"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "delete"), nil)

	// tokens lock/unlock for an existing pool
	vsc.SmartContractExecutionStats["lock"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "lock"), nil)
	vsc.SmartContractExecutionStats["unlock"] = metrics.GetOrRegisterTimer(
		fmt.Sprintf("sc:%v:func:%v", vsc.ID, "unlock"), nil)
}

func (vsc *VestingSmartContract) Execute(t *transaction.Transaction,
	function string, input []byte, balances chainstate.StateContextI) (
	resp string, err error) {

	switch function {

	case "trigger":
		resp, err = vsc.trigger(t, input, balances)

	case "lock":
		resp, err = vsc.lock(t, input, balances)
	case "unlock":
		resp, err = vsc.unlock(t, input, balances)

	case "add":
		resp, err = vsc.add(t, input, balances)
	case "delete":
		resp, err = vsc.delete(t, input, balances)

	case "update_config":
		resp, err = vsc.updateConfig(t, input, balances)

	default:
		err = common.NewError("vesting_sc_failed",
			fmt.Sprintf("no function with %q name", function))
	}
	return
}
