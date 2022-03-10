package stakepool

import (
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/state"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func (sp *StakePool) UnlockPool(
	clientID string,
	providerType spenum.Provider,
	providerId datastore.Key,
	poolId datastore.Key,
	balances cstate.StateContextI,
) (state.Balance, error) {
	var usp *UserStakePools
	usp, err := getOrCreateUserStakePool(providerType, clientID, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get user pools list: %v", err)
	}
	return sp.UnlockPoolI(
		clientID,
		providerType,
		providerId,
		poolId,
		usp,
		balances,
	)
}

func (sp *StakePool) UnlockPoolI(
	clientID string,
	providerType spenum.Provider,
	providerId datastore.Key,
	poolId datastore.Key,
	usp *UserStakePools,
	balances cstate.StateContextI,
) (state.Balance, error) {
	foundProvider := usp.Find(poolId)
	if len(foundProvider) == 0 || providerId != foundProvider {
		return 0, fmt.Errorf("user %v does not own stake pool %v", clientID, poolId)
	}

	dp, ok := sp.Pools[poolId]
	if !ok {
		return 0, fmt.Errorf("can't find pool: %v", poolId)
	}

	dp.Status = spenum.Deleting
	amount, err := sp.MintRewards(
		clientID, poolId, providerId, providerType, usp, balances,
	)
	if err != nil {
		return 0, fmt.Errorf("error emptying account, %v", err)
	}

	return amount, nil
}
