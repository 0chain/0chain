package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"
)

func (msc *MinerSmartContract) minerHealthCheck(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	all, err := getMinersList(balances)
	if err != nil {
		return "", common.NewError("miner_health_check_failed",
			"Failed to get miner list: "+err.Error())
	}

	var existingMiner *MinerNode
	existingMiner, err = getMinerNode(t.ClientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("miner_health_check_failed",
			"can't get the miner "+t.ClientID+": "+err.Error())
	}

	// update the last health check time
	for _, nd := range all.Nodes {
		if nd.ID == t.ClientID {
			nd.LastHealthCheck = t.CreationDate
			// miner does not exist, use the one in the list
			if existingMiner == nil {
				existingMiner = nd
			}
			break
		}
	}

	if existingMiner == nil {
		return "", common.NewError("miner_health_check_failed",
			"can't get the miner "+t.ClientID+": "+err.Error())
	}

	existingMiner.LastHealthCheck = t.CreationDate

	if err = updateMinersList(balances, all); err != nil {
		return "", common.NewError("miner_health_check_failed",
			"can't save all miners list: "+err.Error())
	}

	err = existingMiner.save(balances)
	if err != nil {
		return "", common.NewError("miner_health_check_failed",
			"can't save miner: "+err.Error())
	}

	emitMinerHealthCheck(existingMiner, balances)

	return string(existingMiner.Encode()), nil
}

func (msc *MinerSmartContract) sharderHealthCheck(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	all, err := getAllShardersList(balances)
	if err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"Failed to get sharder list: "+err.Error())
	}

	var existingSharder *MinerNode
	existingSharder, err = msc.getSharderNode(t.ClientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("sharder_health_check_failed",
			"can't get the sharder "+t.ClientID+": "+err.Error())
	}

	for _, nd := range all.Nodes {
		if nd.ID == t.ClientID {
			nd.LastHealthCheck = t.CreationDate
			if existingSharder == nil {
				existingSharder = nd
			}
			break
		}
	}

	if existingSharder == nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't get the sharder "+t.ClientID+": "+err.Error())
	}

	existingSharder.LastHealthCheck = t.CreationDate

	if err = updateAllShardersList(balances, all); err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't save all sharders list: "+err.Error())
	}

	err = existingSharder.save(balances)
	if err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't save sharder: "+err.Error())
	}

	emitSharderHealthCheck(existingSharder, balances)

	return string(existingSharder.Encode()), nil
}
