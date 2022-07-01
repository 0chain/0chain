package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (msc *MinerSmartContract) shutDownMiner(
	txn *transaction.Transaction,
	_ []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	mn, err := getMinerNode(txn.ClientID, balances)
	if err != nil {
		return "", common.NewError("shut-down-miner", err.Error())
	}
	mn.IsShutDown = true
	if err = deleteMiner(mn, gn, balances); err != nil {
		return "", common.NewError("shut-down-miner", err.Error())
	}

	return "", err
}

func (msc *MinerSmartContract) shutDownSharder(
	txn *transaction.Transaction,
	_ []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	sn, err := msc.getSharderNode(txn.ClientID, balances)
	if err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}
	sn.IsShutDown = true
	if err := deleteSharder(sn, gn, balances); err != nil {
		return "", common.NewError("kill-sharder", err.Error())
	}

	return "", err
}
