package zcnsc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
)

//msgp:ignore unlockResponse stakePoolRequest

//go:generate msgp -v -io=false -tests=false -unexported

// unlock response
type unlockResponse struct {
	// one of the fields is set in a response, the Unstake if can't unstake
	// for now and the TokenPoolTransferResponse if it has a pool had unlocked
	Unstake bool          `json:"unstake"` // max time to wait to unstake
	Balance currency.Coin `json:"balance"`
}

type stakePoolRequest struct {
	PoolID       string `json:"pool_id,omitempty"`
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

//type stakePool stakepool.StakePool

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

func (sp *StakePool) GetKey() datastore.Key {
	return StakePoolKey(ADDRESS, sp.Settings.DelegateWallet)
}

// save the stake pool
func (sp *StakePool) save(sscKey, providerID string, balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(StakePoolKey(sscKey, providerID), sp)
	return
}

// empty a delegate pool if possible, call update before the empty
func (sp *StakePool) empty(sscID, poolID, clientID string, balances cstate.StateContextI) (bool, error) {
	var dp, ok = sp.Pools[poolID]
	if !ok {
		return false, fmt.Errorf("no such delegate pool: %q", poolID)
	}

	if dp.DelegateID != clientID {
		return false, errors.New("trying to unlock not by delegate pool owner")
	}

	transfer := state.NewTransfer(sscID, clientID, dp.Balance)
	if err := balances.AddTransfer(transfer); err != nil {
		return false, err
	}

	sp.Pools[poolID].Balance = 0
	sp.Pools[poolID].Status = spenum.Deleting

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

func (zcn *ZCNSmartContract) AddToDelegatePool(
	t *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (resp string, err error) {
	code := "stake_pool_lock_failed"

	gn, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node, err: %v", err)
		return "", common.NewError(code, msg)
	}

	if t.Value < gn.MinLockAmount {
		return "", common.NewError(code, "too small stake to lock")
	}

	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf(code, "invalid request: %v", err)
	}

	var sp *StakePool
	if sp, err = zcn.getStakePool(spr.AuthorizerID, ctx); err != nil {
		return "", common.NewErrorf(code, "can't get stake pool: %v", err)
	}

	if len(sp.Pools) >= gn.MaxDelegates {
		return "", common.NewErrorf(code, "max_delegates reached: %v, no more stake pools allowed", gn.MaxDelegates)
	}

	err = sp.LockPool(t, spenum.Authorizer, spr.AuthorizerID, spenum.Active, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "stake pool digging error: %v", err)
	}

	if err = sp.save(zcn.ID, spr.AuthorizerID, ctx); err != nil {
		return "", common.NewErrorf(code, "saving stake pool: %v", err)
	}

	// TO-DO: Update stake in eventDB

	return
}

func (zcn *ZCNSmartContract) DeleteFromDelegatePool(
	t *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (resp string, err error) {
	const code = "stake_pool_unlock_failed"
	var spr stakePoolRequest

	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf(code, "can't decode request: %v", err)
	}
	var sp *StakePool
	if sp, err = zcn.getStakePool(spr.AuthorizerID, ctx); err != nil {
		return "", common.NewErrorf(code, "can't get related stake pool: %v", err)
	}

	_, err = sp.empty(zcn.ID, spr.PoolID, t.ClientID, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "unlocking tokens: %v", err)
	}

	amount, err := sp.UnlockClientStakePool(t.ClientID, spenum.Blobber, spr.AuthorizerID, spr.PoolID, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "%v", err)
	}

	// save the pool
	if err = sp.save(zcn.ID, spr.AuthorizerID, ctx); err != nil {
		return "", common.NewErrorf(code, "saving stake pool: %v", err)
	}

	return toJson(&unlockResponse{Unstake: true, Balance: amount}), nil
}

func toJson(val interface{}) string {
	var b, err = json.Marshal(val)
	if err != nil {
		panic(err)
	}
	return string(b)
}
