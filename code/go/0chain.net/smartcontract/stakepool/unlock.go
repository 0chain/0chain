package stakepool

import (
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func (sp *StakePool) UnlockClientStakePool(
	clientID string,
	providerType spenum.Provider,
	providerId datastore.Key,
	balances cstate.StateContextI,
) (currency.Coin, error) {
	var usp *UserStakePools
	usp, err := getOrCreateUserStakePool(providerType, clientID, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get user pools list: %v", err)
	}

	return sp.UnlockPool(
		clientID,
		providerType,
		providerId,
		usp,
		balances,
	)
}

func (sp *StakePool) UnlockPool(
	clientID string,
	providerType spenum.Provider,
	providerId datastore.Key,
	usp *UserStakePools,
	balances cstate.StateContextI,
) (currency.Coin, error) {
	if _, ok := usp.Find(providerId); !ok {
		return 0, fmt.Errorf("user %v does not own stake pool for %v", clientID, providerId)
	}

	dp, ok := sp.Pools[clientID]
	if !ok {
		return 0, fmt.Errorf("can't find pool of %v", clientID)
	}

	dp.Status = spenum.Deleting
	amount, err := sp.MintRewards(
		clientID, providerId, providerType, usp, balances,
	)
	if err != nil {
		return 0, fmt.Errorf("error emptying account, %v", err)
	}

	return amount, nil
}
