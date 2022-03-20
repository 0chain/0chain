package zcnsc

import (
	"encoding/json"
	"errors"
	"fmt"

	chainstate "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
)

// unlock response
type unlockResponse struct {
	// one of the fields is set in a response, the Unstake if can't unstake
	// for now and the TokenPoolTransferResponse if it has a pool had unlocked
	Unstake bool          `json:"unstake"` // max time to wait to unstake
	Balance state.Balance `json:"balance"`
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

// ----------- LockingPool pool --------------------------

type stakePool struct {
	stakepool.StakePool
}

func newStakePool() *stakePool {
	bsp := stakepool.NewStakePool()
	return &stakePool{
		StakePool: *bsp,
	}
}

// stake pool key for the storage SC and service provider ID
func stakePoolKey(scKey, providerID string) datastore.Key {
	return scKey + ":stakepool:" + providerID
}

func stakePoolID(scKey, providerID string) datastore.Key {
	return encryption.Hash(stakePoolKey(scKey, providerID))
}

// Encode to []byte
func (sp *stakePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(sp); err != nil {
		panic(err) // must never happen
	}
	return
}

// Decode from []byte
func (sp *stakePool) Decode(input []byte) error {
	return json.Unmarshal(input, sp)
}

// save the stake pool
func (sp *stakePool) save(sscKey, providerID string, balances chainstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(stakePoolKey(sscKey, providerID), sp)
	return
}

// The stake() returns total stake size including delegate pools want to unstake.
func (sp *stakePool) stake() (stake state.Balance) {
	for _, dp := range sp.Pools {
		stake += dp.Balance
	}
	return
}

// empty a delegate pool if possible, call update before the empty
func (sp *stakePool) empty(sscID, poolID, clientID string, balances chainstate.StateContextI) (bool, error) {
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
func (zcn *ZCNSmartContract) getStakePool(providerID datastore.Key, balances chainstate.StateContextI) (sp *stakePool, err error) {
	sp = newStakePool()
	err = balances.GetTrieNode(stakePoolKey(zcn.ID, providerID), sp)
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
	providerId datastore.Key,
	providerType spenum.Provider,
	settings stakepool.StakePoolSettings,
	balances chainstate.StateContextI,
) (*stakePool, error) {
	if err := validateStakePoolSettings(settings, gn); err != nil {
		return nil, fmt.Errorf("invalid stake_pool settings: %v", err)
	}

	// the stake pool can be created by related validator
	sp, err := zcn.getStakePool(providerId, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		sp = newStakePool()
		sp.Settings.DelegateWallet = settings.DelegateWallet
		sp.Minter = chainstate.MinterStorage
	}

	sp.Settings.MinStake = settings.MinStake
	sp.Settings.MaxStake = settings.MaxStake
	sp.Settings.ServiceCharge = settings.ServiceCharge
	sp.Settings.MaxNumDelegates = settings.MaxNumDelegates

	return sp, nil
}

func validateStakePoolSettings(sps stakepool.StakePoolSettings, conf *GlobalNode) error {
	err := conf.validateStakeRange(sps.MinStake, sps.MaxStake)
	if err != nil {
		return err
	}
	if sps.ServiceCharge < 0.0 {
		return errors.New("negative service charge")
	}
	if sps.MaxNumDelegates <= 0 {
		return errors.New("num_delegates <= 0")
	}

	return nil
}

func (gn *GlobalNode) validateStakeRange(min, max state.Balance) (err error) {
	if min < gn.MinStakeAmount {
		return fmt.Errorf("min_stake is less than allowed by SC: %v < %v", min, gn.MinStakeAmount)
	}
	if max < min {
		return fmt.Errorf("max_stake less than min_stake: %v < %v", min, max)
	}

	return
}

func (zcn *ZCNSmartContract) DistributeRewards(
	t *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (resp string, err error) {
	return "", nil
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

	var sp *stakePool
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
	var sp *stakePool
	if sp, err = zcn.getStakePool(spr.AuthorizerID, ctx); err != nil {
		return "", common.NewErrorf(code, "can't get related stake pool: %v", err)
	}

	unstake, err := sp.empty(zcn.ID, spr.PoolID, t.ClientID, ctx)
	if err != nil {
		return "", common.NewErrorf(code,
			"unlocking tokens: %v", err)
	}

	// the tokens can't be unlocked due to opened offers, but we mark it
	// as 'unstake' and returns maximal time to wait to unlock the pool
	if !unstake {
		// save the pool and return special result
		if err = sp.save(zcn.ID, spr.AuthorizerID, ctx); err != nil {
			return "", common.NewErrorf(code, "saving stake pool: %v", err)
		}
		return toJson(&unlockResponse{Unstake: false}), nil
	}

	amount, err := sp.UnlockPool(t.ClientID, spenum.Blobber, spr.AuthorizerID, spr.PoolID, ctx)
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
