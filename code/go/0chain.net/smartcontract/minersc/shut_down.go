package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
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
	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, mn.ID, dbs.DbUpdates{
		Id: mn.ID,
		Updates: map[string]interface{}{
			"is_shut_down": mn.IsShutDown,
		},
	})
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
		return "", common.NewError("shut-down-sharder", err.Error())
	}
	sn.IsShutDown = true
	if err := deleteSharder(sn, gn, balances); err != nil {
		return "", common.NewError("shut-down-sharder", err.Error())
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, sn.ID, dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"is_shut_down": sn.IsShutDown,
		},
	})
	return "", err
}
