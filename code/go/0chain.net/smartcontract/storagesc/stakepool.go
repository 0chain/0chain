package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/currency"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/util"
)

//msgp:ignore unlockResponse stakePoolStat stakePoolRequest delegatePoolStat rewardsStat
//go:generate msgp -io=false -tests=false -unexported=true -v

func validateStakePoolSettings(
	sps stakepool.Settings,
	conf *Config,
) error {
	err := conf.validateStakeRange(sps.MinStake, sps.MaxStake)
	if err != nil {
		return err
	}
	if sps.ServiceChargeRatio < 0.0 {
		return errors.New("negative service charge")
	}
	if sps.ServiceChargeRatio > conf.MaxCharge {
		return fmt.Errorf("service_charge (%f) is greater than"+
			" max allowed by SC (%f)", sps.ServiceChargeRatio, conf.MaxCharge)
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
	TotalOffers currency.Coin `json:"total_offers"`
	// Total amount to be un staked
	TotalUnStake currency.Coin `json:"total_un_stake"`
}

func newStakePool() *stakePool {
	bsp := stakepool.NewStakePool()
	return &stakePool{
		StakePool: *bsp,
	}
}

// stake pool key for the storage SC and  blobber
func stakePoolKey(providerType spenum.Provider, providerID string) datastore.Key {
	return providerType.String() + ":stakepool:" + providerID
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
func (sp *stakePool) save(providerType spenum.Provider, providerID string,
	balances chainstate.StateContextI) (err error) {
	r, err := balances.InsertTrieNode(stakePoolKey(providerType, providerID), sp)
	logging.Logger.Debug("after stake pool save", zap.String("root", r))

	sp.emitSaveEvent(providerType, providerID, balances)

	return
}

func (sp *stakePool) emitSaveEvent(providerType spenum.Provider, providerID string, balances chainstate.StateContextI) {
	data := dbs.DbUpdates{
		Id:      providerID,
		Updates: map[string]interface{}{},
	}
	switch providerType {
	case spenum.Blobber:
		data.Updates["offers_total"] = int64(sp.TotalOffers)
		balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, providerID, data)
	case spenum.Validator:
		//todo: emit validator event
	default:
		logging.Logger.Error("invalid providerType in stakepool SaveEvent")
	}
}

// The cleanStake() is stake amount without delegate pools want to unstake.
func (sp *stakePool) cleanStake() (stake currency.Coin, err error) {
	staked, err := sp.stake()
	if err != nil {
		return 0, err
	}

	return staked - sp.TotalUnStake, nil
}

func (sp *stakePool) stakeByProvider(providerType spenum.Provider, providerID string, balances chainstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}
	sp.emitStakeEvent(providerType, providerID, staked, balances)

	return nil
}

// The stake() returns total stake size including delegate pools want to unstake.
func (sp *stakePool) stake() (stake currency.Coin, err error) {
	var newStake currency.Coin
	for _, dp := range sp.Pools {
		newStake, err = currency.AddCoin(stake, dp.Balance)
		if err != nil {
			return
		}
		stake = newStake
	}
	return
}

func (sp *stakePool) emitStakeEvent(providerType spenum.Provider, providerID string, staked currency.Coin, balances chainstate.StateContextI) {
	logging.Logger.Info("emitting stake event")
	data := dbs.DbUpdates{
		Id:      providerID,
		Updates: map[string]interface{}{},
	}
	switch providerType {
	case spenum.Blobber:
		data.Updates["total_stake"] = int64(staked)
		balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, providerID, data)
	case spenum.Validator:
		data.Updates["stake"] = int64(staked)
		balances.EmitEvent(event.TypeStats, event.TagUpdateValidator, providerID, data)
	default:
		logging.Logger.Error("invalid providerType in stakepool StakeEvent")
	}
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

	totalBalance, err := currency.AddCoin(sp.TotalOffers, dp.Balance)
	if err != nil {
		return false, err
	}

	staked, err := sp.stake()
	if err != nil {
		return false, err
	}
	if staked < totalBalance {
		if dp.Status != spenum.Unstaking {
			totalUnStake, err := currency.AddCoin(sp.TotalUnStake, dp.Balance)
			if err != nil {
				return false, err
			}
			sp.TotalUnStake = totalUnStake

			dp.Status = spenum.Unstaking
		}
		return true, nil
	}

	if dp.Status == spenum.Unstaking {
		totalUnstake, err := currency.MinusCoin(sp.TotalUnStake, dp.Balance)
		if err != nil {
			return false, err
		}
		sp.TotalUnStake = totalUnstake
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
func (sp *stakePool) addOffer(amount currency.Coin) error {
	newTotalOffers, err := currency.AddCoin(sp.TotalOffers, amount)
	if err != nil {
		return err
	}
	sp.TotalOffers = newTotalOffers
	return nil
}

// add offer of an allocation related to blobber owns this stake pool
func (sp *stakePool) reduceOffer(amount currency.Coin) error {
	newTotalOffers, err := currency.MinusCoin(sp.TotalOffers, amount)
	if err != nil {
		return err
	}
	sp.TotalOffers = newTotalOffers
	return nil
}

// remove offer of an allocation related to blobber owns this stake pool
func (sp *stakePool) removeOffer(amount currency.Coin) error {
	if amount > sp.TotalOffers {
		return fmt.Errorf("amount to be removed %v > total offer present %v", amount, sp.TotalOffers)
	}
	sp.TotalOffers -= amount
	return nil
}

// slash represents blobber penalty; it returns number of tokens moved in
// reality, in regard to division errors
func (sp *stakePool) slash(
	blobID string,
	offer, slash currency.Coin,
	balances chainstate.StateContextI,
) (move currency.Coin, err error) {
	if offer == 0 || slash == 0 {
		return // nothing to move
	}

	if slash > offer {
		slash = offer // can't move the offer left
	}

	staked, err := sp.stake()
	if err != nil {
		return 0, err
	}

	// offer ratio of entire stake; we are slashing only part of the offer
	// moving the tokens to allocation user; the ratio is part of entire
	// stake should be moved;
	var ratio = float64(slash) / float64(staked)
	edbSlash := stakepool.NewStakePoolReward(blobID, spenum.Blobber)
	for id, dp := range sp.Pools {
		dpSlash, err := currency.MultFloat64(dp.Balance, ratio)
		if err != nil {
			return 0, err
		}

		if dpSlash == 0 {
			continue
		}

		if balance, err := currency.MinusCoin(dp.Balance, dpSlash); err != nil {
			return 0, err
		} else {
			dp.Balance = balance
		}
		move, err = currency.AddCoin(move, dpSlash)
		if err != nil {
			return 0, err
		}
		edbSlash.DelegateRewards[id] = -1 * int64(dpSlash)
	}
	// todo we should slash from stake pools not rewards. 0chain issue 1495
	if err := edbSlash.Emit(event.TagStakePoolReward, balances); err != nil {
		return 0, err
	}

	return
}

// unallocated capacity of related blobber, excluding delegate pools want to
// unstake.
func (sp *stakePool) unallocatedCapacity(writePrice currency.Coin) (free int64, err error) {

	staked, err := sp.stake()
	if err != nil {
		return
	}
	var total, offers = staked, sp.TotalOffers
	logging.Logger.Debug("clean_capacity", zap.Int64("total", int64(total)), zap.Int64("offers",
		int64(offers)), zap.Int64("writePrice", int64(writePrice)))
	if total <= offers {
		// zero, since the offer stake (not updated) can be greater than the clean stake
		return
	}
	free = int64((float64(total-offers) / float64(writePrice)) * GB)
	return
}

func (sp *stakePool) stakedCapacity(writePrice currency.Coin) (int64, error) {

	cleanStake, err := sp.cleanStake()
	if err != nil {
		return 0, err
	}

	fcleanStake, err := cleanStake.Float64()
	if err != nil {
		return 0, err
	}

	fWritePrice, err := writePrice.Float64()
	if err != nil {
		return 0, err
	}

	return int64((fcleanStake / fWritePrice) * GB), nil
}

type delegatePoolStat struct {
	ID         string        `json:"id"`          // blobber ID
	Balance    currency.Coin `json:"balance"`     // current balance
	DelegateID string        `json:"delegate_id"` // wallet
	Rewards    currency.Coin `json:"rewards"`     // total for all time
	UnStake    bool          `json:"unstake"`     // want to unstake

	TotalReward  currency.Coin `json:"total_reward"`
	TotalPenalty currency.Coin `json:"total_penalty"`
	Status       string        `json:"status"`
	RoundCreated int64         `json:"round_created"`
}

// swagger:model stakePoolStat
type stakePoolStat struct {
	ID      string        `json:"pool_id"` // pool ID
	Balance currency.Coin `json:"balance"` // total balance
	Unstake currency.Coin `json:"unstake"` // total unstake amount

	Free       int64         `json:"free"`        // free staked space
	Capacity   int64         `json:"capacity"`    // blobber bid
	WritePrice currency.Coin `json:"write_price"` // its write price

	OffersTotal  currency.Coin `json:"offers_total"` //
	UnstakeTotal currency.Coin `json:"unstake_total"`
	// delegate pools
	Delegate []delegatePoolStat `json:"delegate"`
	Penalty  currency.Coin      `json:"penalty"` // total for all
	// rewards
	Rewards currency.Coin `json:"rewards"`

	// Settings of the stake pool
	Settings stakepool.Settings `json:"settings"`
}

//
// smart contract methods
//

// getStakePool of given blobber
func (ssc *StorageSmartContract) getStakePool(providerType spenum.Provider, providerID string,
	balances chainstate.CommonStateContextI) (sp *stakePool, err error) {
	sp = newStakePool()
	err = balances.GetTrieNode(stakePoolKey(providerType, providerID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func getProviderStakePool(
	id datastore.Key,
	pType spenum.Provider,
	balances chainstate.CommonStateContextI,
) (sp *stakePool, err error) {
	sp = newStakePool()
	err = balances.GetTrieNode(stakePoolKey(pType, id), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func getStakePool(
	blobberID datastore.Key, balances chainstate.CommonStateContextI,
) (sp *stakePool, err error) {
	sp = newStakePool()
	err = balances.GetTrieNode(stakePoolKey(spenum.Blobber, blobberID), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

// initial or successive method should be used by add_blobber/add_validator
// SC functions

// get existing stake pool or create new one not saving it
func (ssc *StorageSmartContract) getOrCreateStakePool(
	conf *Config,
	providerType spenum.Provider,
	providerId datastore.Key,
	settings stakepool.Settings,
	balances chainstate.StateContextI,
) (*stakePool, error) {
	if err := validateStakePoolSettings(settings, conf); err != nil {
		return nil, fmt.Errorf("invalid stake_pool settings: %v", err)
	}

	// the stake pool can be created by related validator
	sp, err := ssc.getStakePool(providerType, providerId, balances)
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
	sp.Settings.ServiceChargeRatio = settings.ServiceChargeRatio
	sp.Settings.MaxNumDelegates = settings.MaxNumDelegates
	return sp, nil
}

type stakePoolRequest struct {
	ProviderType spenum.Provider `json:"provider_type,omitempty"`
	ProviderID   string          `json:"provider_id,omitempty"`
	PoolID       string          `json:"pool_id,omitempty"`
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
	Balance currency.Coin `json:"balance"`
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
	if sp, err = ssc.getStakePool(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"can't get stake pool: %v", err)
	}

	if len(sp.Pools) >= conf.MaxDelegates {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"max_delegates reached: %v, no more stake pools allowed",
			conf.MaxDelegates)
	}

	err = sp.LockPool(t, spr.ProviderType, spr.ProviderID, spenum.Active, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"stake pool digging error: %v", err)
	}

	if err = sp.save(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"saving stake pool: %v", err)
	}

	err = sp.stakeByProvider(spr.ProviderType, spr.ProviderID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"stake pool staking error: %v", err)
	}

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
	if sp, err = ssc.getStakePool(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related stake pool: %v", err)
	}

	dp, ok := sp.Pools[spr.PoolID]
	if !ok {
		return "", common.NewErrorf("stake_pool_unlock_failed", "no such delegate pool: %v ", spr.PoolID)
	}

	// if StakeAt has valid value and lock period is less than MinLockPeriod
	if dp.StakedAt > 0 {
		stakedAt := common.ToTime(dp.StakedAt)
		minLockPeriod := config.SmartContractConfig.GetDuration("stakepool.min_lock_period")
		if !stakedAt.Add(minLockPeriod).Before(time.Now()) {
			return "", common.NewErrorf("stake_pool_unlock_failed", "token can only be unstaked till: %s", stakedAt.Add(minLockPeriod))
		}
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
		if err = sp.save(spr.ProviderType, spr.ProviderID, balances); err != nil {
			return "", common.NewErrorf("stake_pool_unlock_failed",
				"saving stake pool: %v", err)
		}
		err = sp.stakeByProvider(spr.ProviderType, spr.ProviderID, balances)
		if err != nil {
			return "", common.NewErrorf("stake_pool_unlock_failed",
				"stake pool staking error: %v", err)
		}

		return toJson(&unlockResponse{Unstake: false}), nil
	}

	amount, err := sp.UnlockClientStakePool(t.ClientID, spr.ProviderType, spr.ProviderID, spr.PoolID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed", "%v", err)
	}

	// save the pool
	if err = sp.save(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"saving stake pool: %v", err)
	}

	err = sp.stakeByProvider(spr.ProviderType, spr.ProviderID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"stake pool staking error: %v", err)
	}

	return toJson(&unlockResponse{Unstake: true, Balance: amount}), nil
}
