package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

// offerPool represents stake tokens of a blobber locked
// for an allocation, it required for cases where blobber
// changes terms or changes its capacity including reducing
// the capacity to zero; it implemented not as a token
// pool, but as set or values
type offerPool struct {
	Lock   state.Balance    `json:"lock"`   // offer stake
	Expire common.Timestamp `json:"expire"` // offer expiration
}

// stake pool of a blobber

type stakePool struct {
	Pools      map[string]*tokenpool.ZcnPool `json:"locked"`
	DelegateID datastore.Key                 `json:"delegate_id"`

	// mints (periodic mints, no integral mints)

	// MintedAt is last time the stake pool has minted (interests has payed).
	MintedAt common.Timestamp `json:"minted_at"`

	// offers (allocations)

	// Offers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	Offers map[string]*offerPool `json:"offers"`

	// rewards pool

	Rewards state.Balance `json:"rewards"` // pool

	// rewards statistic
	BlobberReward   state.Balance `json:"blobber_reward"`   // blobber
	ValidatorReward state.Balance `json:"validator_reward"` // validator
	InterestReward  state.Balance `json:"interest_reward"`  // mints
}

// newStakePool for given blobber, use empty blobberID to create a stakePool to
// decode, since the blobberID is stored
func newStakePool(delegateID datastore.Key) *stakePool {
	return &stakePool{
		Pools:      make(map[string](tokenpool.ZcnPool)),
		DelegateID: delegateID,
		Offers:     make(map[string]*offerPool),
	}
}

// stake pool key for the storage SC and  blobber
func stakePoolKey(scKey, blobberID string) datastore.Key {
	return datastore.Key(scKey + ":stakepool:" + blobberID)
}

func stakePoolID(scKey, blobberID string) datastore.Key {
	// return encryption.Hash(stakePoolKey(scKey, blobberID))
	_ = encryption.Hash
	return stakePoolKey(scKey, blobberID)
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
func (sp *stakePool) offersStake(now common.Timestamp) (os state.Balance) {
	for allocID, off := range sp.Offers {
		if off.Expire < now {
			delete(sp.Offers, allocID) //remove expired
			continue                   // an expired offer
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

// fill the pool by transaction
func (sp *stakePool) fill(t *transaction.Transaction,
	balances chainstate.StateContextI) (
	transfer *state.Transfer, resp string, err error) {

	if transfer, resp, err = sp.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
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
	stake   state.Balance // required stake
	offers  state.Balance // offers stake
	lack    state.Balance // lack tokens
	excess  state.Balance // excess tokens
	balance state.Balance // tokens for interests
}

// (statistic) all rewards of all time
func (sp *stakePool) rewards() state.Balance {
	return sp.BlobberReward + sp.ValidatorReward + sp.InterestReward
}

// balance for interests (stake pool balance, excluding current rewards)
func (sp *stakePool) balance() state.Balance {
	return sp.Balance - sp.Rewards
}

// pay interests for stake pool
func (sp *stakePool) minting(conf *scConfig, sscID string,
	now common.Timestamp, balances chainstate.StateContextI) (err error) {

	var (
		at      = sp.MintedAt                      // last periodic mint
		rate    = conf.InterestRate                // %
		period  = toSeconds(conf.InterestInterval) // interests period
		balance = sp.balance()

		mints state.Balance
	)

	if period == 0 {
		return // invalid period
	}

	for ; at+period < now; at += period {
		mints += state.Balance(rate * float64(balance))
	}

	// add mints
	err = balances.AddMint(&state.Mint{
		Minter:     sscID, // from the storage SC
		ToClientID: sscID, // to the stake pool (to the storage SC)
		Amount:     mints, // amount of mints
	})
	if err != nil {
		return fmt.Errorf("adding mints: %v", err)
	}

	sp.Balance += mints // add mints to pool balance

	// update the pool
	sp.MintedAt = at
	sp.Rewards += mints
	sp.InterestReward += mints
	return
}

// update information about the stake pool internals
func (sp *stakePool) update(conf *scConfig, sscID string, now common.Timestamp,
	blobber *StorageNode, balances chainstate.StateContextI) (
	info *stakePoolUpdateInfo, err error) {

	var (
		offersStake   = sp.offersStake(now) // locked by offers
		capacityStake = blobber.stake()     // capacity lock
		balance       = sp.balance()        // stake balance

		stake, lack, excess state.Balance
	)

	// if a blobber reduces its capacity after some offers,
	// the the offersStake can be greater the capacity stake;
	// the stake is stake required for now

	stake = maxBalance(offersStake, capacityStake)

	if balance < stake {
		lack = stake - balance
	} else if balance > stake {
		excess = balance - stake
	}

	// mints
	if err = sp.minting(conf, sscID, now, balances); err != nil {
		return
	}

	info = new(stakePoolUpdateInfo)
	info.stake = stake        // required stake
	info.offers = offersStake // offers stake
	info.lack = lack          // tokens lack
	info.excess = excess      // tokens excess
	info.balance = balance    // interests payments stake
	return
}

// update stake pool before moving
func (sp *stakePool) moveToWallet(sscID, walletID string, value state.Balance,
	balances chainstate.StateContextI) (resp string, err error) {

	var transfer *state.Transfer
	transfer, resp, err = sp.DrainPool(sscID, walletID, value, nil)
	if err != nil {
		return "", fmt.Errorf("draining stake pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer: %v", err)
	}

	return
}

// moveToWritePool moves tokens to write pool on challenge failed
func (sp *stakePool) moveToWritePool(allocID, blobID string,
	until common.Timestamp, wp *writePool, value state.Balance) (err error) {

	if value == 0 {
		return // nothing to move
	}

	if sp.Balance < value {
		return fmt.Errorf("not enough tokens in stake pool %s: %d < %d",
			sp.ID, sp.Balance, value)
	}

	var ap = wp.allocPool(allocID, until)
	if ap == nil {
		ap = new(allocationPool)
		ap.AllocationID = allocID
		ap.ExpireAt = 0
		wp.Pools.add(ap)
	}

	// move
	if blobID != "" {
		var bp, ok = ap.Blobbers.get(blobID)
		if !ok {
			ap.Blobbers.add(&blobberPool{
				BlobberID: blobID,
				Balance:   value,
			})
		} else {
			bp.Balance += value
		}
	}
	_, _, err = sp.TransferTo(ap, value, nil)
	return
}

// update the pool to get the stat
func (sp *stakePool) stat(conf *scConfig, scKey string, now common.Timestamp,
	blobber *StorageNode) (stat *stakePoolStat) {

	var balance = sp.balance()

	stat = new(stakePoolStat)
	stat.ID = stakePoolID(scKey, blobber.ID)

	stat.Locked = balance

	stat.CapacityStake = blobber.stake()

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

	// lack tokens
	var stake = maxBalance(stat.CapacityStake, stat.OffersTotal)

	if balance < stake {
		stat.Lack = stake - balance
	} else if balance > stake {
		stat.Excess = balance - stake
	}

	// virtual interests rewards
	var (
		at     = sp.MintedAt                      // last periodic mint
		rate   = conf.InterestRate                // %
		period = toSeconds(conf.InterestInterval) // interests period
		mints  state.Balance
	)
	for ; at+period < now; at += period {
		mints += state.Balance(rate * float64(balance))
	}
	stat.Rewards = sp.Rewards + mints
	stat.InterestReward = sp.InterestReward + mints
	//

	// other rewards
	stat.BlobberReward = sp.BlobberReward
	stat.ValidatorReward = sp.ValidatorReward
	return
}

// stat

type offerPoolStat struct {
	Lock         state.Balance    `json:"lock"`
	Expire       common.Timestamp `json:"expire"`
	AllocationID string           `json:"allocation_id"`
	IsExpired    bool             `json:"is_expired"`
}

type stakePoolStat struct {
	ID            datastore.Key   `json:"pool_id"`
	Locked        state.Balance   `json:"locked"`
	Offers        []offerPoolStat `json:"offers"`
	OffersTotal   state.Balance   `json:"offers_total"`
	CapacityStake state.Balance   `json:"capacity_stake"`
	Lack          state.Balance   `json:"lack"`
	Excess        state.Balance   `json:"excess"`
	// rewards
	Rewards         state.Balance `json:"rewards"`
	InterestReward  state.Balance `json:"interest_reward"`
	BlobberReward   state.Balance `json:"blobber_reward"`
	ValidatorReward state.Balance `json:"validator_reward"`
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

// get existing stake pool or create new one not saving it
func (ssc *StorageSmartContract) getOrCreateStakePool(blobberID datastore.Key,
	balances chainstate.StateContextI) (sp *stakePool, err error) {

	// the stake pool can be created by related validator
	sp, err = ssc.getStakePool(blobberID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	if err == util.ErrValueNotPresent {
		sp, err = newStakePool(), nil // create new, reset error
		sp.ZcnPool.ID = stakePoolID(ssc.ID, blobberID)
	}

	return
}

// newStakePool SC function creates new stake pool for a blobber don't saving it
func (ssc *StorageSmartContract) newStakePool(blobberID string,
	balances chainstate.StateContextI) (sp *stakePool, err error) {

	_, err = ssc.getStakePoolBytes(blobberID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("new_stake_pool_failed", err.Error())
	}

	if err == nil {
		return nil, common.NewError("new_stake_pool_failed", "already exist")
	}

	err = nil // reset the util.ErrValueNotPresent

	sp = newStakePool()
	sp.ZcnPool.ID = stakePoolID(ssc.ID, blobberID)
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

// Blobber owner can lock tokens lack without addBlobber transaction.
// All tokens over the required stake will be returned to the blobber.
// Note, a blobber can use addBlobber transaction to fill the stake pool.
func (ssc *StorageSmartContract) stakePoolLock(t *transaction.Transaction,
	_ []byte, balances chainstate.StateContextI) (resp string, err error) {

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get SC configurations: "+err.Error())
	}

	var blobber *StorageNode
	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get blobber: "+err.Error())
	}

	var sp *stakePool
	if sp, err = ssc.getStakePool(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get related stake pool: "+err.Error())
	}

	_, err = sp.update(conf, ssc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"updating stake pool: "+err.Error())
	}

	if err = checkFill(t, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed", err.Error())
	}

	if _, resp, err = sp.fill(t, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"stake pool filling error: "+err.Error())
	}

	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"saving stake pool: "+err.Error())
	}

	return
}

type unlockStakeRequest struct {
	Amount state.Balance `json:"amount"`
}

func (usr *unlockStakeRequest) decode(p []byte) (err error) {
	if err = json.Unmarshal(p, usr); err != nil {
		return
	}
	if usr.Amount <= 0 {
		return fmt.Errorf("invalid unlock request amount: %v <= 0", usr.Amount)
	}
	return // ok
}

// stake pool can return excess tokens from stake pool
func (ssc *StorageSmartContract) stakePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var (
		blobber *StorageNode
		sp      *stakePool
		info    *stakePoolUpdateInfo
		conf    *scConfig
	)

	var usr unlockStakeRequest
	if err = usr.decode(input); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't decode request: "+err.Error())
	}

	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get SC configurations: "+err.Error())
	}

	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get blobber: "+err.Error())
	}

	if sp, err = ssc.getStakePool(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get related stake pool: "+err.Error())
	}

	info, err = sp.update(conf, ssc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"updating stake pool: "+err.Error())
	}

	// how many tokens can unlock
	if info.excess <= usr.Amount {
		return "", common.NewError("stake_pool_unlock_failed",
			fmt.Sprintf("can't unlock %d tokens, can %d only",
				usr.Amount, info.excess))
	}

	// unlock
	resp, err = sp.moveToWallet(ssc.ID, t.ClientID, usr.Amount, balances)
	if err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"unlocking tokens: "+err.Error())
	}

	// save the pool
	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"saving stake pool: "+err.Error())
	}

	return
}

// unlock all rewards
func (ssc *StorageSmartContract) stakePoolUnlockRewards(
	t *transaction.Transaction, _ []byte, balances chainstate.StateContextI) (
	resp string, err error) {

	var (
		blobber *StorageNode
		sp      *stakePool
		conf    *scConfig
	)

	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get SC configurations: "+err.Error())
	}

	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get blobber: "+err.Error())
	}

	if sp, err = ssc.getStakePool(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get related stake pool: "+err.Error())
	}

	_, err = sp.update(conf, ssc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"updating stake pool: "+err.Error())
	}

	if sp.Rewards == 0 {
		return "", common.NewError("stake_pool_unlock_failed",
			"no rewards to unlock")
	}

	// unlock
	resp, err = sp.moveToWallet(ssc.ID, t.ClientID, sp.Rewards, balances)
	if err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"unlocking tokens: "+err.Error())
	}
	sp.Rewards = 0 // reset to zero

	// save the pool
	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"saving stake pool: "+err.Error())
	}

	return
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
