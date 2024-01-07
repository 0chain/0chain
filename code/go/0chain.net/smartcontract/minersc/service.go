package minersc

import (
	"strconv"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/config"
	common2 "0chain.net/smartcontract/common"
)

// killMiner
// killing is permanent and a killed miner cannot receive any rewards
func (_ *MinerSmartContract) addHardFork(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances common2.StateContextI,
) (resp string, err error) {
	var changes config.StringMap
	if err = changes.Decode(input); err != nil {
		return "", common.NewError("add_hardfork", err.Error())
	}

	for key, val := range changes.Fields {
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return "", common.NewError("add_hardfork", err.Error())
		}
		h := common2.NewHardFork(key, i)
		if _, err := balances.InsertTrieNode(h.GetKey(), h); err != nil {
			return "", common.NewError("add_hardfork", err.Error())
		}

	}
	return "", nil
}
