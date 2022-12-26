package stakepool

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

var (
	unstakeHandlers = map[spenum.Provider]func(ID string, totalStake currency.Coin) (tag event.EventTag, data interface{}){
		spenum.Blobber:    event.NewUpdateBlobberTotalUnStakeEvent,
		spenum.Validator:  event.NewUpdateValidatorTotalUnStakeEvent,
		spenum.Miner:      event.NewUpdateMinerTotalUnStakeEvent,
		spenum.Sharder:    event.NewUpdateSharderTotalUnStakeEvent,
		spenum.Authorizer: event.NewUpdateAuthorizerTotalUnStakeEvent,
	}
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

	i, _ := amount.Int64()
	lock := event.DelegatePoolLock{
		Client:       clientID,
		ProviderId:   providerId,
		ProviderType: providerType,
		Amount:       i,
	}

	if dp.Status == spenum.Deleted {
		delete(sp.Pools, clientID)
	}

	dpUpdate := newDelegatePoolUpdate(clientID, providerId, providerType)
	dpUpdate.Updates["status"] = dp.Status
	dpUpdate.emitUpdate(balances)

	err = sp.EmitUnStakeEvent(providerType, providerId, amount, balances)
	if err != nil {
		return "", fmt.Errorf(
			"stake pool staking error: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockStakePool, clientID, lock)
	return toJson(lock), nil
}

func (sp *StakePool) EmitUnStakeEvent(providerType spenum.Provider, providerID string, amount currency.Coin, balances cstate.StateContextI) error {
	logging.Logger.Info("emitting stake event")

	h, ok := unstakeHandlers[providerType]
	if !ok {
		return errors.New("unsupported providerType in stakepool StakeEvent")
	}

	tag, data := h(providerID, amount)
	balances.EmitEvent(event.TypeStats, tag, providerID, data)
	return nil
}
