package zcnsc

import (
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
)

//msgp:ignore unlockResponse stakePoolRequest

//go:generate msgp -v -io=false -tests=false -unexported

// ----------- LockingPool pool --------------------------

//type stakePool stakepool.Provider

type StakePool struct {
	stakepool.StakePool
}

func NewStakePool() *StakePool {
	pool := stakepool.NewStakePool()
	return &StakePool{
		StakePool: *pool,
	}
}

func (sp *StakePool) GetKey() datastore.Key {
	return stakepool.StakePoolKey(spenum.Authorizer, sp.Settings.DelegateWallet)
}

// save the stake pool
func (sp *StakePool) save(sscKey, providerID string, balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(stakepool.StakePoolKey(spenum.Authorizer, providerID), sp)
	return
}

// empty a delegate pool if possible, call update before the empty
func (sp *StakePool) empty(sscID, clientID string, balances cstate.StateContextI) (bool, error) {
	var dp, ok = sp.Pools[clientID]
	if !ok {
		return false, fmt.Errorf("no such delegate pool: %q", clientID)
	}

	if dp.DelegateID != clientID {
		return false, errors.New("trying to unlock not by delegate pool owner")
	}

	transfer := state.NewTransfer(sscID, clientID, dp.Balance)
	if err := balances.AddTransfer(transfer); err != nil {
		return false, err
	}

	sp.Pools[clientID].Balance = 0
	sp.Pools[clientID].Status = spenum.Deleted

	return true, nil
}

//
// smart contract methods
//

// getStakePool of given authorizer
func (zcn *ZCNSmartContract) getStakePool(authorizerID datastore.Key, balances cstate.StateContextI) (sp *StakePool, err error) {
	sp = NewStakePool()
	err = balances.GetTrieNode(stakepool.StakePoolKey(spenum.Authorizer, authorizerID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (zcn *ZCNSmartContract) getStakePoolForAdapter(_ spenum.Provider, providerID datastore.Key, balances cstate.CommonStateContextI) (sp *StakePool, err error) {
	sp = NewStakePool()
	err = balances.GetTrieNode(stakepool.StakePoolKey(spenum.Authorizer, providerID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (zcn *ZCNSmartContract) getStakePoolAdapter(providerType spenum.Provider, providerID string,
	balances cstate.StateContextI) (sp stakepool.AbstractStakePool, err error) {
	pool, err := zcn.getStakePoolForAdapter(providerType, providerID, balances)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// initial or successive method should be used by add_authorizer

// SC functions

// get existing stake pool or create new one not saving it
func (zcn *ZCNSmartContract) getOrUpdateStakePool(gn *GlobalNode,
	authorizerID datastore.Key,
	settings stakepool.Settings,
	ctx cstate.StateContextI,
) (*StakePool, error) {
	beforeFunc := func() (e error) {
		return validateStakePoolSettings(settings)
	}

	afterFunc := func() (e error) {
		gn, e = GetGlobalNode(ctx)
		if e != nil {
			return
		}

		return validateStakePoolSettings(settings, gn)
	}

	actErr := cstate.WithActivation(ctx, "apollo", beforeFunc, afterFunc)
	if actErr != nil {
		return nil, actErr
	}

	changed := false

	// the stake pool can be created by related validator
	sp, err := zcn.getStakePool(authorizerID, ctx)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		sp = NewStakePool()
		sp.Minter = cstate.MinterZcn
		sp.Settings.DelegateWallet = settings.DelegateWallet
		changed = true
	}

	if sp.Settings.ServiceChargeRatio != settings.ServiceChargeRatio {
		sp.Settings.ServiceChargeRatio = settings.ServiceChargeRatio
		changed = true
	}

	if sp.Settings.MaxNumDelegates != settings.MaxNumDelegates {
		sp.Settings.MaxNumDelegates = settings.MaxNumDelegates
		changed = true
	}

	if sp.Settings.MinStake != gn.MinStakePerDelegate {
		sp.Settings.MinStake = gn.MinStakePerDelegate
		changed = true
	}

	if changed {
		return sp, nil
	}

	return nil, fmt.Errorf("no changes have been made to stakepool for authorizerID (%s)", authorizerID)
}

func validateStakePoolSettings(poolSettings stakepool.Settings, gn ...*GlobalNode) error {
	if poolSettings.ServiceChargeRatio < 0.0 {
		return errors.New("negative service charge")
	}
	if poolSettings.MaxNumDelegates <= 0 {
		return errors.New("num_delegates <= 0")
	}

	if len(gn) > 0 && poolSettings.MaxNumDelegates > gn[0].MaxDelegates {
		return errors.New("num_delegates > max_delegates")
	}

	return nil
}

func (zcn *ZCNSmartContract) AddToDelegatePool(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (
	resp string, err error) {
	gn, err := GetGlobalNode(balances)
	if err != nil {
		return "", common.NewErrorf("add-to-delegate-pool-failed",
			"failed to get global node error: %v", err)
	}

	return stakepool.StakePoolLock(t, input, balances, stakepool.ValidationSettings{
		MinStake:        gn.MinStakeAmount,
		MaxStake:        gn.MaxStakeAmount,
		MaxNumDelegates: gn.MaxDelegates,
	}, zcn.getStakePoolAdapter)
}

func (zcn *ZCNSmartContract) DeleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte,
	balances cstate.StateContextI) (resp string, err error) {

	return stakepool.StakePoolUnlock(t, inputData, balances, zcn.getStakePoolAdapter)
}
