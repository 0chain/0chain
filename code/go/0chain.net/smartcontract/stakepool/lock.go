package stakepool

import (
	"fmt"

	"0chain.net/chaincore/state"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

func (sp *StakePool) LockPool(
	txn *transaction.Transaction,
	providerType Provider,
	providerId datastore.Key,
	status PoolStatus,
	balances cstate.StateContextI,
) error {
	const MaxDelegates = 100

	if len(sp.Pools) >= MaxDelegates {
		return fmt.Errorf("max_delegates reached: %v, no more stake pools allowed",
			MaxDelegates)
	}

	dp := DelegatePool{
		Balance:      state.Balance(txn.Value),
		Reward:       0,
		Status:       status,
		RoundCreated: balances.GetBlock().Round,
	}

	if err := balances.AddTransfer(state.NewTransfer(
		txn.ClientID, txn.ToClientID, state.Balance(txn.Value),
	)); err != nil {
		return err
	}

	var newPoolId = txn.Hash
	sp.Pools[newPoolId] = &dp

	var usp *UserStakePools
	usp, err := getOrCreateUserStakePool(providerType, txn.ClientID, balances)
	if err != nil {
		return fmt.Errorf("can't get user pools list: %v", err)
	}
	usp.add(providerId, newPoolId)
	if err = usp.Save(providerType, txn.ClientID, balances); err != nil {
		return fmt.Errorf("saving user pools: %v", err)
	}

	if err := dp.emitNew(
		txn.ClientID,
		newPoolId,
		providerId,
		providerType,
		status,
		balances,
	); err != nil {
		return err
	}

	return nil
}
