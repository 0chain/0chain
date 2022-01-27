package stakepool

import (
	"fmt"

	"0chain.net/chaincore/state"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

func UnlockPool(
	txn *transaction.Transaction,
	providerType Provider,
	providerId datastore.Key,
	poolId datastore.Key,
	balances cstate.StateContextI,
) (state.Balance, error) {
	sp, err := GetStakePool(providerType, providerId, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get stake pool: %v", err)
	}

	var usp *userStakePools
	usp, err = getOrCreateUserStakePool(providerType, txn.ClientID, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get user pools list: %v", err)
	}
	foundProvider := usp.find(poolId)
	if len(foundProvider) == 0 || providerId != foundProvider {
		return 0, fmt.Errorf("user %v does not own stake pool %v", txn.ClientID, poolId)
	}

	dp, ok := sp.Pools[poolId]
	if !ok {
		return 0, fmt.Errorf("can't get find pools: %v", poolId)
	}

	dp.Status = Deleting
	amount, removed, err := sp.EmptyAccount(txn.ClientID, poolId, balances)
	if err != nil {
		return 0, fmt.Errorf("error emptying account, %v", err)
	}
	if !removed {
		return 0, fmt.Errorf("can't delete pool: %v", poolId)
	}

	usp.del(providerId, poolId)

	return amount, nil
}
