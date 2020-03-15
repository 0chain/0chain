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
	Lock         state.Balance    `json:"lock"`          // offer stake
	Expire       common.Timestamp `json:"expire"`        // offer expiration
	AllocationID string           `json:"allocation_id"` // offer reference
}

// stake pool of a blobber

type stakePool struct {
	// Unlocked tokens are tokens released. When a blobber reduces
	// its capacity some of tokens moved to the unlocked and can be
	// released by the blobber (with regards to open offers).
	Unlocked *tokenpool.ZcnPool `json:"unlocked"`
	// Locker tokens is blobber's capacity stake.
	Locked *tokenpool.ZcnPool `json:"locked"`
	// Offers represents tokens required by currently
	// open offers of the blobber.
	Offers []*offerPool `json:"offers"`
	// BlobberID is pool owner.
	BlobberID string `json:"blobber_id"`
}

// newStakePool for given blobber, use empty blobberID to
// create a stakePool to decode, since the blobberID is
// stored
func newStakePool(blobberID string) *stakePool {
	return &stakePool{
		Unlocked:  &tokenpool.ZcnPool{},
		Locked:    &tokenpool.ZcnPool{},
		BlobberID: blobberID,
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
	var i int
	for _, off := range sp.Offers {
		if off.Expire < now {
			continue // an expired offer
		}
		os += off.Lock
		sp.Offers[i] = off
		i++
	}
	sp.Offers = sp.Offers[:i] // remove expired offers from the list
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

	if transfer, resp, err = sp.Locked.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

// add offer of an allocation related to blobber owns this stake pool
func (sp *stakePool) addOffer(alloc *StorageAllocation,
	balloc *BlobberAllocation) {

	sp.Offers = append(sp.Offers, &offerPool{
		Lock: state.Balance(balloc.Size) * balloc.Terms.WritePrice,
		Expire: alloc.Expiration +
			toSeconds(balloc.Terms.ChallengeCompletionTime),
		AllocationID: alloc.ID,
	})
}

// findOffer by allocation id or nil
func (sp *stakePool) findOffer(allocID string) *offerPool {
	for _, o := range sp.Offers {
		if o.AllocationID == allocID {
			return o
		}
	}
	return nil
}

// extendOffer changes offer lock and expiration on update allocations
func (sp *stakePool) extendOffer(alloc *StorageAllocation,
	balloc *BlobberAllocation) (err error) {

	var (
		op      = sp.findOffer(alloc.ID)
		newLock = state.Balance(balloc.Size) * balloc.Terms.WritePrice
	)
	if op == nil {
		return errors.New("missing offer pool for " + alloc.ID)
	}
	// unlike a write pool, here we can reduce a lock
	op.Lock = newLock
	op.Expire = alloc.Expiration +
		toSeconds(balloc.Terms.ChallengeCompletionTime)
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

// update moves tokens between unlocked and locked pools regarding
// offers and their expiration; all tokens left
func (sp *stakePool) update(now common.Timestamp, blobber *StorageNode,
	balances chainState.StateContextI) (stake state.Balance, err error) {

	var (
		offersStake   = sp.offersStake(now) // locked by offers
		capacityStake = blobber.stake()     // capacity lock
	)

	// if a blobber reduces its capacity after some offers,
	// the the offersStake can be greater the capacity stake;
	// the stake is stake required for now

	stake = maxBalance(offersStake, capacityStake)

	// expired offers remove, and we know required stake

	switch {
	case stake == sp.Locked.Balance:
		// required stake is equal to number of locked tokens, nothing to move

	case stake > sp.Locked.Balance:
		// move some tokens to unlocked

		var transfer *state.Transfer
		transfer, _, err = sp.Locked.TransferTo(sp.Unlocked,
			stake-sp.Locked.Balance, nil)
		if err != nil {
			return // an error
		}

		err = balances.AddTransfer(transfer)

	case stake < sp.Locked.Balance && sp.Unlocked.Balance > 0:
		// move some tokens to locked

		var (
			lack = stake - sp.Locked.Balance
			move = minBalance(lack, sp.Unlocked.Balance)
		)

		var transfer *state.Transfer
		transfer, _, err = sp.Unlocked.TransferTo(sp.Locked, move, nil)
		if err != nil {
			return // an error
		}

		err = balances.AddTransfer(transfer)

	default:
		// doesn't need to move something, or nothing to move
	}

	return
}

// update without a real transfers to get correct stat for now
func (sp *stakePool) dryUpdate(now common.Timestamp, blobber *StorageNode) {

	var (
		offersStake   = sp.offersStake(now) // locked by offers
		capacityStake = blobber.stake()     // capacity lock
		stake         state.Balance         // required stake
	)

	// if a blobber reduces its capacity after some offers,
	// the the offersStake can be greater the capacity stake;
	// the stake is stake required for now

	stake = maxBalance(offersStake, capacityStake)

	// expired offers remove, and we know required stake

	// dry transferring
	switch {
	case stake == sp.Locked.Balance:
		// nothing to transfer
	case stake > sp.Locked.Balance:
		// dry unlock some tokens
		var move = stake - sp.Locked.Balance
		sp.Unlocked.Balance += move
		sp.Locked.Balance -= move
	case stake < sp.Locked.Balance && sp.Unlocked.Balance > 0:
		// dry lock some tokens
		var (
			lack = stake - sp.Locked.Balance
			move = minBalance(lack, sp.Unlocked.Balance)
		)
		sp.Unlocked.Balance -= move
		sp.Locked.Balance += move
	}

	return
}

// update the pool to get the stat
func (sp *stakePool) stat(scKey string) (stat *stakePoolStat) {
	stat = new(stakePoolStat)
	stat.ID = stakePoolKey(scKey, sp.BlobberID)
	stat.Locked = sp.Locked.Balance
	stat.Unlocked = sp.Unlocked.Balance
	stat.Offers = make([]offerPoolStat, 0, len(sp.Offers))
	for _, off := range sp.Offers {
		stat.Offers = append(stat.Offers, offerPoolStat{
			Lock:         off.Lock,
			Expire:       off.Expire,
			AllocationID: off.AllocationID,
		})
		stat.OffersTotal += off.Lock
	}
	return
}

// stat

type offerPoolStat struct {
	Lock         state.Balance    `json:"lock"`
	Expire       common.Timestamp `json:"expire"`
	AllocationID string           `json:"allocation_id"`
}

type stakePoolStat struct {
	ID          datastore.Key   `json:"pool_id"`
	Locked      state.Balance   `json:"locked"`
	Unlocked    state.Balance   `json:"unlocked"`
	Offers      []offerPoolStat `json:"offers"`
	OffersTotal state.Balance   `json:"offers_total"`
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
	sp = newStakePool("")
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

	sp = newStakePool(blobberID)
	sp.Locked.ID = stakePoolKey(ssc.ID, blobberID) + ":locked"
	sp.Unlocked.ID = stakePoolKey(ssc.ID, blobberID) + ":unlocked"
	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) stakePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	var blobber *StorageNode
	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get releated blobber: "+err.Error())
	}

	var sp *stakePool
	if sp, err = ssc.getStakePool(t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't get related stake pool: "+err.Error())
	}

	if _, err = sp.update(t.CreationDate, blobber, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't update stake pool: "+err.Error())
	}

	if sp.Unlocked.Balance == 0 {
		return "", common.NewError("stake_pool_unlock_failed",
			"no tokens to unlock")
	}

	var transfer *state.Transfer
	transfer, resp, err = sp.Unlocked.EmptyPool(ssc.ID, t.ClientID, nil)
	if err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't unlock tokens: "+err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"transferring unlocked tokens: "+err.Error())
	}

	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("stake_pool_unlock_failed",
			"can't save stake pool: "+err.Error())
	}

	return // resp, nil
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

	// update the stake pool just to get actual statistic, without a
	// real locking and unlocking inside (dry updating)
	sp.dryUpdate(common.Now(), blobber)

	return sp.stat(ssc.ID), nil
}
