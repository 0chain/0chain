package minersc

import (
	"encoding/json"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type killInput struct {
	ID string `json:"id"`
}

func (ki *killInput) decode(p []byte) error {
	return json.Unmarshal(p, ki)
}

func (ki *killInput) Encode() []byte {
	buff, _ := json.Marshal(ki)
	return buff
}

func (msc *MinerSmartContract) killMiner(
	_ *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var id killInput
	if err := id.decode(input); err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}

	mn, err := getMinerNode(id.ID, balances)
	if err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}
	mn.IsKilled = true
	if err = deleteMiner(mn, gn, balances); err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}

	return "", err
}

func (msc *MinerSmartContract) killSharder(
	_ *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var id killInput
	if err := id.decode(input); err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}

	sn, err := msc.getSharderNode(id.ID, balances)
	if err != nil {
		return "", common.NewError("kill-miner", err.Error())
	}
	sn.IsKilled = true
	if err := deleteSharder(sn, gn, balances); err != nil {
		return "", common.NewError("kill-sharder", err.Error())
	}

	return "", err
}
