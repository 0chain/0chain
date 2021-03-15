package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
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

// offerPool represents stake tokens of a blobber locked
// for an allocation, it required for cases where blobber
// changes terms or changes its capacity including reducing
// the capacity to zero; it implemented not as a token
// pool, but as set or values
type offerPool struct {
	Lock   state.Balance    `json:"lock"`   // offer stake
	Expire common.Timestamp `json:"expire"` // offer expiration
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
	Interests         state.Balance    `json:"interests"`   // total
	Rewards           state.Balance    `json:"rewards"`     // total
	Penalty           state.Balance    `json:"penalty"`     // total
	Unstake           common.Timestamp `json:"unstake"`     // want to unstake
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
	// delegates
	Pools map[string]*delegatePool `json:"pools"`
	// offers (allocations)
	// Offers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	Offers map[string]*offerPool `json:"offers"`
	// total rewards information
	Rewards stakePoolRewards `json:"rewards"`
	// Settings of the stake pool.
	Settings stakePoolSettings `json:"settings"`
}

func newStakePool() *stakePool {
	return &stakePool{
		Pools:  make(map[string]*delegatePool),
		Offers: make(map[string]*offerPool),
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

// offersStake returns stake required by currently open offers;
// the method remove expired offers from internal offers pool
func (sp *stakePool) offersStake(now common.Timestamp, dry bool) (
	os state.Balance) {

	for allocID, off := range sp.Offers {
		if off.Expire <= now {
			if !dry {
				delete(sp.Offers, allocID) //remove expired
			}
			continue // an expired offer
		}
		os += off.Lock
	}
	return
}

// save the stake pool
func (sp *stakePool) save(sscKey, blobberID string,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(stakePoolKey(sscKey, blobberID), sp)
	return
}

// The cleanStake() is stake amount without delegate pools want to unstake.
func (sp *stakePool) cleanStake() (stake state.Balance) {
	for _, dp := range sp.Pools {
		if dp.Unstake > 0 {
			continue // don't count stake pools want to unstake
		}
		stake += dp.Balance
	}
	return
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

// The timeToUnstake is time where given balance can be unstaked. The
// timeToUnstake should be called after 'unpdate' and before the new pool
// will be marked as unstake.
func (sp *stakePool) timeToUnstake(dp *delegatePool) (
	tp common.Timestamp, err error) {

	if len(sp.Offers) == 0 {
		return 0, fmt.Errorf("invalid state: no offer, but can't unlock"+
			" tokens trying to mark them as 'unstake': %s", dp.ID)
	}

	var (
		dpb     = dp.Balance
		ops     = make([]*offerPool, 0, len(sp.Offers))
		unstake state.Balance
	)

	// calculate total balance of all pools marked as 'unstake'
	for _, dx := range sp.Pools {
		if dx.Unstake > 0 {
			unstake += dx.Balance
		}
	}

	// sort offer by expiration (earlier first)

	for _, op := range sp.Offers {
		ops = append(ops, op)
	}

	// since, the timeToUnstake called after the 'unpdate' then all expired
	// offer had removed; and we shouldn't care about the expiration

	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Expire < ops[j].Expire
	})

	// range over opened offers
	for _, op := range ops {
		// skip already 'unstaked' pools first
		if unstake > 0 {
			if unstake -= op.Lock; unstake < 0 {
				dpb -= (-unstake)
				unstake = 0
			}
		}
		// after the reduction
		if unstake > 0 {
			continue // continue the unstake reduction
		}
		// ok, now all already unstaked pools are skipped, let's reduce dpb
		if dpb > 0 {
			dpb -= op.Lock
		}
		if dpb <= 0 {
			return op.Expire, nil // expire here
		}
	}

	// ok, then last expiration is what we need, length of the offers has
	// already checked
	return ops[len(ops)-1].Expire, nil
}

// empty a delegate pool if possible, call update before the empty
func (sp *stakePool) empty(sscID, poolID, clientID string,
	info *stakePoolUpdateInfo, balances chainstate.StateContextI) (
	resp string, unstake common.Timestamp, err error) {

	var dp, ok = sp.Pools[poolID]
	if !ok {
		return "", 0, fmt.Errorf("no such delegate pool: %q", poolID)
	}

	if dp.DelegateID != clientID {
		return "", 0, errors.New("trying to unlock not by delegate pool owner")
	}

	if info.stake-info.offers-dp.Balance < 0 {
		// is marked as 'unstake'
		if dp.Unstake > 0 {
			return "", 0, errors.New("the stake pool locked for opened " +
				"offers and already marked as 'unstake'")
		}
		if unstake, err = sp.timeToUnstake(dp); err != nil {
			return "", 0, err // return the error as is
		}
		// mark the delegate pool as pool to unstake, keep max time to wait
		// to unstake; save the mark after
		dp.Unstake = unstake
		return // no errors here, handle in caller
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
func (sp *stakePool) addOffer(alloc *StorageAllocation,
	balloc *BlobberAllocation) {

	sp.Offers[alloc.ID] = &offerPool{
		Lock: state.Balance(
			sizeInGB(balloc.Size) * float64(balloc.Terms.WritePrice),
		),
		Expire: alloc.Until(),
	}
}

// findOffer by allocation id or nil
func (sp *stakePool) findOffer(allocID string) *offerPool {
	return sp.Offers[allocID]
}

// extendOffer changes offer lock and expiration on update allocations
func (sp *stakePool) extendOffer(alloc *StorageAllocation,
	balloc *BlobberAllocation) (err error) {

	var (
		op      = sp.findOffer(alloc.ID)
		newLock = state.Balance(sizeInGB(balloc.Size) *
			float64(balloc.Terms.WritePrice))
	)
	if op == nil {
		return errors.New("missing offer pool for " + alloc.ID)
	}
	// unlike a write pool, here we can reduce a lock
	op.Lock = newLock
	op.Expire = alloc.Until()
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
	minted state.Balance // minted tokens
}

// mintPool for a period
func (sp *stakePool) mintPool(sscID string, dp *delegatePool,
	now common.Timestamp, rate float64, period common.Timestamp,
	balances chainstate.StateContextI) (mint state.Balance, err error) {

	var at = dp.MintAt // last periodic mint

	for ; at+period < now; at += period {
		mint += state.Balance(rate * float64(dp.Balance))
	}

	dp.MintAt = at // update last minting time

	if mint == 0 {
		return // no mints for the pool
	}

	err = balances.AddMint(&state.Mint{
		Minter:     sscID,         // storage SC
		ToClientID: dp.DelegateID, // delegate wallet
		Amount:     mint,          // move total mints at once
	})

	if err != nil {
		return mint, fmt.Errorf("adding mint: %v", err)
	}

	dp.Interests += mint
	return
}

// interests not payed yet (virtual interests)
func (sp *stakePool) interests(dp *delegatePool, now common.Timestamp,
	rate float64, period common.Timestamp) (mint state.Balance) {

	if period == 0 {
		return // avoid infinity loop
	}

	for at := dp.MintAt; at+period < now; at += period {
		mint += state.Balance(rate * float64(dp.Balance))
	}
	return
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

// pay interests for stake pool
func (sp *stakePool) minting(conf *scConfig, sscID string,
	now common.Timestamp, balances chainstate.StateContextI) (
	minted state.Balance, err error) {

	if !conf.canMint() {
		return // can't mint anymore, max_mint reached
	}

	if len(sp.Pools) == 0 {
		return
	}

	var (
		rate   = conf.StakePool.InterestRate                // %
		period = toSeconds(conf.StakePool.InterestInterval) // interests period
	)

	if period == 0 {
		return // invalid period
	}

	// ordered

	var mint state.Balance
	for _, dp := range sp.orderedPools() {
		mint, err = sp.mintPool(sscID, dp, now, rate, period, balances)
		if err != nil {
			return
		}
		minted += mint
	}

	return
}

// update information about the stake pool internals
func (sp *stakePool) update(conf *scConfig, sscID string, now common.Timestamp,
	balances chainstate.StateContextI) (info *stakePoolUpdateInfo, err error) {

	// mints
	var mint state.Balance
	if mint, err = sp.minting(conf, sscID, now, balances); err != nil {
		return
	}

	info = new(stakePoolUpdateInfo)
	info.stake = sp.stake()                  // capacity stake
	info.offers = sp.offersStake(now, false) // offers stake
	info.minted = mint                       // minted tokens
	return
}

// slash represents blobber penalty; it returns number of tokens moved in
// reality, with regards to division errors
func (sp *stakePool) slash(allocID, blobID string, until common.Timestamp,
	wp *writePool, offer, slash state.Balance) (
	move state.Balance, err error) {

	if offer == 0 || slash == 0 {
		return // nothing to move
	}

	if slash > offer {
		slash = offer // can't move the offer left
	}

	// the move is total movements, but it should be divided by all
	// related stake holders, that can loose some tokens due to
	// division error;

	var ap = wp.allocPool(allocID, until)
	if ap == nil {
		ap = new(allocationPool)
		ap.AllocationID = allocID
		ap.ExpireAt = 0
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
	var total, offers = sp.cleanStake(), sp.offersStake(now, dryRun)
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
	var total, offers = sp.stake(), sp.offersStake(now, dryRun)
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
	stat.Unstake = sp.stake() - sp.cleanStake()
	// Free is free space, excluding delegate pools want to unstake.
	stat.Free = sp.cleanCapacity(now, blobber.Terms.WritePrice)
	stat.Capacity = blobber.Capacity
	stat.WritePrice = blobber.Terms.WritePrice

	// offers
	stat.Offers = make([]offerPoolStat, 0, len(sp.Offers))
	for allocID, off := range sp.Offers {
		stat.Offers = append(stat.Offers, offerPoolStat{
			Lock:         off.Lock,
			Expire:       off.Expire,
			AllocationID: allocID,
			IsExpired:    off.Expire < now,
		})
		if off.Expire >= now {
			stat.OffersTotal += off.Lock
		}
	}

	var (
		rate   = conf.StakePool.InterestRate
		period = toSeconds(conf.StakePool.InterestInterval)
	)

	// delegate pools
	stat.Delegate = make([]delegatePoolStat, 0, len(sp.Pools))
	for _, dp := range sp.orderedPools() {
		var dps = delegatePoolStat{
			ID:         dp.ID,
			Balance:    dp.Balance,
			DelegateID: dp.DelegateID,
			Interests:  dp.Interests,
			Rewards:    dp.Rewards,
			Penalty:    dp.Penalty,
			Unstake:    dp.Unstake,
		}
		stat.Interests += dp.Rewards
		stat.Penalty += dp.Penalty
		if conf.canMint() {
			dps.PendingInterests = sp.interests(dp, now, rate, period)
		}
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

type offerPoolStat struct {
	Lock         state.Balance    `json:"lock"`
	Expire       common.Timestamp `json:"expire"`
	AllocationID string           `json:"allocation_id"`
	IsExpired    bool             `json:"is_expired"`
}

type rewardsStat struct {
	Charge    state.Balance `json:"charge"`    // total for all time
	Blobber   state.Balance `json:"blobber"`   // total for all time
	Validator state.Balance `json:"validator"` // total for all time
}

type delegatePoolStat struct {
	ID               datastore.Key    `json:"id"`                // blobber ID
	Balance          state.Balance    `json:"balance"`           // current balance
	DelegateID       datastore.Key    `json:"delegate_id"`       // wallet
	Rewards          state.Balance    `json:"rewards"`           // total for all time
	Interests        state.Balance    `json:"interests"`         // total for all time (payed)
	Penalty          state.Balance    `json:"penalty"`           // total for all time
	PendingInterests state.Balance    `json:"pending_interests"` // not payed yet
	Unstake          common.Timestamp `json:"unstake"`           // want to unstake
}

type stakePoolStat struct {
	ID      datastore.Key `json:"pool_id"` // pool ID
	Balance state.Balance `json:"balance"` // total balance
	Unstake state.Balance `json:"unstake"` // total unstake amount

	Free       int64         `json:"free"`        // free staked space
	Capacity   int64         `json:"capacity"`    // blobber bid
	WritePrice state.Balance `json:"write_price"` // its write price

	Offers      []offerPoolStat `json:"offers"`       //
	OffersTotal state.Balance   `json:"offers_total"` //
	// delegate pools
	Delegate  []delegatePoolStat `json:"delegate"`
	Interests state.Balance      `json:"interests"` // total for all (TO REMOVE)
	Penalty   state.Balance      `json:"penalty"`   // total for all
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

func (ssc *StorageSmartContract) updateSakePoolOffer(ba *BlobberAllocation,
	alloc *StorageAllocation, balances chainstate.StateContextI) (err error) {

	var sp *stakePool
	if sp, err = ssc.getStakePool(ba.BlobberID, balances); err != nil {
		return fmt.Errorf("can't get stake pool of %s: %v", ba.BlobberID,
			err)
	}
	if err = sp.extendOffer(alloc, ba); err != nil {
		return fmt.Errorf("can't change stake pool offer %s: %v", ba.BlobberID,
			err)
	}
	if err = sp.save(ssc.ID, ba.BlobberID, balances); err != nil {
		return fmt.Errorf("can't save stake pool of %s: %v", ba.BlobberID,
			err)
	}

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

	if t.Value < int64(conf.StakePool.MinLock) {
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

	var info *stakePoolUpdateInfo
	info, err = sp.update(conf, ssc.ID, t.CreationDate, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"updating stake pool: %v", err)
	}
	conf.Minted += info.minted

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"saving configurations: %v", err)
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

	return
}

// stake pool can return excess tokens from stake pool
func (ssc *StorageSmartContract) stakePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var (
		sp   *stakePool
		info *stakePoolUpdateInfo
		conf *scConfig
	)

	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't decode request: %v", err)
	}

	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get SC configurations: %v", err)
	}

	if sp, err = ssc.getStakePool(spr.BlobberID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related stake pool: %v", err)
	}

	info, err = sp.update(conf, ssc.ID, t.CreationDate, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"updating stake pool: %v", err)
	}
	conf.Minted += info.minted

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"saving configuration: %v", err)
	}

	var usp *userStakePools
	usp, err = ssc.getUserStakePool(t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related user stake pools: %v", err)
	}

	var unstake common.Timestamp
	resp, unstake, err = sp.empty(ssc.ID, spr.PoolID, t.ClientID, info,
		balances)
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

	return
}

// pay interests not payed for now
func (ssc *StorageSmartContract) stakePoolPayInterests(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		sp   *stakePool
		conf *scConfig
	)

	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("stake_pool_take_rewards_failed",
			"can't get SC configurations: "+err.Error())
	}

	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewError("stake_pool_take_rewards_failed",
			"can't get SC configurations: "+err.Error())
	}

	if sp, err = ssc.getStakePool(spr.BlobberID, balances); err != nil {
		return "", common.NewError("stake_pool_take_rewards_failed",
			"can't get related stake pool: "+err.Error())
	}

	var info *stakePoolUpdateInfo
	info, err = sp.update(conf, ssc.ID, t.CreationDate, balances)
	if err != nil {
		return "", common.NewError("stake_pool_take_rewards_failed",
			"updating stake pool: "+err.Error())
	}
	conf.Minted += info.minted

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewError("stake_pool_take_rewards_failed",
			"saving configurations: "+err.Error())
	}

	// save the pool
	if err = sp.save(ssc.ID, spr.BlobberID, balances); err != nil {
		return "", common.NewError("stake_pool_take_rewards_failed",
			"saving stake pool: "+err.Error())
	}

	return "interests has payed", nil
}

//
// stat
//

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
		return nil, fmt.Errorf("can't get SC configurations: %v", err)
	}

	if blobber, err = ssc.getBlobber(blobberID, balances); err != nil {
		return nil, fmt.Errorf("can't get blobber: %v", err)
	}

	if sp, err = ssc.getStakePool(blobberID, balances); err != nil {
		return nil, fmt.Errorf("can't get related stake pool: %v", err)
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
		now      = common.Now()
		conf     *scConfig
		usp      *userStakePools
	)

	if conf, err = ssc.getConfig(balances, false); err != nil {
		return nil, fmt.Errorf("can't get SC configurations: %v", err)
	}

	var (
		rate   = conf.StakePool.InterestRate
		period = toSeconds(conf.StakePool.InterestInterval)
	)

	usp, err = ssc.getUserStakePool(clientID, balances)
	if err != nil {
		return nil, fmt.Errorf("can't get user stake pools: %v", err)
	}

	var ups = new(userPoolStat)
	ups.Pools = make(map[datastore.Key][]*delegatePoolStat)

	for blobberID, poolIDs := range usp.Pools {

		var sp *stakePool
		if sp, err = ssc.getStakePool(blobberID, balances); err != nil {
			return nil, fmt.Errorf("can't get related stake pool: %v", err)
		}

		for _, id := range poolIDs {
			var dp, ok = sp.Pools[id]
			if !ok {
				return nil, errors.New("invalid state: missing delegate pool")
			}
			var dps = delegatePoolStat{
				ID:         dp.ID,
				Balance:    dp.Balance,
				DelegateID: dp.DelegateID,
				Interests:  dp.Interests,
				Rewards:    dp.Rewards,
				Penalty:    dp.Penalty,
			}
			if conf.canMint() {
				dps.PendingInterests = sp.interests(dp, now, rate, period)
			}
			ups.Pools[blobberID] = append(ups.Pools[blobberID], &dps)
		}
	}

	return ups, nil
}
