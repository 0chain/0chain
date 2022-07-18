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
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var id idInput
	if err := id.decode(input); err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}

	mn, err := getMinerNode(id.ID, balances)
	if err != nil {
		return "", common.NewError("shut-down-miner", err.Error())
	}

	if txn.ClientID != mn.StakePool.Settings.DelegateWallet {
		return "", common.NewError("shut-down-miner",
			"access denied, allowed for delegate_wallet owner only")
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
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var id idInput
	if err := id.decode(input); err != nil {
		return "", common.NewError("kill-sharder", err.Error())
	}

	sn, err := msc.getSharderNode(id.ID, balances)
	if err != nil {
		return "", common.NewError("shut-down-sharder", err.Error())
	}

	if txn.ClientID != sn.StakePool.Settings.DelegateWallet {
		return "", common.NewError("shut-down-sharder",
			"access denied, allowed for delegate_wallet owner only")
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
