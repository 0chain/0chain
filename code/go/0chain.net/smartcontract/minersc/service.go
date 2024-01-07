package minersc

import (
	"sort"
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/config"
	common2 "0chain.net/smartcontract/common"
)

// addHardFork
// addHardFork creates a hardfork in db
func (_ *MinerSmartContract) addHardFork(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	var changes config.StringMap
	if err = changes.Decode(input); err != nil {
		return "", common.NewError("add_hardfork", err.Error())
	}

	sortedKeys := make([]string, len(changes.Fields))
	for k := range changes.Fields {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		val := changes.Fields[key]
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
