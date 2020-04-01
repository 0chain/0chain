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

// write pool lock/unlock request

type writePoolRequest struct {
	AllocationID string `json:"allocation_id"`
}

func (req *writePoolRequest) decode(input []byte) error {
	return json.Unmarshal(input, req)
}

// write pool is a locked tokens for a duration for an allocation

type writePool struct {
	*tokenpool.ZcnPool `json:"pool"`
}

func newWritePool() *writePool {
	return &writePool{
		ZcnPool: &tokenpool.ZcnPool{},
	}
}

func writePoolKey(scKey, allocationID string) datastore.Key {
	return datastore.Key(scKey + ":writepool:" + allocationID)
}

func (wp *writePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(wp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (wp *writePool) Decode(input []byte) (err error) {

	type writePoolJSON struct {
		Pool json.RawMessage `json:"pool"`
	}

	var writePoolVal writePoolJSON
	if err = json.Unmarshal(input, &writePoolVal); err != nil {
		return
	}

	if len(writePoolVal.Pool) == 0 {
		return // no data given
	}

	err = wp.ZcnPool.Decode(writePoolVal.Pool)
	return
}

// save the write pool
func (wp *writePool) save(sscKey, allocationID string,
	balances chainState.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(writePoolKey(sscKey, allocationID), wp)
	return
}

// fill the pool by transaction
func (wp *writePool) fill(t *transaction.Transaction,
	balances chainState.StateContextI) (
	transfer *state.Transfer, resp string, err error) {

	if transfer, resp, err = wp.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

// moveToChallenge moves given amount of token to challenge pool
func (wp *writePool) moveToChallenge(cp *challengePool, value state.Balance) (
	err error) {

	if wp.Balance < value {
		return errors.New("not enough tokens in write pool")
	}

	_, _, err = wp.TransferTo(cp, value, nil)
	return
}

// moveToStake moves given amount of token to stake pool (unlocked)
func (wp *writePool) moveToStake(sp *stakePool, value state.Balance) (
	err error) {

	if value == 0 {
		return // nothing to move
	}

	if wp.Balance < value {
		return errors.New("not enough tokens in write pool")
	}

	_, _, err = wp.TransferTo(sp, value, nil)
	return
}

// stat

type writePoolStat struct {
	ID         datastore.Key    `json:"pool_id"`
	Balance    state.Balance    `json:"balance"`
	StartTime  common.Timestamp `json:"start_time"`
	Expiration common.Timestamp `json:"expiration"`
	Finalized  bool             `json:"finalized"`
}

func (wp *writePool) stat(alloc *StorageAllocation) (stat *writePoolStat) {

	stat = new(writePoolStat)

	stat.ID = wp.ID
	stat.Balance = wp.Balance
	stat.StartTime = alloc.StartTime
	stat.Expiration = alloc.Expiration +
		toSeconds(alloc.ChallengeCompletionTime)
	stat.Finalized = alloc.Finalized

	return
}

//
// smart contract methods
//

// getWritePoolBytes of a client
func (ssc *StorageSmartContract) getWritePoolBytes(allocationID datastore.Key,
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(writePoolKey(ssc.ID, allocationID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getWritePool of current client
func (ssc *StorageSmartContract) getWritePool(allocationID datastore.Key,
	balances chainState.StateContextI) (wp *writePool, err error) {

	var poolb []byte
	if poolb, err = ssc.getWritePoolBytes(allocationID, balances); err != nil {
		return
	}
	wp = newWritePool()
	err = wp.Decode(poolb)
	return
}

// newWritePool SC function creates new write pool for a client don't saving it
func (ssc *StorageSmartContract) newWritePool(allocationID string,
	balances chainState.StateContextI) (wp *writePool, err error) {

	_, err = ssc.getWritePoolBytes(allocationID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("new_write_pool_failed", err.Error())
	}

	if err == nil {
		return nil, common.NewError("new_write_pool_failed", "already exist")
	}

	err = nil // reset the util.ErrValueNotPresent

	wp = newWritePool()
	wp.TokenPool.ID = writePoolKey(ssc.ID, allocationID)
	return
}

// create, fill and save write pool for new allocation
func (ssc *StorageSmartContract) createWritePool(t *transaction.Transaction,
	alloc *StorageAllocation, balances chainState.StateContextI) (err error) {

	// create related write_pool expires with the allocation + challenge
	// completion time
	var wp *writePool
	wp, err = ssc.newWritePool(alloc.ID, balances)
	if err != nil {
		return fmt.Errorf("can't create write pool: %v", err)
	}

	// lock required number of tokens

	var minLockDemand = alloc.minLockDemandLeft()

	if state.Balance(t.Value) < minLockDemand {
		return fmt.Errorf("not enough tokens to create allocation: %v < %v",
			t.Value, minLockDemand)
	}

	if err = ssc.checkFill(t, balances); err != nil {
		return fmt.Errorf("can't fill write pool: %v", err)
	}

	if _, _, err = wp.fill(t, balances); err != nil {
		return fmt.Errorf("can't fill write pool: %v", err)
	}

	// save the write pool
	if err = wp.save(ssc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save write pool: %v", err)
	}

	return
}

// lock tokens for write pool of transaction's client
func (ssc *StorageSmartContract) writePoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// configurations

	var conf *writePoolConfig
	if conf, err = ssc.getWritePoolConfig(balances, true); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// filter by configurations

	if t.Value < conf.MinLock {
		return "", common.NewError("write_pool_lock_failed",
			"insufficient amount to lock")
	}

	// write lock request

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// check the request

	if req.AllocationID == "" {
		return "", common.NewError("write_pool_lock_failed",
			"missing allocation_id")
	}

	// allocation

	var alloc *StorageAllocation
	if alloc, err = ssc.getAllocation(req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get related allocation: "+err.Error())
	}

	if alloc.Finalized {
		return "", common.NewError("write_pool_lock_failed",
			"allocation is finalized")
	}

	if alloc.Owner != t.ClientID {
		return "", common.NewError("write_pool_lock_failed",
			"only owner can fill the write pool")
	}

	// user write pools

	var wp *writePool
	if wp, err = ssc.getWritePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// lock more tokens

	if err = ssc.checkFill(t, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if _, resp, err = wp.fill(t, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			err.Error())
	}

	// save

	if err = wp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	return
}

//
// stat
//

// statistic for all locked tokens of a write pool
func (ssc *StorageSmartContract) getWritePoolStatHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = datastore.Key(params.Get("allocation_id"))
		alloc        *StorageAllocation
		wp           *writePool
	)

	if allocationID == "" {
		return nil, errors.New("missing allocation_id URL query parameter")
	}

	if alloc, err = ssc.getAllocation(allocationID, balances); err != nil {
		return
	}

	if wp, err = ssc.getWritePool(allocationID, balances); err != nil {
		return
	}

	return wp.stat(alloc), nil
}

//
// finalize an allocation (after expire + challenge completion time)
//

// 1. challenge pool                  -> write pool
// 2. write pool min_lock_demand left -> blobbers
// 3. remove offer from blobber (stake pool)
// 4. update blobbers used and in all blobbers list too
// 5. write pool                      -> client
func (ssc *StorageSmartContract) finalizeAllocation(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// the request

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	// write pool

	var wp *writePool
	if wp, err = ssc.getWritePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	// allocation
	var alloc *StorageAllocation
	if alloc, err = ssc.getAllocation(req.AllocationID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get related allocation: "+err.Error())
	}

	if alloc.Finalized {
		return "", common.NewError("fini_alloc_failed",
			"allocation already finalized")
	}

	// should be expired
	var expire = alloc.Expiration + toSeconds(alloc.ChallengeCompletionTime)
	if expire > t.CreationDate {
		return "", common.NewError("fini_alloc_failed",
			"allocation is not expired yet, or waiting a challenge completion")
	}

	// transaction initiator can be the client or a blobber of the transaction
	var validInitiator = (t.ClientID == alloc.Owner)
	if !validInitiator {
		for _, d := range alloc.BlobberDetails {
			if d.BlobberID == t.ClientID {
				validInitiator = true
				break
			}
		}
	}
	if !validInitiator {
		return "", common.NewError("fini_alloc_failed", "only allocation"+
			" owner or related blobber can finalize an allocation")
	}

	// challenge pool
	var cp *challengePool
	if cp, err = ssc.getChallengePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get related challenge pool")
	}

	// 1. empty the challenge pool
	if err = cp.moveToWritePool(wp, cp.Balance); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"emptying challenge pool: "+err.Error())
	}

	// save the challenge pool
	if err = cp.save(ssc.ID, alloc.ID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving challenge pool: "+err.Error())
	}

	// 4. update blobbers' used and blobbers in all
	var blobbers []*StorageNode
	if blobbers, err = ssc.getAllocationBlobbers(alloc, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"invalid state: can't get related blobbers: "+err.Error())
	}

	// 2. move min_lock_demands left to blobber's stake pools
	// 3. remove blobber's offers (update)
	for i, d := range alloc.BlobberDetails {
		var sp *stakePool
		if sp, err = ssc.getStakePool(d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		if d.MinLockDemand > d.Spent {
			if err = wp.moveToStake(sp, d.MinLockDemand-d.Spent); err != nil {
				return "", common.NewError("fini_alloc_failed", "can't send"+
					" min lock demand left for "+d.BlobberID+": "+err.Error())
			}
		}
		// get the blobber
		var b *StorageNode
		if b, err = ssc.getBlobber(d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get blobber "+d.BlobberID+": "+err.Error())
		}
		if _, err = sp.update(ssc.ID, t.CreationDate, b, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't update stake pool of "+d.BlobberID+": "+err.Error())
		}
		// save the stake pool
		if err = sp.save(ssc.ID, d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't save stake pool of "+d.BlobberID+": "+err.Error())
		}
		d.Spent = d.MinLockDemand // to save

		// the size has released (deallocation)
		var blob = blobbers[i]
		blob.Used -= d.Size
		// save the independent blobber instance
		_, err = balances.InsertTrieNode(blob.GetKey(ssc.ID), blob)
		if err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't save blobber "+blob.ID+": "+err.Error())
		}
	}

	var allBlobbers *StorageNodes
	if allBlobbers, err = ssc.getBlobbersList(balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get all blobbers list: "+err.Error())
	}

	if err = updateBlobbersInAll(allBlobbers, blobbers, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't save all blobbers list: "+err.Error())
	}

	// 5. move all of the write pool left to allocation owner
	var transfer *state.Transfer
	transfer, resp, err = wp.EmptyPool(ssc.ID, alloc.Owner,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	// save pool
	if err = wp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving write pool: "+err.Error())
	}

	// save the allocation
	alloc.Finalized = true
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving allocation: "+err.Error())
	}

	// remove the allocation from the all allocations list
	var all *Allocations
	if all, err = ssc.getAllAllocationsList(balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"getting all allocations list: "+err.Error())
	}
	if !all.List.remove(alloc.ID) {
		return "", common.NewError("fini_alloc_failed",
			"invalid state: allocation not found in all allocations list")
	}
	_, err = balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, all)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving all allocations list: "+err.Error())
	}

	return
}
