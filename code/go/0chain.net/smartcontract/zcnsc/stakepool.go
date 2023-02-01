package zcnsc

import (
	"encoding/json"
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"
)

//msgp:ignore unlockResponse stakePoolRequest

//go:generate msgp -v -io=false -tests=false -unexported

type stakePoolRequest struct {
	AuthorizerID string `json:"authorizer_id,omitempty"`
}

func (spr *stakePoolRequest) decode(p []byte) (err error) {
	if err = json.Unmarshal(p, spr); err != nil {
		return
	}
	return // ok
}

func (spr *stakePoolRequest) encode() []byte {
	bytes, _ := json.Marshal(spr)
	return bytes
}

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

// StakePoolKey stake pool key for the storage SC and service provider ID
func StakePoolKey(scKey, providerID string) datastore.Key {
	return scKey + ":stakepool:" + providerID
}

func stakePoolKey(providerType spenum.Provider, providerID string) datastore.Key {
	return providerType.String() + ":stakepool:" + providerID
}

func (sp *StakePool) GetKey() datastore.Key {
	return StakePoolKey(ADDRESS, sp.Settings.DelegateWallet)
}

// save the stake pool
func (sp *StakePool) save(sscKey, providerID string, balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(StakePoolKey(sscKey, providerID), sp)
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
	err = balances.GetTrieNode(StakePoolKey(zcn.ID, authorizerID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (zcn *ZCNSmartContract) getStakePoolForAdapter(providerType spenum.Provider, providerID datastore.Key, balances cstate.CommonStateContextI) (sp *StakePool, err error) {
	sp = NewStakePool()
	err = balances.GetTrieNode(stakePoolKey(providerType, providerID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (zcn *ZCNSmartContract) getStakePoolAdapter(providerType spenum.Provider, providerID string,
	balances cstate.CommonStateContextI) (sp stakepool.AbstractStakePool, err error) {
	pool, err := zcn.getStakePoolForAdapter(providerType, providerID, balances)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// add delegated stake pool
func (zcn *ZCNSmartContract) stakePoolLock(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (res string, err error) {
	return stakepool.StakePoolLock(t, input, balances, zcn.getStakePoolAdapter)
}

// stake pool can return excess tokens from stake pool
func (zcn *ZCNSmartContract) stakePoolUnlock(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (resp string, err error) {
	return stakepool.StakePoolUnlock(t, input, balances, zcn.getStakePoolAdapter)
}

// initial or successive method should be used by add_authorizer

// SC functions

// get existing stake pool or create new one not saving it
func (zcn *ZCNSmartContract) getOrUpdateStakePool(
	gn *GlobalNode,
	authorizerID datastore.Key,
	settings stakepool.Settings,
	ctx cstate.StateContextI,
) (*StakePool, error) {
	if err := validateStakePoolSettings(settings, gn); err != nil {
		return nil, fmt.Errorf("invalid stake_pool settings: %v", err)
	}

	changed := false

	// the stake pool can be created by related validator
	sp, err := zcn.getStakePool(authorizerID, ctx)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		sp = NewStakePool()
		sp.Minter = cstate.MinterStorage
		sp.Settings.DelegateWallet = settings.DelegateWallet
		changed = true
	}

	if sp.Settings.MinStake != settings.MinStake {
		sp.Settings.MinStake = settings.MinStake
		changed = true
	}

	if sp.Settings.MaxStake != settings.MaxStake {
		sp.Settings.MaxStake = settings.MaxStake
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

	if changed {
		return sp, nil
	}

	return nil, fmt.Errorf("no changes have been made to stakepool for authorizerID (%s)", authorizerID)
}

func validateStakePoolSettings(poolSettings stakepool.Settings, conf *GlobalNode) error {
	err := conf.validateStakeRange(poolSettings.MinStake, poolSettings.MaxStake)
	if err != nil {
		return err
	}
	if poolSettings.ServiceChargeRatio < 0.0 {
		return errors.New("negative service charge")
	}
	if poolSettings.MaxNumDelegates <= 0 {
		return errors.New("num_delegates <= 0")
	}

	return nil
}

func (gn *GlobalNode) validateStakeRange(min, max currency.Coin) (err error) {
	if min < gn.MinStakeAmount {
		return fmt.Errorf("min_stake is less than allowed by SC: %v < %v", min, gn.MinStakeAmount)
	}
	if max < min {
		return fmt.Errorf("max_stake less than min_stake: %v < %v", min, max)
	}

	return
}

func (zcn *ZCNSmartContract) AddToDelegatePool(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (
	resp string, err error) {

	return stakepool.StakePoolLock(t, input, balances, zcn.getStakePoolAdapter)
}

func (zcn *ZCNSmartContract) DeleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte,
	balances cstate.StateContextI) (resp string, err error) {

	return stakepool.StakePoolUnlock(t, inputData, balances, zcn.getStakePoolAdapter)
}
