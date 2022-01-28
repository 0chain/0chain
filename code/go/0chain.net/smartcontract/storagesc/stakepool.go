package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
)

// A userStakePools collects stake pools references for a user.
type userStakePools struct {
	// Pools is map blobber_id -> []pool_id.
	Pools map[datastore.Key][]datastore.Key `json:"pools"`
}

func newUserStakePools() (usp *userStakePools) {
	usp = new(userStakePools)
	usp.Pools = make(map[datastore.Key][]datastore.Key)
	return
}

// add or overwrite
func (usp *userStakePools) add(blobberID, poolID datastore.Key) {
	usp.Pools[blobberID] = append(usp.Pools[blobberID], poolID)
}

// delete by id
func (usp *userStakePools) del(blobberID, poolID datastore.Key) (empty bool) {
	var (
		list = usp.Pools[blobberID]
		i    int
	)
	for _, id := range list {
		if id == poolID {
			continue
		}
		list[i], i = id, i+1
	}
	list = list[:i]
	if len(list) == 0 {
		delete(usp.Pools, blobberID) // delete empty
	} else {
		usp.Pools[blobberID] = list // update
	}
	return len(usp.Pools) == 0
}

func (usp *userStakePools) Encode() []byte {
	var p, err = json.Marshal(usp)
	if err != nil {
		panic(err) // must never happen
	}
	return p
}

func (usp *userStakePools) Decode(p []byte) error {
	return json.Unmarshal(p, usp)
}

// save the user stake pools
func (usp *userStakePools) save(scKey, clientID datastore.Key,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(userStakePoolsKey(scKey, clientID), usp)
	return
}

// remove the entire user stake pools node
func (usp *userStakePools) remove(scKey, clientID datastore.Key,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.DeleteTrieNode(userStakePoolsKey(scKey, clientID))
	return
}

func userStakePoolsKey(scKey, clientID datastore.Key) datastore.Key {
	return datastore.Key(scKey + ":stakepool:userpools:" + clientID)
}

// stake pool internal rewards information
type stakePoolRewards struct {
	Charge    state.Balance `json:"charge"`    // blobber charge
	Blobber   state.Balance `json:"blobber"`   // blobber stake holders reward
	Validator state.Balance `json:"validator"` // validator stake holders reward
}

// delegate pool
type delegatePool struct {
	tokenpool.ZcnPool `json:"pool"`    // the pool
	MintAt            common.Timestamp `json:"mint_at"`     // last mint time
	DelegateID        datastore.Key    `json:"delegate_id"` // user
	Rewards           state.Balance    `json:"rewards"`     // total
	Penalty           state.Balance    `json:"penalty"`     // total
	UnStake           bool             `json:"unstake"`     // want to unstake
}

// stake pool settings

type stakePoolSettings struct {
	// DelegateWallet for pool owner.
	DelegateWallet string `json:"delegate_wallet"`
	// MinStake allowed.
	MinStake state.Balance `json:"min_stake"`
	// MaxStake allowed.
	MaxStake state.Balance `json:"max_stake"`
	// NumDelegates maximum allowed.
	NumDelegates int `json:"num_delegates"`
	// ServiceCharge of the blobber. The blobber gets this % (actually, value in
	// [0; 1) range). If the ServiceCharge greater than max_charge of the SC
	// then the blobber can't be registered / updated.
	ServiceCharge float64 `json:"service_charge"`
}

func (sps *stakePoolSettings) validate(conf *scConfig) (err error) {
	err = conf.validateStakeRange(sps.MinStake, sps.MaxStake)
	if err != nil {
		return
	}
	if sps.ServiceCharge < 0.0 {
		return errors.New("negative service charge")
	}
	if sps.ServiceCharge > conf.MaxCharge {
		return fmt.Errorf("service_charge (%f) is greater than"+
			" max allowed by SC (%f)", sps.ServiceCharge, conf.MaxCharge)
	}
	if sps.NumDelegates <= 0 {
		return errors.New("num_delegates <= 0")
	}
	return
}

// stake pool of a blobber

type stakePool struct {
	stakepool.StakePool

	// delegates
	Pools map[string]*delegatePool `json:"pools"`
	// TotalOffers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	TotalOffers state.Balance
	// Total amount to be un staked
	TotalUnStake state.Balance
	// total rewards information
	Rewards stakePoolRewards `json:"rewards"`
	// Settings of the stake pool.
	Settings stakePoolSettings `json:"settings"`
}

func newStakePool() *stakePool {
	bsp := stakepool.NewStakePool()
	return &stakePool{
		StakePool: *bsp,
		Pools:     make(map[string]*delegatePool),
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

// is allowed delegate wallet
func (sp *stakePool) isAllowed(id datastore.Key) bool {
	return sp.Settings.DelegateWallet == "" ||
		sp.Settings.DelegateWallet == id
}

// add delegate wallet
func (sp *stakePool) dig(t *transaction.Transaction,
	balances chainstate.StateContextI) (
	resp string, dp *delegatePool, err error) {

	if err = checkFill(t, balances); err != nil {
		return
	}

	dp = new(delegatePool)

	var transfer *state.Transfer
	if transfer, resp, err = dp.DigPool(t.Hash, t); err != nil {
		return
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return
	}

	dp.DelegateID = t.ClientID
	dp.MintAt = t.CreationDate

	sp.Pools[t.Hash] = dp

	return
}

// empty a delegate pool if possible, call update before the empty
func (sp *stakePool) empty(
	sscID,
	poolID,
	clientID string,
	balances chainstate.StateContextI,
) (resp string, unstake common.Timestamp, err error) {
	var dp, ok = sp.Pools[poolID]
	if !ok {
		return "", 0, fmt.Errorf("no such delegate pool: %q", poolID)
	}

	if dp.DelegateID != clientID {
		return "", 0, errors.New("trying to unlock not by delegate pool owner")
	}

	// If insufficient funds in stake pool left after unlock,
	// we can't do an immediate unlock.
	// Instead we mark as unstake to prevent being used for further allocations.
	if sp.stake()-sp.TotalOffers-dp.Balance < 0 {
		sp.TotalUnStake += dp.Balance
		dp.UnStake = true
		return // no errors here, handle in caller
	}

	if dp.UnStake {
		sp.TotalUnStake -= dp.Balance
	}

	var transfer *state.Transfer
	if transfer, resp, err = dp.EmptyPool(sscID, clientID, nil); err != nil {
		return
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return
	}

	delete(sp.Pools, poolID)
	return
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

func maxBalance(a, b state.Balance) state.Balance {
	if a > b {
		return a
	}
	return b
}

func minBalance(a, b state.Balance) state.Balance {
	if a < b {
		return a
	}
	return b
}

type stakePoolUpdateInfo struct {
	stake  state.Balance // stake of all delegate pools
	offers state.Balance // offers stake
}

func (sp *stakePool) orderedPools() (dps []*delegatePool) {
	dps = make([]*delegatePool, 0, len(sp.Pools))
	for _, dp := range sp.Pools {
		dps = append(dps, dp)
	}
	sort.Slice(dps, func(i, j int) bool {
		return dps[i].DelegateID < dps[j].DelegateID
	})
	return
}

// update information about the stake pool internals
func (sp *stakePool) update(conf *scConfig, sscID string, now common.Timestamp,
	balances chainstate.StateContextI) (info *stakePoolUpdateInfo, err error) {
	info = new(stakePoolUpdateInfo)
	info.stake = sp.stake() // capacity stake
	info.offers = sp.TotalOffers
	return
}

// slash represents blobber penalty; it returns number of tokens moved in
// reality, with regards to division errors
func (sp *stakePool) slash(
	alloc *StorageAllocation,
	blobID string,
	until common.Timestamp,
	wp *writePool,
	offer, slash state.Balance,
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

	for _, dp := range sp.orderedPools() {
		var one = state.Balance(float64(dp.Balance) * ratio)
		if one == 0 {
			continue
		}
		if _, _, err = dp.TransferTo(ap, one, nil); err != nil {
			return 0, fmt.Errorf("transferring blobber slash: %v", err)
		}
		dp.Penalty += one
		move += one
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
func (sp *stakePool) stat(conf *scConfig, sscKey string,
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
	for _, dp := range sp.orderedPools() {
		var dps = delegatePoolStat{
			ID:         dp.ID,
			Balance:    dp.Balance,
			DelegateID: dp.DelegateID,
			Rewards:    dp.Rewards,
			Penalty:    dp.Penalty,
			UnStake:    dp.UnStake,
		}
		stat.Penalty += dp.Penalty
		stat.Delegate = append(stat.Delegate, dps)
	}

	// rewards
	stat.Rewards.Charge = sp.Rewards.Charge       // total for all time
	stat.Rewards.Blobber = sp.Rewards.Blobber     // total for all time
	stat.Rewards.Validator = sp.Rewards.Validator // total for all time

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
	ID         datastore.Key `json:"id"`          // blobber ID
	Balance    state.Balance `json:"balance"`     // current balance
	DelegateID datastore.Key `json:"delegate_id"` // wallet
	Rewards    state.Balance `json:"rewards"`     // total for all time
	Penalty    state.Balance `json:"penalty"`     // total for all time
	UnStake    bool          `json:"unstake"`     // want to unstake

}

type stakePoolStat struct {
	ID      datastore.Key `json:"pool_id"` // pool ID
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
	Rewards rewardsStat `json:"rewards"`

	// Settings of the stake pool
	Settings stakePoolSettings `json:"settings"`
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

// user stake pools (e.g. user delegate pools)
//

// getUserStakePoolBytes of a client
func (ssc *StorageSmartContract) getUserStakePoolBytes(clientID datastore.Key,
	balances chainstate.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(userStakePoolsKey(ssc.ID, clientID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getUserStakePool of given client
func (ssc *StorageSmartContract) getUserStakePool(clientID datastore.Key,
	balances chainstate.StateContextI) (usp *userStakePools, err error) {

	var poolb []byte
	if poolb, err = ssc.getUserStakePoolBytes(clientID, balances); err != nil {
		return
	}
	usp = newUserStakePools()
	err = usp.Decode(poolb)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return
}

// getOrCreateUserStakePool of given client
func (ssc *StorageSmartContract) getOrCreateUserStakePool(
	clientID datastore.Key, balances chainstate.StateContextI) (
	usp *userStakePools, err error) {

	var poolb []byte
	poolb, err = ssc.getUserStakePoolBytes(clientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	if err == util.ErrValueNotPresent {
		return newUserStakePools(), nil
	}

	usp = newUserStakePools()
	err = usp.Decode(poolb)
	return
}

// blobber's and validator's stake pool
//

// getStakePoolBytes of a blobber
func (ssc *StorageSmartContract) getStakePoolBytes(blobberID datastore.Key,
	balances chainstate.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(stakePoolKey(ssc.ID, blobberID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getStakePool of given blobber
func (ssc *StorageSmartContract) getStakePool(blobberID datastore.Key,
	balances chainstate.StateContextI) (sp *stakePool, err error) {

	var poolb []byte
	if poolb, err = ssc.getStakePoolBytes(blobberID, balances); err != nil {
		return
	}
	sp = newStakePool()
	err = sp.Decode(poolb)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return
}

// initial or successive method should be used by add_blobber/add_validator
// SC functions

// get existing stake pool or create new one not saving it
func (ssc *StorageSmartContract) getOrCreateStakePool(conf *scConfig,
	blobberID datastore.Key, settings *stakePoolSettings,
	balances chainstate.StateContextI) (sp *stakePool, err error) {

	if err = settings.validate(conf); err != nil {
		return nil, fmt.Errorf("invalid stake_pool settings: %v", err)
	}

	// the stake pool can be created by related validator
	sp, err = ssc.getStakePool(blobberID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	if err == util.ErrValueNotPresent {
		sp, err = newStakePool(), nil
		sp.Settings.DelegateWallet = settings.DelegateWallet
	}

	sp.Settings.MinStake = settings.MinStake
	sp.Settings.MaxStake = settings.MaxStake
	sp.Settings.ServiceCharge = settings.ServiceCharge
	sp.Settings.NumDelegates = settings.NumDelegates
	return
}

type stakePoolRequest struct {
	BlobberID datastore.Key `json:"blobber_id,omitempty"`
	PoolID    datastore.Key `json:"pool_id,omitempty"`
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

	Unstake common.Timestamp `json:"unstake"` // max time to wait to unstake
	tokenpool.TokenPoolTransferResponse
}

// add delegated stake pool
func (ssc *StorageSmartContract) stakePoolLock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var conf *scConfig
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

	err = sp.LockPool(t, stakepool.Blobber, spr.BlobberID, stakepool.Active, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed", "%v", err)
	}

	if len(sp.Pools) >= conf.MaxDelegates {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"max_delegates reached: %v, no more stake pools allowed",
			conf.MaxDelegates)
	}

	var dp *delegatePool // created delegate pool
	if resp, dp, err = sp.dig(t, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"stake pool digging error: %v", err)
	}

	// add to user pools
	var usp *userStakePools
	usp, err = ssc.getOrCreateUserStakePool(t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"can't get user pools list: %v", err)
	}
	usp.add(spr.BlobberID, dp.ID) // add the new delegate pool

	if err = usp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"saving user pools: %v", err)
	}

	if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"saving stake pool: %v", err)
	}

	// TO-DO: Update stake in eventDB

	return
}

// stake pool can return excess tokens from stake pool
func (ssc *StorageSmartContract) stakePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var (
		sp *stakePool
	)

	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't decode request: %v", err)
	}

	if sp, err = ssc.getStakePool(spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related stake pool: %v", err)
	}

	amount, err := sp.UnlockPool(t, stakepool.Blobber, spr.BlobberID, spr.PoolID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed", "%v", err)
	}
	resp = strconv.FormatInt(int64(amount), 10)
	logging.Logger.Info("piers stake pool unlock", zap.String("unlocked", resp))

	var usp *userStakePools
	usp, err = ssc.getUserStakePool(t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related user stake pools: %v", err)
	}

	var unstake common.Timestamp
	resp, unstake, err = sp.empty(ssc.ID, spr.PoolID, t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"unlocking tokens: %v", err)
	}

	// the tokens can't be unlocked due to opened offers, but we mark it
	// as 'unstake' and returns maximal time to wait to unlock the pool
	if unstake > 0 {
		// save the pool and return special result
		if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
			return "", common.NewErrorf("stake_pool_unlock_failed",
				"saving stake pool: %v", err)
		}
		return toJson(&unlockResponse{Unstake: unstake}), nil
	}

	if !usp.del(spr.BlobberID, spr.PoolID) {
		err = usp.save(ssc.ID, t.ClientID, balances)
	} else {
		err = usp.remove(ssc.ID, t.ClientID, balances)
	}

	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"saving user pools: %v", err)
	}

	// save the pool
	if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"saving stake pool: %v", err)
	}

	// TO-DO: Update stake in eventDB

	return
}

//
// stat
//

const cantGetStakePoolMsg = "can't get related stake pool"

// statistic for all locked tokens of a stake pool
func (ssc *StorageSmartContract) getStakePoolStatHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
	resp interface{}, err error) {

	var (
		blobberID = datastore.Key(params.Get("blobber_id"))
		conf      *scConfig
		blobber   *StorageNode
		sp        *stakePool
	)

	if conf, err = ssc.getConfig(balances, false); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg)
	}

	if blobber, err = ssc.getBlobber(blobberID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetBlobberMsg)
	}

	if sp, err = ssc.getStakePool(blobberID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetStakePoolMsg)
	}

	return sp.stat(conf, ssc.ID, common.Now(), blobber), nil
}

type userPoolStat struct {
	Pools map[datastore.Key][]*delegatePoolStat `json:"pools"`
}

// user oriented statistic
func (ssc *StorageSmartContract) getUserStakePoolStatHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = datastore.Key(params.Get("client_id"))
		usp      *userStakePools
	)

	usp, err = ssc.getUserStakePool(clientID, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get user stake pool")
	}

	var ups = new(userPoolStat)
	ups.Pools = make(map[datastore.Key][]*delegatePoolStat)

	for blobberID, poolIDs := range usp.Pools {
		var sp *stakePool
		if sp, err = ssc.getStakePool(blobberID, balances); err != nil {
			return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetStakePoolMsg)
		}

		for _, id := range poolIDs {
			var dp, ok = sp.Pools[id]
			if !ok {
				return nil, common.NewErrNoResource("missing delegate pool")
			}
			var dps = delegatePoolStat{
				ID:         dp.ID,
				Balance:    dp.Balance,
				DelegateID: dp.DelegateID,
				Rewards:    dp.Rewards,
				Penalty:    dp.Penalty,
			}
			ups.Pools[blobberID] = append(ups.Pools[blobberID], &dps)
		}
	}

	return ups, nil
}
