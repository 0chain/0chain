package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type stakePoolSettings struct {
	// DelegateWallet for pool owner.
	DelegateWallet string `json:"delegate_wallet"`
	// MinStake allowed.
	MinStake state.Balance `json:"min_stake"`
	// MaxStake allowed.
	MaxStake state.Balance `json:"max_stake"`
	// NumDelegates maximum allowed.
	MaxNumDelegates int `json:"num_delegates"`
	// ServiceCharge of the blobber. The blobber gets this % (actually, value in
	// [0; 1) range). If the ServiceCharge greater than max_charge of the SC
	// then the blobber can't be registered / updated.
	ServiceCharge float64 `json:"service_charge"`
}

func validateStakePoolSettings(
	sps stakepool.StakePoolSettings,
	conf *Config,
) error {
	err := conf.validateStakeRange(sps.MinStake, sps.MaxStake)
	if err != nil {
		return err
	}
	if sps.ServiceCharge < 0.0 {
		return errors.New("negative service charge")
	}
	if sps.ServiceCharge > conf.MaxCharge {
		return fmt.Errorf("service_charge (%f) is greater than"+
			" max allowed by SC (%f)", sps.ServiceCharge, conf.MaxCharge)
	}
	if sps.MaxNumDelegates <= 0 {
		return errors.New("num_delegates <= 0")
	}
	return nil
}

// stake pool of a blobber

type stakePool struct {
	stakepool.StakePool
	// TotalOffers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	TotalOffers state.Balance `json:"total_offers"`
	// Total amount to be un staked
	TotalUnStake state.Balance `json:"total_un_stake"`
}

func newStakePool() *stakePool {
	bsp := stakepool.NewStakePool()
	return &stakePool{
		StakePool: *bsp,
	}
}

// stake pool key for the storage SC and  blobber
func stakePoolKey(scKey, blobberID string) datastore.Key {
	return datastore.Key(scKey + ":stakepool:" + blobberID)
}

func stakePoolID(scKey, blobberID string) datastore.Key {
	return encryption.Hash(stakePoolKey(scKey, blobberID))
}

// Encode to []byte
func (sp *stakePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(sp); err != nil {
		panic(err) // must never happens
	}
	return
}

// Decode from []byte
func (sp *stakePool) Decode(input []byte) error {
	return json.Unmarshal(input, sp)
}

// save the stake pool
func (sp *stakePool) save(sscKey, blobberID string,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(stakePoolKey(sscKey, blobberID), sp)
	return
}

// The cleanStake() is stake amount without delegate pools want to unstake.
func (sp *stakePool) cleanStake() (stake state.Balance) {
	return sp.stake() - sp.TotalUnStake
}

// The stake() returns total stake size including delegate pools want to unstake.
func (sp *stakePool) stake() (stake state.Balance) {
	for _, dp := range sp.Pools {
		stake += dp.Balance
	}
	return
}

// empty a delegate pool if possible, call update before the empty
func (sp *stakePool) empty(
	sscID,
	poolID,
	clientID string,
	balances chainstate.StateContextI,
) (bool, error) {
	var dp, ok = sp.Pools[poolID]
	if !ok {
		return false, fmt.Errorf("no such delegate pool: %q", poolID)
	}

	if dp.DelegateID != clientID {
		return false, errors.New("trying to unlock not by delegate pool owner")
	}

	// If insufficient funds in stake pool left after unlock,
	// we can't do an immediate unlock.
	// Instead we mark as unstake to prevent being used for further allocations.
	if sp.stake()-sp.TotalOffers-dp.Balance < 0 {
		sp.TotalUnStake += dp.Balance
		dp.Status = spenum.Unstaking
		return true, nil
	}

	if dp.Status == spenum.Unstaking {
		sp.TotalUnStake -= dp.Balance
	}

	transfer := state.NewTransfer(sscID, clientID, dp.Balance)
	if err := balances.AddTransfer(transfer); err != nil {
		return false, err
	}

	sp.Pools[poolID].Balance = 0
	sp.Pools[poolID].Status = spenum.Deleting

	return true, nil
}

// add offer of an allocation related to blobber owns this stake pool
func (sp *stakePool) addOffer(amount state.Balance) {
	sp.TotalOffers += amount
}

// extendOffer changes offer lock and expiration on update allocations
func (sp *stakePool) extendOffer(delta state.Balance) (err error) {
	sp.TotalOffers += delta
	return
}

//type stakePoolUpdateInfo struct {
//	stake  state.Balance // stake of all delegate pools
//	offers state.Balance // offers stake
//}

// slash represents blobber penalty; it returns number of tokens moved in
// reality, with regards to division errors
func (sp *stakePool) slash(
	alloc *StorageAllocation,
	blobID string,
	until common.Timestamp,
	wp *writePool,
	offer, slash state.Balance,
	balances chainstate.StateContextI,
) (move state.Balance, err error) {
	if offer == 0 || slash == 0 {
		return // nothing to move
	}

	if slash > offer {
		slash = offer // can't move the offer left
	}

	// the move is total movements, but it should be divided by all
	// related stake holders, that can loose some tokens due to
	// division error;
	var ap = wp.allocPool(alloc.ID, until)
	if ap == nil {
		ap = new(allocationPool)
		ap.AllocationID = alloc.ID
		ap.ExpireAt = 0
		alloc.addWritePoolOwner(alloc.Owner)
		wp.Pools.add(ap)
	}

	// offer ratio of entire stake; we are slashing only part of the offer
	// moving the tokens to allocation user; the ratio is part of entire
	// stake should be moved;
	var ratio = (float64(slash) / float64(sp.stake()))
	edbSlash := stakepool.NewStakePoolReward(blobID, spenum.Blobber)
	for id, dp := range sp.Pools {
		var dpSlash = state.Balance(float64(dp.Balance) * ratio)
		if dpSlash == 0 {
			continue
		}
		dp.Balance -= dpSlash
		ap.Balance += dpSlash

		move += dpSlash
		edbSlash.DelegateRewards[id] = -1 * int64(dpSlash)
	}
	if err := edbSlash.Emit(event.TagStakePoolReward, balances); err != nil {
		return 0, err
	}

	// move
	if blobID != "" {
		var bp, ok = ap.Blobbers.get(blobID)
		if !ok {
			ap.Blobbers.add(&blobberPool{
				BlobberID: blobID,
				Balance:   move,
			})
		} else {
			bp.Balance += move
		}
	}

	return
}

// free staked capacity of related blobber, excluding delegate pools want to
// unstake.
func (sp *stakePool) cleanCapacity(now common.Timestamp,
	writePrice state.Balance) (free int64) {

	const dryRun = true // don't update the stake pool state, just calculate
	var total, offers = sp.cleanStake(), sp.TotalOffers
	if total <= offers {
		// zero, since the offer stake (not updated) can be greater
		// then the clean stake
		return
	}
	free = int64((float64(total-offers) / float64(writePrice)) * GB)
	return
}

// free staked capacity of related blobber
func (sp *stakePool) capacity(now common.Timestamp,
	writePrice state.Balance) (free int64) {

	const dryRun = true // don't update the stake pool state, just calculate
	var total, offers = sp.stake(), sp.TotalOffers
	free = int64((float64(total-offers) / float64(writePrice)) * GB)
	return
}

// update the pool to get the stat
func (sp *stakePool) stat(_ *Config, _ string,
	now common.Timestamp, blobber *StorageNode) (stat *stakePoolStat) {

	stat = new(stakePoolStat)
	stat.ID = blobber.ID
	// Balance is full balance including all.
	stat.Balance = sp.stake()
	// Unstake is total balance of delegate pools want to unsake. But
	// can't for now. Total stake for new offers (new allocations) can
	// be calculated as (Balance - Unstake).
	stat.UnstakeTotal = sp.TotalUnStake
	// Free is free space, excluding delegate pools want to unstake.
	stat.Free = sp.cleanCapacity(now, blobber.Terms.WritePrice)
	stat.Capacity = blobber.Capacity
	stat.WritePrice = blobber.Terms.WritePrice

	stat.OffersTotal = sp.TotalOffers

	// delegate pools
	stat.Delegate = make([]delegatePoolStat, 0, len(sp.Pools))
	for poolId, dp := range sp.Pools {
		var dps = delegatePoolStat{
			ID:         poolId,
			Balance:    dp.Balance,
			DelegateID: dp.DelegateID,
			Rewards:    dp.Reward,
			//Penalty:    dp.Penalty,
			UnStake: dp.Status == spenum.Unstaking,
		}
		//stat.Penalty += dp.Penalty
		stat.Delegate = append(stat.Delegate, dps)
	}

	// rewards
	//	stat.Rewards.Charge = sp.Reward // total for all time
	//stat.Rewards.Blobber = sp.Rewards.Blobber     // total for all time
	//stat.Rewards.Validator = sp.Rewards.Validator // total for all time

	stat.Settings = sp.Settings
	return
}

// stat
type rewardsStat struct {
	Charge    state.Balance `json:"charge"`    // total for all time
	Blobber   state.Balance `json:"blobber"`   // total for all time
	Validator state.Balance `json:"validator"` // total for all time
}

type delegatePoolStat struct {
	ID         string        `json:"id"`          // blobber ID
	Balance    state.Balance `json:"balance"`     // current balance
	DelegateID string        `json:"delegate_id"` // wallet
	Rewards    state.Balance `json:"rewards"`     // total for all time
	UnStake    bool          `json:"unstake"`     // want to unstake

	TotalReward  state.Balance `json:"total_reward"`
	TotalPenalty state.Balance `json:"total_penalty"`
	Status       string        `json:"status"`
	RoundCreated int64         `json:"round_created"`
}

type stakePoolStat struct {
	ID      string        `json:"pool_id"` // pool ID
	Balance state.Balance `json:"balance"` // total balance
	Unstake state.Balance `json:"unstake"` // total unstake amount

	Free       int64         `json:"free"`        // free staked space
	Capacity   int64         `json:"capacity"`    // blobber bid
	WritePrice state.Balance `json:"write_price"` // its write price

	OffersTotal  state.Balance `json:"offers_total"` //
	UnstakeTotal state.Balance `json:"unstake_total"`
	// delegate pools
	Delegate []delegatePoolStat `json:"delegate"`
	Penalty  state.Balance      `json:"penalty"` // total for all
	// rewards
	Rewards state.Balance `json:"rewards"`

	// Settings of the stake pool
	Settings stakepool.StakePoolSettings `json:"settings"`
}

func (stat *stakePoolStat) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stat); err != nil {
		panic(err) // must never happen
	}
	return
}

func (stat *stakePoolStat) decode(input []byte) error {
	return json.Unmarshal(input, stat)
}

//
// smart contract methods
//

// getStakePool of given blobber
func (ssc *StorageSmartContract) getStakePool(blobberID datastore.Key,
	balances chainstate.StateContextI) (sp *stakePool, err error) {

	sp = newStakePool()
	err = balances.GetTrieNode(stakePoolKey(ssc.ID, blobberID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

// initial or successive method should be used by add_blobber/add_validator
// SC functions

// get existing stake pool or create new one not saving it
func (ssc *StorageSmartContract) getOrUpdateStakePool(
	conf *Config,
	providerId datastore.Key,
	providerType spenum.Provider,
	settings stakepool.StakePoolSettings,
	balances chainstate.StateContextI,
) (*stakePool, error) {
	if err := validateStakePoolSettings(settings, conf); err != nil {
		return nil, fmt.Errorf("invalid stake_pool settings: %v", err)
	}

	// the stake pool can be created by related validator
	sp, err := ssc.getStakePool(providerId, balances)
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

type stakePoolRequest struct {
	BlobberID string `json:"blobber_id,omitempty"`
	PoolID    string `json:"pool_id,omitempty"`
}

func (spr *stakePoolRequest) decode(p []byte) (err error) {
	if err = json.Unmarshal(p, spr); err != nil {
		return
	}
	return // ok
}

// unlock response
type unlockResponse struct {
	// one of the fields is set in a response, the Unstake if can't unstake
	// for now and the TokenPoolTransferResponse if has a pool had unlocked
	Unstake bool          `json:"unstake"` // max time to wait to unstake
	Balance state.Balance `json:"balance"`
}

// add delegated stake pool
func (ssc *StorageSmartContract) stakePoolLock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"can't get SC configurations: %v", err)
	}

	if t.Value < conf.StakePool.MinLock {
		return "", common.NewError("stake_pool_lock_failed",
			"too small stake to lock")
	}

	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"invalid request: %v", err)
	}

	var sp *stakePool
	if sp, err = ssc.getStakePool(spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"can't get stake pool: %v", err)
	}

	if len(sp.Pools) >= conf.MaxDelegates {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"max_delegates reached: %v, no more stake pools allowed",
			conf.MaxDelegates)
	}

	err = sp.LockPool(t, spenum.Blobber, spr.BlobberID, spenum.Active, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"stake pool digging error: %v", err)
	}

	if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"saving stake pool: %v", err)
	}

	// TO-DO: Update stake in eventDB

	return
}

// stake pool can return excess tokens from stake pool
func (ssc *StorageSmartContract) stakePoolUnlock(
	t *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't decode request: %v", err)
	}
	var sp *stakePool
	if sp, err = ssc.getStakePool(spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related stake pool: %v", err)
	}

	unstake, err := sp.empty(ssc.ID, spr.PoolID, t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"unlocking tokens: %v", err)
	}

	// the tokens can't be unlocked due to opened offers, but we mark it
	// as 'unstake' and returns maximal time to wait to unlock the pool
	if !unstake {
		// save the pool and return special result
		if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
			return "", common.NewErrorf("stake_pool_unlock_failed",
				"saving stake pool: %v", err)
		}
		return toJson(&unlockResponse{Unstake: false}), nil
	}

	amount, err := sp.UnlockPool(t.ClientID, spenum.Blobber, spr.BlobberID, spr.PoolID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed", "%v", err)
	}

	// save the pool
	if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"saving stake pool: %v", err)
	}

	return toJson(&unlockResponse{Unstake: true, Balance: amount}), nil
}
