package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"
)

func (msc *MinerSmartContract) minerHealthCheck(t *transaction.Transaction,
	_ []byte, gn *GlobalNode, balances cstate.StateContextI) (resp string, err error) {
	mn, err := getMinerNode(t.ClientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("miner_health_check_failed",
			"can't get the miner "+t.ClientID+": "+err.Error())
	}

	if mn == nil {
		return "", common.NewError("miner_health_check_failed",
			"can't get the miner "+t.ClientID+": "+err.Error())
	}

	//TODO move it to config
	downtime := common.Downtime(mn.LastHealthCheck, t.CreationDate, gn.MustBase().HealthCheckPeriod)
	mn.LastHealthCheck = t.CreationDate
	emitMinerHealthCheck(mn, downtime, balances)

	if err := mn.save(balances); err != nil {
		return "", common.NewError("miner_health_check_failed",
			"can't save miner: "+err.Error())
	}

	return string(mn.Encode()), nil
}

func (msc *MinerSmartContract) sharderHealthCheck(t *transaction.Transaction,
	_ []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	sn, err := msc.getSharderNode(t.ClientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("sharder_health_check_failed",
			"can't get the sharder "+t.ClientID+": "+err.Error())
	}

	if sn == nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't get the sharder "+t.ClientID+": "+err.Error())
	}

	downtime := common.Downtime(sn.LastHealthCheck, t.CreationDate, gn.MustBase().HealthCheckPeriod)
	sn.LastHealthCheck = t.CreationDate
	emitSharderHealthCheck(sn, downtime, balances)

	if err := sn.save(balances); err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't save sharder: "+err.Error())
	}

	return string(sn.Encode()), nil
}
