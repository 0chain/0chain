package stakepool

import (
	cstate "0chain.net/smartcontract/common"
	"fmt"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/datastore"
)

func (sp *StakePool) UnlockPool(clientID string, providerType spenum.Provider, providerId datastore.Key,
	balances cstate.StateContextI) (string, error) {
	dp, ok := sp.Pools[clientID]
	if !ok {
		return "", fmt.Errorf("can't find pool of %v", clientID)
	}

	amount, err := sp.MintRewards(clientID, providerId, providerType, balances)
	if err != nil {
		return "", fmt.Errorf("error emptying account, %v", err)
	}

	b, err := dp.Balance.Int64()
	if err != nil {
		return "", fmt.Errorf("can't cast Balance of value (%v) to Int64", b)
	}
	i, err := amount.Int64()
	if err != nil {
		return "", fmt.Errorf("can't cast amount of value (%v) to Int64", amount)
	}
	lock := event.DelegatePoolLock{
		Client:       clientID,
		ProviderId:   providerId,
		ProviderType: providerType,
		Amount:       b,
		Reward:       amount,
		Total:        b + i,
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockStakePool, clientID, lock)
	return toJson(lock), nil
}

func (sp *StakePool) DeletePool(clientID string, providerType spenum.Provider, providerId datastore.Key,
	balances cstate.StateContextI) error {
	dp, ok := sp.Pools[clientID]
	if !ok {
		return fmt.Errorf("can't find pool of %v", clientID)
	}

	if dp.Status == spenum.Deleted {
		delete(sp.Pools, clientID)
	}

	dpUpdate := newDelegatePoolUpdate(clientID, providerId, providerType)
	dpUpdate.Updates["status"] = dp.Status
	dpUpdate.emitUpdate(balances)

	return nil
}
