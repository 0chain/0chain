package stakepool

import (
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
	dpUpdate := newDelegatePoolUpdate(clientID, providerId, providerType)
	dpUpdate.Updates["status"] = spenum.Deleting
	dpUpdate.emitUpdate(balances)

	err = sp.EmitUnStakeEvent(providerType, providerId, amount, balances)
	if err != nil {
		return "", fmt.Errorf(
			"stake pool staking error: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockStakePool, clientID, lock)
	if err != nil {
		return "", fmt.Errorf("error emptying account, %v", err)
	}

	return toJson(lock), nil
}

func (sp *StakePool) EmitUnStakeEvent(providerType spenum.Provider, providerID string, amount currency.Coin, balances cstate.StateContextI) error {
	logging.Logger.Info("emitting stake event")

	h, ok := unstakeHandlers[providerType]
	if !ok {
		logging.Logger.Error("unsupported providerType in stakepool StakeEvent")
		return nil
	}

	tag, data := h(providerID, amount)
	balances.EmitEvent(event.TypeStats, tag, providerID, data)
	return nil
}
