package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
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
	*stakepool.StakePool
	// TotalOffers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	TotalOffers    currency.Coin `json:"total_offers"`
	isOfferChanged bool          `json:"-" msg:"-"`
	TotalUnStake   currency.Coin `json:"total_un_stake"` // Total amount to be un staked
}

func newStakePool() *stakePool {
	bsp := stakepool.NewStakePool()
	return &stakePool{
		StakePool: bsp,
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

// Save the stake pool
func (sp *stakePool) Save(providerType spenum.Provider, providerID string,
	balances chainstate.StateContextI) error {
	_, err := balances.InsertTrieNode(stakePoolKey(providerType, providerID), sp)
	if err != nil {
		return err
	}

	if sp.isOfferChanged {
		switch providerType {
		case spenum.Blobber:
			tag, data := event.NewUpdateBlobberTotalOffersEvent(providerID, sp.TotalOffers)
			balances.EmitEvent(event.TypeStats, tag, providerID, data)
			sp.isOfferChanged = false
		case spenum.Validator:
			// TODO: perhaps implement validator stake update events
		}
	}

	return nil
}

// The cleanStake() is stake amount without delegate pools want to unstake.
func (sp *stakePool) cleanStake() (stake currency.Coin, err error) {
	staked, err := sp.stake()
	if err != nil {
		return 0, err
	}

	return staked - sp.TotalUnStake, nil
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

// empty a delegate pool if possible, call update before the empty
func (sp *stakePool) Empty(
	sscID,
	poolID,
	clientID string,
	balances chainstate.StateContextI,
) error {
	var dp, ok = sp.Pools[poolID]
	if !ok {
		return fmt.Errorf("no such delegate pool: %q", poolID)
	}

	if dp.DelegateID != clientID {
		return errors.New("trying to unlock not by delegate pool owner")
	}

	// If insufficient funds in stake pool left after unlock,
	// we can't do an immediate unlock.
	// Instead we mark as unstake to prevent being used for further allocations.

	requiredBalance, err := currency.AddCoin(sp.TotalOffers, dp.Balance)
	if err != nil {
		return err
	}

	staked, err := sp.stake()
	if err != nil {
		return err
	}
	if staked < requiredBalance {
		if dp.Status != spenum.Unstaking {
			totalUnStake, err := currency.AddCoin(sp.TotalUnStake, dp.Balance)
			if err != nil {
				return err
			}
			//we are locking current pool and hold the tokens, they won't be counted in clean stake after that
			sp.TotalUnStake = totalUnStake

			dp.Status = spenum.Unstaking
		}
		return nil
	}

	//clean up unstaking lock
	if dp.Status == spenum.Unstaking {
		totalUnstake := currency.Coin(0)
		if sp.TotalUnStake > dp.Balance {
			totalUnstake, err = currency.MinusCoin(sp.TotalUnStake, dp.Balance)
			if err != nil {
				return err
			}
		}
		//we can release these tokens now so UnStake hold can now be released
		sp.TotalUnStake = totalUnstake
	}

	transfer := state.NewTransfer(sscID, clientID, dp.Balance)
	if err := balances.AddTransfer(transfer); err != nil {
		return err
	}

	sp.Pools[poolID].Balance = 0
	sp.Pools[poolID].Status = spenum.Deleted

	return nil
}

// add offer of an allocation related to blobber owns this stake pool
func (sp *stakePool) addOffer(amount currency.Coin) error {
	newTotalOffers, err := currency.AddCoin(sp.TotalOffers, amount)
	if err != nil {
		return err
	}
	sp.TotalOffers = newTotalOffers
	sp.isOfferChanged = true
	return nil
}

// add offer of an allocation related to blobber owns this stake pool
func (sp *stakePool) reduceOffer(amount currency.Coin) error {
	newTotalOffers, err := currency.MinusCoin(sp.TotalOffers, amount)
	if err != nil {
		return err
	}
	sp.TotalOffers = newTotalOffers
	sp.isOfferChanged = true
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
	edbSlash := stakepool.NewStakePoolReward(blobID, spenum.Blobber, spenum.ChallengeSlashPenalty)
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
		edbSlash.DelegatePenalties[id] = dpSlash
	}
	// todo we should slash from stake pools not rewards. 0chain issue 1495
	if err := edbSlash.Emit(event.TagStakePoolReward, balances); err != nil {
		return 0, err
	}

	return
}

func unallocatedCapacity(writePrice, total, offers currency.Coin) (free int64, err error) {
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

//
// smart contract methods
//

// getStakePool of given blobber
func (ssc *StorageSmartContract) getStakePool(providerType spenum.Provider, providerID string,
	balances chainstate.CommonStateContextI) (sp *stakePool, err error) {
	return getStakePool(providerType, providerID, balances)
}

// getStakePool of given blobber
func (ssc *StorageSmartContract) getStakePoolAdapter(providerType spenum.Provider, providerID string,
	balances chainstate.CommonStateContextI) (sp stakepool.AbstractStakePool, err error) {
	pool, err := getStakePool(providerType, providerID, balances)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func getStakePool(providerType spenum.Provider, providerID datastore.Key, balances chainstate.CommonStateContextI) (
	sp *stakePool, err error) {
	sp = newStakePool()
	err = balances.GetTrieNode(stakePoolKey(providerType, providerID), sp)
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

func (ssc *StorageSmartContract) createStakePool(
	conf *Config,
	settings stakepool.Settings,
) (*stakePool, error) {
	if err := validateStakePoolSettings(settings, conf); err != nil {
		return nil, fmt.Errorf("invalid stake_pool settings: %v", err)
	}

	sp := newStakePool()
	sp.Settings.DelegateWallet = settings.DelegateWallet
	sp.Minter = chainstate.MinterStorage
	sp.Settings.MinStake = settings.MinStake
	sp.Settings.MaxStake = settings.MaxStake
	sp.Settings.ServiceChargeRatio = settings.ServiceChargeRatio
	sp.Settings.MaxNumDelegates = settings.MaxNumDelegates

	return sp, nil
}

type stakePoolRequest struct {
	ProviderType spenum.Provider `json:"provider_type,omitempty"`
	ProviderID   string          `json:"provider_id,omitempty"`
}

func (spr *stakePoolRequest) decode(p []byte) (err error) {
	if err = json.Unmarshal(p, spr); err != nil {
		return
	}
	return // ok
}

// add delegated stake pool
func (ssc *StorageSmartContract) stakePoolLock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {
	return stakepool.StakePoolLock(t, input, balances, ssc.getStakePoolAdapter)
}

// stake pool can return excess tokens from stake pool
func (ssc *StorageSmartContract) stakePoolUnlock(
	t *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	return stakepool.StakePoolUnlock(t, input, balances, ssc.getStakePoolAdapter)
}
