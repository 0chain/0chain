package stakepool

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/util"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

var (
	stakeHandlers = map[spenum.Provider]func(ID string, totalStake currency.Coin) (tag event.EventTag, data interface{}){
		spenum.Blobber:    event.NewUpdateBlobberTotalStakeEvent,
		spenum.Validator:  event.NewUpdateValidatorTotalStakeEvent,
		spenum.Miner:      event.NewUpdateMinerTotalStakeEvent,
		spenum.Sharder:    event.NewUpdateSharderTotalStakeEvent,
		spenum.Authorizer: event.NewUpdateAuthorizerTotalStakeEvent,
	}
)

func CheckClientBalance(
	clientId string,
	toLock currency.Coin,
	balances cstate.StateContextI,
) (err error) {

	var balance currency.Coin
	balance, err = balances.GetClientBalance(clientId)

	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	if err == util.ErrValueNotPresent {
		return errors.New("no tokens to lock")
	}

	if toLock > balance {
		return errors.New("lock amount is greater than balance")
	}

	return
}

func (sp *StakePool) LockPool(
	txn *transaction.Transaction,
	providerType spenum.Provider,
	providerId datastore.Key,
	status spenum.PoolStatus,
	balances cstate.StateContextI,
) (string, error) {
	if err := CheckClientBalance(txn.ClientID, txn.Value, balances); err != nil {
		return "", err
	}

	var newPoolId = txn.ClientID
	dp, ok := sp.Pools[newPoolId]
	if !ok {
		// new stake
		dp = &DelegatePool{
			Balance:      txn.Value,
			Reward:       0,
			Status:       status,
			DelegateID:   txn.ClientID,
			RoundCreated: balances.GetBlock().Round,
			StakedAt:     txn.CreationDate,
		}

		sp.Pools[newPoolId] = dp

	} else {
		// stake from the same clients
		if dp.DelegateID != txn.ClientID {
			return "", fmt.Errorf("could not stake for different delegate id: %s, txn client id: %s", dp.DelegateID, txn.ClientID)
		}

		//  check status, only allow staking more when current pool is active
		if dp.Status != spenum.Active && dp.Status != spenum.Pending {
			return "", fmt.Errorf("could not stake pool in %s status", dp.Status)
		}

		b, err := currency.AddCoin(dp.Balance, txn.Value)
		if err != nil {
			return "", err
		}

		dp.Balance = b
		dp.StakedAt = txn.CreationDate
	}

	i, _ := txn.Value.Int64()
	logging.Logger.Info("emmit TagLockStakePool", zap.String("client_id", txn.ClientID), zap.String("provider_id", providerId))

	lock := event.DelegatePoolLock{
		Client:       txn.ClientID,
		ProviderId:   providerId,
		ProviderType: providerType,
		Amount:       i,
	}
	balances.EmitEvent(event.TypeStats, event.TagLockStakePool, newPoolId, lock)

	if err := balances.AddTransfer(state.NewTransfer(
		txn.ClientID, txn.ToClientID, txn.Value,
	)); err != nil {
		return "", err
	}

	dp.emitNew(newPoolId, providerId, providerType, balances)

	return toJson(lock), nil
}

func (sp *StakePool) EmitStakeEvent(providerType spenum.Provider, providerID string, balances cstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}

	logging.Logger.Info("emitting stake event")

	h, ok := stakeHandlers[providerType]
	if !ok {
		return errors.New("unsupported providerType in stakepool StakeEvent")
	}

	tag, data := h(providerID, staked)
	balances.EmitEvent(event.TypeStats, tag, providerID, data)
	return nil
}
