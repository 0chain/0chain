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

// The stake() returns total stake size including delegate pools want to unstake.
func (sp *stakePool) stake() (stake currency.Coin, err error) {
	var newStake currency.Coin
	for _, dp := range sp.GetOrderedPools() {
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

	requiredBalance, err := currency.AddCoin(sp.TotalOffers, dp.Balance)
	if err != nil {
		return err
	}

	staked, err := sp.stake()
	if err != nil {
		return err
	}

	if staked < requiredBalance {
		return fmt.Errorf("insufficent stake to cover offers: existing stake %d, unlock balance %d, offers %d",
			staked, dp.Balance, sp.TotalOffers)
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
	allocationID string,
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
	edbSlash.AllocationID = allocationID
	for _, dp := range sp.GetOrderedPools() {
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
		edbSlash.DelegatePenalties[dp.DelegateID] = dpSlash
	}
	//Added New Tag for StakePoolPenalty
	if err := edbSlash.Emit(event.TagStakePoolPenalty, balances); err != nil {
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
	stake, err := sp.stake()
	if err != nil {
		return 0, err
	}

	fStake, err := stake.Float64()
	if err != nil {
		return 0, err
	}

	fWritePrice, err := writePrice.Float64()
	if err != nil {
		return 0, err
	}

	return int64((fStake / fWritePrice) * GB), nil
}

//
// smart contract methods
//

// getStakePool of given blobber
func (_ *StorageSmartContract) getStakePool(providerType spenum.Provider, providerID string,
	balances chainstate.CommonStateContextI) (sp *stakePool, err error) {
	return getStakePool(providerType, providerID, balances)
}

func getStakePoolAdapter(
	providerType spenum.Provider, providerID string, balances chainstate.CommonStateContextI,
) (sp stakepool.AbstractStakePool, err error) {
	pool, err := getStakePool(providerType, providerID, balances)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// getStakePool of given blobber
func (_ *StorageSmartContract) getStakePoolAdapter(
	providerType spenum.Provider, providerID string, balances chainstate.CommonStateContextI,
) (sp stakepool.AbstractStakePool, err error) {
	return getStakePoolAdapter(providerType, providerID, balances)
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
	gn, err := getConfig(balances)
	if err != nil {
		return "", err
	}

	st := stakepool.ValidationSettings{
		MaxStake:        gn.MaxStake,
		MinStake:        gn.MinStake,
		MaxNumDelegates: gn.MaxDelegates}

	return stakepool.StakePoolLock(
		balances,
		t,
		input,
		st,
		ssc.getStakePoolAdapter,
		func(req *stakepool.StakePoolRequest, sp stakepool.AbstractStakePool) error {
			if req.ProviderType != spenum.Blobber {
				return nil
			}

			return updateBlobberTotalStake(balances, req.ProviderID, sp)
		})
}

func updateBlobberTotalStake(
	balances chainstate.StateContextI,
	blobberID string,
	sp stakepool.AbstractStakePool) error {
	bil, err := getBlobbersInfoList(balances)
	if err != nil {
		return err
	}
	if len(bil) == 0 {
		return errors.New("no blobber found")
	}

	b, err := getBlobber(blobberID, balances)
	if err != nil {
		return fmt.Errorf("could not get blobber: %v", err)
	}
	s, err := sp.Stake()
	if err != nil {
		return fmt.Errorf("update blobber total stake failed: %v", err)
	}

	bil[b.Index].TotalStake = s
	return bil.Save(balances)
}

// stake pool can return excess tokens from stake pool
func (ssc *StorageSmartContract) stakePoolUnlock(
	t *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	return stakepool.StakePoolUnlock(balances, t, input, ssc.getStakePoolAdapter,
		func(req *stakepool.StakePoolRequest, sp stakepool.AbstractStakePool) error {
			if req.ProviderType != spenum.Blobber {
				return nil
			}

			return updateBlobberTotalStake(balances, req.ProviderID, sp)
		})
}
