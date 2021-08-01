package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	zchainErrors "github.com/0chain/gosdk/errors"
	"github.com/pkg/errors"
)

func (msc *MinerSmartContract) minerHealthCheck(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	all, err := getMinersList(balances)
	if err != nil {
		return "", errors.Wrap(err, zchainErrors.New("miner_health_check_failed", "Failed to get miner list").Error())

	}

	var existingMiner *MinerNode
	if existingMiner, err = getMinerNode(t.ClientID, balances); err != nil {
		return "", errors.Wrap(err, zchainErrors.New("miner_health_check_failed", "can't get the miner "+t.ClientID).Error())

	}

	existingMiner.LastHealthCheck = t.CreationDate

	for _, nodes := range all.Nodes {
		if nodes.ID == t.ClientID {
			nodes.LastHealthCheck = t.CreationDate
			break
		}
	}

	if err = updateMinersList(balances, all); err != nil {
		return "", errors.Wrap(err, zchainErrors.New("miner_health_check_failed",
			"can't save all miners list").Error())
	}

	err = existingMiner.save(balances)
	if err != nil {
		return "", errors.Wrap(err, zchainErrors.New("miner_health_check_failed",
			"can't save miner").Error())
	}

	return string(existingMiner.Encode()), nil
}

func (msc *MinerSmartContract) sharderHealthCheck(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	all, err := getAllShardersList(balances)
	if err != nil {
		return "", errors.Wrap(err, zchainErrors.New("sharder_health_check_failed",
			"Failed to get sharder list").Error())

	}

	var existingSharder *MinerNode
	if existingSharder, err = msc.getSharderNode(t.ClientID, balances); err != nil {
		return "", errors.Wrap(err, zchainErrors.New("sharder_health_check_failed",
			"can't get the sharder "+t.ClientID).Error())
	}

	existingSharder.LastHealthCheck = t.CreationDate

	for _, nodes := range all.Nodes {
		if nodes.ID == t.ClientID {
			nodes.LastHealthCheck = t.CreationDate
			break
		}
	}

	if err = updateAllShardersList(balances, all); err != nil {
		return "", errors.Wrap(err, zchainErrors.New("sharder_health_check_failed",
			"can't save all sharders list").Error())
	}

	err = existingSharder.save(balances)
	if err != nil {
		return "", errors.Wrap(err, zchainErrors.New("sharder_health_check_failed",
			"can't save sharder").Error())

	}

	return string(existingSharder.Encode()), nil
}
