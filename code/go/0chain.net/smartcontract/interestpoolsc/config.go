package interestpoolsc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract"
)

type Setting int

const (
	MinLock Setting = iota
	Apr
	MinLockPeriod
	MaxMint
	OwnerId
)

var (
	Settings = []string{
		"min_lock",
		"apr",
		"min_lock_period",
		"max_mint",
		"owner_id",
	}
)

func (ip *InterestPoolSmartContract) updateVariables(
	t *transaction.Transaction,
	gn *GlobalNode,
	inputData []byte,
	balances c_state.StateContextI,
) (string, error) {
	if err := smartcontractinterface.AuthorizeWithOwner("update_variables", func() bool {
		return gn.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	changes := &smartcontract.StringMap{}
	if err := changes.Decode(inputData); err != nil {
		return "", common.NewError("failed to update variables",
			"request not formatted correctly")
	}

	for key, value := range changes.Fields {
		gn.set(key, value)
	}

	balances.InsertTrieNode(gn.getKey(), gn)
	return string(gn.Encode()), nil
}
