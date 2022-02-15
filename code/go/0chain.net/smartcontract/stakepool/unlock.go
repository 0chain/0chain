package stakepool

import (
	"fmt"

	"0chain.net/chaincore/state"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func (sp *StakePool) UnlockPool(
	clientId string,
	providerType Provider,
	providerId datastore.Key,
	poolId datastore.Key,
	balances cstate.StateContextI,
) (state.Balance, error) {
	var usp *UserStakePools
	usp, err := getOrCreateUserStakePool(providerType, txn.ClientID, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get user pools list: %v", err)
	}
	foundProvider := usp.Find(poolId)
	if len(foundProvider) == 0 || providerId != foundProvider {
		return 0, fmt.Errorf("user %v does not own stake pool %v", txn.ClientID, poolId)
	}

	dp, ok := sp.Pools[poolId]
	if !ok {
		return 0, fmt.Errorf("can't find pool: %v", poolId)
	}
	minter, err := cstate.GetMinter(sp.Minter)
	if err != nil {
		return 0, fmt.Errorf("can't find minter: %v", err)
	}
	transfer := state.NewTransfer(minter, txn.ClientID, dp.Balance)
	if err := balances.AddTransfer(transfer); err != nil {
		return 0, err
	}

	dp.Balance = 0
	dp.Status = Deleted
	amount, err := sp.MintRewards(
		txn.ClientID, poolId, providerId, providerType, usp, balances,
	)
	if err != nil {
		return 0, fmt.Errorf("error emptying account, %v", err)
	}

	return amount, nil
}
