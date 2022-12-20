package stakepool

import (
	"fmt"

	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/provider/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func (sp *StakePool) UnlockPool(
	clientID string,
	providerType spenum.Provider,
	providerId datastore.Key,
	balances cstate.StateContextI,
) (string, error) {
	dp, ok := sp.Pools[clientID]
	if !ok {
		return "", fmt.Errorf("can't find pool of %v", clientID)
	}

	dp.Status = spenum.Deleting
	amount, err := sp.MintRewards(
		clientID, providerId, providerType, balances,
	)

	i, _ := amount.Int64()
	lock := event.DelegatePoolLock{
		Client:       clientID,
		ProviderId:   providerId,
		ProviderType: providerType,
		Amount:       i,
	}
	dpUpdate := provider.NewDelegatePoolUpdate(clientID, providerId, providerType)
	dpUpdate.Updates["status"] = spenum.Deleting
	dpUpdate.EmitUpdate(balances)

	balances.EmitEvent(event.TypeStats, event.TagUnlockStakePool, clientID, lock)
	if err != nil {
		return "", fmt.Errorf("error emptying account, %v", err)
	}

	return toJson(lock), nil
}
