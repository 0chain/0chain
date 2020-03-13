package storagesc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

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

func (req *writePoolRequest) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(req); err != nil {
		panic(err) // must not happen
	}
	return
}

func (req *writePoolRequest) decode(input []byte) error {
	return json.Unmarshal(input, req)
}

// write pool is a locked tokens for a duration for an allocation

type writePool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
	ClientID                  string `json:"client_id"`
}

func newWritePool(clientID string) *writePool {
	return &writePool{
		ZcnLockingPool: &tokenpool.ZcnLockingPool{},
		ClientID:       clientID,
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
		Pool     json.RawMessage `json:"pool"`
		ClientID string          `json:"client_id"`
	}

	var writePoolVal writePoolJSON
	if err = json.Unmarshal(input, &writePoolVal); err != nil {
		return
	}

	if len(writePoolVal.Pool) == 0 {
		return // no data given
	}

	err = wp.ZcnLockingPool.Decode(writePoolVal.Pool, &tokenLock{})
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

// extend write pool expiration adding given time difference
func (wp *writePool) extend(add common.Timestamp) (err error) {
	if add == 0 {
		return // as is
	}
	tl, ok := wp.TokenLockInterface.(*tokenLock)
	if !ok {
		return fmt.Errorf(
			"invalid write pool state, invalid token lock type: %T",
			wp.TokenLockInterface)
	}
	tl.Duration += time.Duration(add) * time.Second // change
	return
}

// stat

type writePoolStat struct {
	ID        datastore.Key    `json:"pool_id"`
	StartTime common.Timestamp `json:"start_time"`
	Duartion  time.Duration    `json:"duration"`
	TimeLeft  time.Duration    `json:"time_left"`
	Locked    bool             `json:"locked"`
	Balance   state.Balance    `json:"balance"`
}

func (stat *writePoolStat) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stat); err != nil {
		panic(err) // must never happen
	}
	return
}

func (stat *writePoolStat) decode(input []byte) error {
	return json.Unmarshal(input, stat)
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
	wp = newWritePool("")
	err = wp.Decode(poolb)
	return
}

// newWritePool SC function creates new write pool for a client don't saving it
func (ssc *StorageSmartContract) newWritePool(allocationID, clientID string,
	creationDate, expiresAt common.Timestamp,
	balances chainState.StateContextI) (wp *writePool, err error) {

	_, err = ssc.getWritePoolBytes(allocationID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("new_write_pool_failed", err.Error())
	}

	if err == nil {
		return nil, common.NewError("new_write_pool_failed", "already exist")
	}

	err = nil // reset the util.ErrValueNotPresent

	wp = newWritePool(clientID)
	wp.TokenLockInterface = &tokenLock{
		StartTime: creationDate,
		Duration:  common.ToTime(expiresAt).Sub(common.ToTime(creationDate)),
		Owner:     clientID,
	}
	wp.TokenPool.ID = writePoolKey(ssc.ID, allocationID)
	return
}

// update Used field of blobbers and in all blobbers list;
// the allocs list have the same size and order as the list
func (ssc *StorageSmartContract) updateBlobbersUsed(allocs []*BlobberAllocation,
	balances chainState.StateContextI) (err error) {

	var all *StorageNodes
	if all, err = ssc.getBlobbersList(balances); err != nil {
		return fmt.Errorf("can't get all blobbers list: %v", err)
	}

	// update Used in the list
	for _, ab := range all.Nodes {
		for _, a := range allocs {
			if ab.ID == a.BlobberID {
				ab.Used += a.Size
				//
				break
			}
		}
	}

	// update Used in the list; save updates
	for i, b := range all.Nodes {
		b.Used += allocs[i].MinLockDemand
		_, err = balances.InsertTrieNode(b.GetKey(sc.ID), b)
		if err != nil {
			return errors.New("failed to save updated blobber")
		}
	}

	// update the Used in all blobbers list
	for _, ab := range all.Nodes {
		for _, b := range all {
			if ab.ID == b {
				ab.Used = b.Used
				break
			}
		}
	}
	// save all blobbers list
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, all)
	if err != nil {
		return errors.New("failed to save updated list of all blobber")
	}
	return // nil
}

func (ssc *StorageSmartContract) writePoolLockHelper(t *transaction.Transaction,
	conf *writePoolConfig, alloc *StorageAllocation,
) (err error) {

	if t.Value < conf.MinLock {
		return errors.New("insufficient amount to lock")
	}

	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	if err == util.ErrValueNotPresent {
		return errors.New("no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return errors.New("lock amount is greater than balance")
	}

	if _, resp, err = wp.fill(t, balances); err != nil {
		return
	}

	// conclude the allocation (got min lock demand)
	if !alloc.IsConcluded && wp.Balance >= alloc.MinLockDemand {
		err = ssc.updateBlobbersUsed(alloc.BlobberDetails, balances)
		if err != nil {
			return
		}
		req.IsConcluded = true
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

	// write lock request & user balance

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}
	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewError("write_pool_lock_failed", "no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return "", common.NewError("write_pool_lock_failed",
			"lock amount is greater than balance")
	}

	if req.AllocationID == "" {
		return "", common.NewError("write_pool_lock_failed",
			"missing allocation_id")
	}

	// get allocation

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get related allocation: "+err.Error())
	}

	// user write pools

	var wp *writePool
	if wp, err = ssc.getWritePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// lock more tokens

	if _, resp, err = wp.fill(t, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			err.Error())
	}

	if err = wp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) writePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// the request

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// write pool

	var wp *writePool
	if wp, err = ssc.getWritePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	if wp.ClientID != t.ClientID {
		return "", common.NewError("write_pool_unlock_failed",
			"only owner can unlock the pool")
	}

	var transfer *state.Transfer
	transfer, resp, err = wp.EmptyPool(ssc.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// save pool
	if err = wp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	return
}

//
// stat
//

func (ssc *StorageSmartContract) getWritePoolStat(wp *writePool, tp time.Time) (
	stat *writePoolStat, err error) {

	stat = new(writePoolStat)

	if err = stat.decode(wp.LockStats(tp)); err != nil {
		return nil, err
	}

	stat.ID = wp.ID
	stat.Locked = wp.IsLocked(tp)
	stat.Balance = wp.Balance

	return
}

// statistic for all locked tokens of a write pool
func (ssc *StorageSmartContract) getWritePoolStatHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = datastore.Key(params.Get("allocation_id"))
		wp           *writePool
	)
	if wp, err = ssc.getWritePool(allocationID, balances); err != nil {
		return
	}

	var (
		tp   = time.Now()
		stat *writePoolStat
	)
	if stat, err = ssc.getWritePoolStat(wp, tp); err != nil {
		return nil, common.NewError("write_pool_stats", err.Error())
	}

	return &stat, nil
}
