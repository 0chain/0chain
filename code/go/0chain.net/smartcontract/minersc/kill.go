package minersc

import (
	"encoding/json"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/chaincore/smartcontractinterface"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type idInput struct {
	ID string `json:"id"`
}

func (ki *idInput) decode(p []byte) error {
	return json.Unmarshal(p, ki)
}

func (ki *idInput) Encode() []byte {
	buff, _ := json.Marshal(ki)
	return buff
}

func (msc *MinerSmartContract) killMiner(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	if err := smartcontractinterface.AuthorizeWithOwner("kill-miner", func() bool {
		return gn.OwnerId == txn.ClientID
	}); err != nil {
		return "", err
	}
	var id idInput
	if err := id.decode(input); err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}

	mn, err := getMinerNode(id.ID, balances)
	if err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}
	if mn.IsKilled {
		return "", common.NewError("kill-miner", "miner already dead")
	}

	mn.IsKilled = true
	if err = deleteMiner(mn, gn, balances); err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}

	mn.IsDead = true
	if err := mn.SlashFraction(gn.StakeKillSlash, id.ID, spenum.Miner, balances); err != nil {
		return "", common.NewError("kill-miner", "slashing stake pools: "+err.Error())
	}
	if err := mn.save(balances); err != nil {
		return "", common.NewError("kill-miner", "saving miner: "+err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, mn.ID, dbs.DbUpdates{
		Id: mn.ID,
		Updates: map[string]interface{}{
			"is_killed": mn.IsKilled,
		},
	})

	return "", err
}

func (msc *MinerSmartContract) killSharder(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	if err := smartcontractinterface.AuthorizeWithOwner("kill_sharder", func() bool {
		return gn.OwnerId == txn.ClientID
	}); err != nil {
		return "", err
	}
	var id idInput
	if err := id.decode(input); err != nil {
		return "", common.NewError("kill-sharder", err.Error())
	}

	sn, err := msc.getSharderNode(id.ID, balances)
	if err != nil {
		return "", common.NewError("kill-sharder", err.Error())
	}
	sn.IsKilled = true
	if err := deleteSharder(sn, gn, balances); err != nil {
		return "", common.NewError("kill-sharder", err.Error())
	}

	sn.IsDead = true
	if err := sn.SlashFraction(gn.StakeKillSlash, id.ID, spenum.Sharder, balances); err != nil {
		return "", common.NewError("kill-miner", "slashing stake pools: "+err.Error())
	}
	if err := sn.save(balances); err != nil {
		return "", common.NewError("kill-sharder", "saving sharder: "+err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, sn.ID, dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"is_killed": sn.IsKilled,
		},
	})

	return "", err
}
