package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (msc *MinerSmartContract) minerHealthCheck(t *transaction.Transaction,
	_ []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	all, err := getMinersList(balances)
	if err != nil {
		return "", common.NewError("miner_health_check_failed",
			"Failed to get miner list: "+err.Error())
	}

	var existingMiner *MinerNode
	if existingMiner, err = getMinerNode(t.ClientID, balances); err != nil {
		return "", common.NewError("miner_health_check_failed",
			"can't get the miner "+t.ClientID+": "+err.Error())
	}

	existingMiner.LastHealthCheck = t.CreationDate

	for _, nodes := range all.Nodes {
		if nodes.ID == t.ClientID {
			nodes.LastHealthCheck = t.CreationDate
			break
		}
	}

	if err = updateMinersList(balances, all); err != nil {
		return "", common.NewError("miner_health_check_failed",
			"can't save all miners list: "+err.Error())
	}

	err = existingMiner.save(balances)
	if err != nil {
		return "", common.NewError("miner_health_check_failed",
			"can't save miner: "+err.Error())
	}

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
	if existingSharder, err = msc.getSharderNode(t.ClientID, balances); err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't get the sharder "+t.ClientID+": "+err.Error())
	}

	existingSharder.LastHealthCheck = t.CreationDate

	for _, nodes := range all.Nodes {
		if nodes.ID == t.ClientID {
			nodes.LastHealthCheck = t.CreationDate
			break
		}
	}

	if err = updateAllShardersList(balances, all); err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't save all sharders list: "+err.Error())
	}

	err = existingSharder.save(balances)
	if err != nil {
		return "", common.NewError("sharder_health_check_failed",
			"can't save sharder: "+err.Error())
	}

	return string(existingSharder.Encode()), nil
}
