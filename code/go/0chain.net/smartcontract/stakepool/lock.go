package stakepool

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/state"

	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/util"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
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
) error {
	if err := CheckClientBalance(txn.ClientID, txn.Value, balances); err != nil {
		return err
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
		dp.emitNew(newPoolId, providerId, providerType, balances)
	} else {
		// stake from the same clients
		if dp.DelegateID != txn.ClientID {
			return fmt.Errorf("could not stake for different delegate id: %s, txn client id: %s", dp.DelegateID, txn.ClientID)
		}

		//  check status, only allow staking more when current pool is active
		if dp.Status != spenum.Active && dp.Status != spenum.Pending {
			return fmt.Errorf("could not stake pool in %s status", dp.Status)
		}

		var dpUpdate = newDelegatePoolUpdate(txn.ClientID, providerId, providerType)
		var err error
		dpUpdate.Updates["balance"], err = currency.AddCoin(dp.Balance, txn.Value)
		if err != nil {
			return err
		}
		dpUpdate.emitUpdate(balances)
	}

	if err := balances.AddTransfer(state.NewTransfer(
		txn.ClientID, txn.ToClientID, txn.Value,
	)); err != nil {
		return err
	}

	return nil
}
