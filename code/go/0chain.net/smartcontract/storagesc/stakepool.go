package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
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
	*tokenpool.ZcnPool `json:"locked"`
	// Offers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	Offers map[string]*offerPool `json:"offers"`
	// reward represents total blobber reward.
	Reward state.Balance `json:"reward"`
}

// newStakePool for given blobber, use empty blobberID to create a stakePool to
// decode, since the blobberID is stored
func newStakePool() *stakePool {
	return &stakePool{
		ZcnPool: &tokenpool.ZcnPool{},
		Offers:  make(map[string]*offerPool),
	}
}

// stake pool key for the storage SC and  blobber
func stakePoolKey(scKey, blobberID string) datastore.Key {
	return datastore.Key(scKey + ":stakepool:" + blobberID)
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
	balances chainState.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(stakePoolKey(sscKey, blobberID), sp)
	return
}

// fill the pool by transaction
func (sp *stakePool) fill(t *transaction.Transaction,
	balances chainState.StateContextI) (
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
		Expire: alloc.Expiration + toSeconds(alloc.ChallengeCompletionTime),
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
	op.Expire = alloc.Expiration + toSeconds(alloc.ChallengeCompletionTime)
	return
}

func maxBalance(a, b state.Balance) state.Balance {
	if a > b {
		return a
	}
	return b
}

type stakePoolUpdateInfo struct {
	stake state.Balance // required stake
	lack  state.Balance // lack tokens
	resp  string        // return overfill transfer response
}

// update moves all tokens over required back to blobber, removes expired
// offers, calculates required stake size and tokens lack
func (sp *stakePool) update(sscID string, now common.Timestamp,
	blobber *StorageNode, balances chainState.StateContextI) (
	info *stakePoolUpdateInfo, err error) {

	var (
		offersStake   = sp.offersStake(now) // locked by offers
		capacityStake = blobber.stake()     // capacity lock

		stake, lack state.Balance
		resp        string
	)

	// if a blobber reduces its capacity after some offers,
	// the the offersStake can be greater the capacity stake;
	// the stake is stake required for now

	stake = maxBalance(offersStake, capacityStake)

	// expired offers remove, and we know required stake

	switch {
	case stake == sp.Balance:
		// required stake is equal to number of locked tokens, nothing to move
	case sp.Balance > stake:
		// move tokens over the stake to blobber
		var (
			move     = sp.Balance - stake
			transfer *state.Transfer
		)
		transfer, resp, err = sp.DrainPool(sscID, blobber.ID, move, nil)
		if err != nil {
			return
		}
		if err = balances.AddTransfer(transfer); err != nil {
			return
		}
	case sp.Balance < stake:
		lack = stake - sp.Balance
	}

	info = new(stakePoolUpdateInfo)
	info.stake = stake
	info.lack = lack
	info.resp = resp
	return
}

// moveToWritePool moves tokens to write pool on challenge failed
func (sp *stakePool) moveToWritePool(wp *writePool,
	value state.Balance) (err error) {

	if value == 0 {
		return // nothing to move
	}

	if sp.Balance < value {
		return fmt.Errorf("not enough tokens in stake pool %s: %d < %d",
			sp.ID, sp.Balance, value)
	}

	// move
	_, _, err = sp.TransferTo(&wp.Back, value, nil)
	return
}

// update the pool to get the stat
func (sp *stakePool) stat(scKey string, now common.Timestamp,
	blobber *StorageNode) (stat *stakePoolStat) {

	stat = new(stakePoolStat)
	stat.ID = stakePoolKey(scKey, blobber.ID)

	stat.Locked = sp.Balance

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

	if sp.Balance < stake {
		stat.Lack = stake - sp.Balance
	} else if sp.Balance > stake {
		stat.Overfill = sp.Balance - stake
	}

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
	Overfill      state.Balance   `json:"overfill"`
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
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(stakePoolKey(ssc.ID, blobberID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getStakePool of given blobber
func (ssc *StorageSmartContract) getStakePool(blobberID datastore.Key,
	balances chainState.StateContextI) (sp *stakePool, err error) {

	var poolb []byte
	if poolb, err = ssc.getStakePoolBytes(blobberID, balances); err != nil {
		return
	}
	sp = newStakePool()
	err = sp.Decode(poolb)
	return
}

// newStakePool SC function creates new stake pool for a blobber don't saving it
func (ssc *StorageSmartContract) newStakePool(blobberID string,
	balances chainState.StateContextI) (sp *stakePool, err error) {

	_, err = ssc.getStakePoolBytes(blobberID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("new_stake_pool_failed", err.Error())
	}

	if err == nil {
		return nil, common.NewError("new_stake_pool_failed", "already exist")
	}

	err = nil // reset the util.ErrValueNotPresent

	sp = newStakePool()
	sp.ZcnPool.ID = stakePoolKey(ssc.ID, blobberID)
	return
}

func (ssc *StorageSmartContract) updateSakePoolOffer(ba *BlobberAllocation,
	alloc *StorageAllocation, balances chainState.StateContextI) (err error) {

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
	_ []byte, balances chainState.StateContextI) (resp string, err error) {

	var (
		blobber *StorageNode
		sp      *stakePool
		info    *stakePoolUpdateInfo
	)

	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get blobber: "+err.Error())
	}

	if sp, err = ssc.getStakePool(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get related stake pool: "+err.Error())
	}

	info, err = sp.update(ssc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"updating stake pool: "+err.Error())
	}

	if info.lack == 0 {
		return "", common.NewError("stake_pool_lock_failed",
			"no tokens lack in the stake pool")
	}

	if err = checkFill(t, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed", err.Error())
	}

	if _, resp, err = sp.fill(t, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"stake pool filling error: "+err.Error())
	}

	// return overfill back to client
	_, err = sp.update(ssc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"updating stake pool: "+err.Error())
	}

	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"saving stake pool: "+err.Error())
	}

	return
}

// stakePoolUnlock can return overfill of the stake pool to related blobber
// any time. Note: a blobber can use addBlobber transaction to returns an
// overfill.
func (ssc *StorageSmartContract) stakePoolUnlock(t *transaction.Transaction,
	_ []byte, balances chainState.StateContextI) (resp string, err error) {

	var (
		blobber *StorageNode
		sp      *stakePool
		info    *stakePoolUpdateInfo
	)

	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get blobber: "+err.Error())
	}

	if sp, err = ssc.getStakePool(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"can't get related stake pool: "+err.Error())
	}

	// the update transfers an overfill back to blobber
	info, err = sp.update(ssc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"updating stake pool: "+err.Error())
	}

	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_lock_failed",
			"saving stake pool: "+err.Error())
	}

	return info.resp, nil
}

//
// stat
//

// statistic for all locked tokens of a stake pool
func (ssc *StorageSmartContract) getStakePoolStatHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		blobberID = datastore.Key(params.Get("blobber_id"))
		blobber   *StorageNode
		sp        *stakePool
	)

	if blobber, err = ssc.getBlobber(blobberID, balances); err != nil {
		return nil, fmt.Errorf("can't get blobber: %v", err)
	}

	if sp, err = ssc.getStakePool(blobberID, balances); err != nil {
		return nil, fmt.Errorf("can't get related stake pool: %v", err)
	}

	return sp.stat(ssc.ID, common.Now(), blobber), nil
}
