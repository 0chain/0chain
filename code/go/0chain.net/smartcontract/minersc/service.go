package minersc

import (
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/config"
)

// killMiner
// killing is permanent and a killed miner cannot receive any rewards
func (_ *MinerSmartContract) addHardFork(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {

	if txn.ClientID != gn.OwnerId {
		return "", common.NewError("add_hardfork", "only sc owner can add hardfork")
	}

	var changes config.StringMap
	if err = changes.Decode(input); err != nil {
		return "", common.NewError("add_hardfork", err.Error())
	}

	for key, val := range changes.Fields {
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return "", common.NewError("add_hardfork", err.Error())
		}
		h := cstate.NewHardFork(key, i)
		if _, err := balances.InsertTrieNode(h.GetKey(), h); err != nil {
			return "", common.NewError("add_hardfork", err.Error())
		}

	}
	return "", nil
}
