package storagesc

import (
	"context"
	"encoding/json"
	"errors"
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

// lock request

type lockRequest struct {
	Duration     time.Duration `json:"duration"`      // lock duration
	AllocationID string        `json:"allocation_id"` // for allocation
	Blobbers     []string      `json:"blobbers"`      // for blobbers

	// 1. value and start time provided in transaction
	// 2. value divided by the number of blobbers with read price weight
}

// validate the lock request
func (lr *lockRequest) validate(conf *readPoolConfig,
	alloc *StorageAllocation) (err error) {

	if lr.Duration < conf.MinLockPeriod {
		return errors.New("insufficient lock period")
	}

	if lr.Duration > conf.MaxLockPeriod {
		return errors.New("lock duration is too big")
	}

	if lr.AllocationID == "" {
		return errors.New("missing allocation id")
	}

	if len(lr.Blobbers) == 0 {
		return errors.New("missing allocation blobbers")
	}

	for _, id := range lr.Blobbers {
		if _, ok := alloc.BlobberMap[id]; !ok {
			return fmt.Errorf("blobber %q doesn't belong to allocation %q",
				id, lr.AllocationID)
		}
	}

	return
}

func (lr *lockRequest) decode(input []byte) error {
	return json.Unmarshal(input, lr)
}

// read pool -> [ allocation -> [ blobber -> [ lock, ... ], ... ] , ... ]

// locked tokens for an internal
type readPoolLock struct {
	StartTime common.Timestamp `json:"start_time"` // lock start
	Duration  time.Duration    `json:"duration"`   // lock duration
	Value     state.Balance    `json:"value"`      // locked tokens
}

func (rpl *readPoolLock) copy() (cp *readPoolLock) {
	cp = new(readPoolLock)
	(*cp) = (*rpl)
	return
}

// read pool blobbers
type readPoolBlobbers map[string][]*readPoolLock

func (rpb readPoolBlobbers) copy() (cp readPoolBlobbers) {
	cp = make(readPoolBlobbers, len(rpb))
	for k, v := range rpb {
		var locks = make([]*readPoolLock, len(v))
		for i, l := range v {
			locks[i] = l.copy()
		}
		cp[k] = locks
	}
	return
}

// read pool allocations
type readPoolAllocs map[string]readPoolBlobbers

func (rpa readPoolAllocs) update(now common.Timestamp) (
	ul state.Balance) {

	for allocID, alloc := range rpa {
		for blobID, blob := range alloc {
			var i int
			for _, b := range blob {
				if now > b.StartTime+toSeconds(b.Duration) {
					ul += b.Value // unlock, expired
					continue
				}
				blob[i] = b
				i++
			}
			blob = blob[:i] // remove expired locks
			if len(blob) == 0 {
				delete(alloc, blobID) // remove empty locks list
			}
		}
		if len(alloc) == 0 {
			delete(rpa, allocID) // remove empty allocation
		}
	}

	return
}

func (rpa readPoolAllocs) copy() (cp readPoolAllocs) {
	cp = make(readPoolAllocs, len(rpa))
	for k, v := range rpa {
		cp[k] = v.copy()
	}
	return

}

// readPool consist of two pools: all locked tokens and all unlocked,
// and records with locks "allocation:blobber" -> {value, duration}
type readPool struct {
	Locked   *tokenpool.ZcnPool `json:"locked"`
	Unlocked *tokenpool.ZcnPool `json:"unlocked"`
	Locks    readPoolAllocs     `json:"locks"`
}

// new read pool instance
func newReadPool() *readPool {
	return &readPool{
		Locked:   &tokenpool.ZcnPool{}, // locked tokens
		Unlocked: &tokenpool.ZcnPool{}, // unlocked tokens
		Locks:    make(readPoolAllocs), // locks
	}
}

func (rp *readPool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(rp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (rp *readPool) Decode(input []byte) (err error) {

	type readPoolJSON struct {
		Locked   json.RawMessage `json:"locked"`
		Unlocked json.RawMessage `json:"unlocked"`
		Locks    readPoolAllocs  `json:"locks"`
	}

	var readPoolVal readPoolJSON
	if err = json.Unmarshal(input, &readPoolVal); err != nil {
		return
	}

	if len(readPoolVal.Locked) > 0 {
		if err = rp.Locked.Decode(readPoolVal.Locked); err != nil {
			return
		}
	}

	if len(readPoolVal.Unlocked) > 0 {
		if err = rp.Unlocked.Decode(readPoolVal.Unlocked); err != nil {
			return
		}
	}

	rp.Locks = readPoolVal.Locks
	return
}

func readPoolKey(scKey, clientID string) datastore.Key {
	return datastore.Key(scKey + ":readpool:" + clientID)
}

// fill the Lock pool by transaction
func (rp *readPool) fill(t *transaction.Transaction,
	balances chainState.StateContextI) (
	transfer *state.Transfer, resp string, err error) {

	if transfer, resp, err = rp.Locked.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

// addLocks for allocation (client balance already checked,
// tokens already moved to Lock pool)
func (rp *readPool) addLocks(now common.Timestamp, value state.Balance,
	req *lockRequest, alloc *StorageAllocation) (re state.Balance) {

	if rp.Locks == nil {
		rp.Locks = make(readPoolAllocs, 1)
	}

	var blobbers, ok = rp.Locks[req.AllocationID]
	if !ok {
		blobbers = make(readPoolBlobbers, len(req.Blobbers))
		rp.Locks[req.AllocationID] = blobbers
	}

	// divide by read price (weighted division)
	var (
		prices = make(map[string]float64, len(alloc.BlobberDetails))
		rps    float64
	)
	// build map blobber_id -> read price
	for _, details := range alloc.BlobberDetails {
		prices[details.BlobberID] = float64(details.Terms.ReadPrice)
	}
	// calculate read_prices sum for all blobbers in request
	for _, b := range req.Blobbers {
		rps += prices[b]
	}
	// rounding (truncating) error
	var locked state.Balance
	// add locks for the blobbers
	for _, b := range req.Blobbers {
		var lock readPoolLock
		lock.StartTime = now
		lock.Duration = req.Duration
		lock.Value = state.Balance(float64(value) * (prices[b] / rps))
		locked += lock.Value
		blobbers[b] = append(blobbers[b], &lock) // add the lock
	}

	re = value - locked
	return
}

// move all expired tokens from Lock pool to Unlock pool; returns number
// of unlocked tokens
func (rp *readPool) update(now common.Timestamp) (ul state.Balance, err error) {

	if ul = rp.Locks.update(now); ul == 0 {
		return // nothing to move
	}

	// move tokens
	_, _, err = rp.Locked.TransferTo(rp.Unlocked, ul, nil)
	return
}

// save the read pool
func (rp *readPool) save(sscKey, clientID string,
	balances chainState.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(readPoolKey(sscKey, clientID), rp)
	return
}

// stat
func (rp *readPool) stat() (stat *readPoolStat) {
	stat = new(readPoolStat)
	stat.Locked = rp.Locked.Balance
	stat.Unlocked = rp.Unlocked.Balance
	// dry update, dry unlock
	stat.Locks = rp.Locks.copy()             // use copy
	var ul = stat.Locks.update(common.Now()) // update
	stat.Locked -= ul
	stat.Unlocked += ul
	return
}

// read pool statistic
type readPoolStat struct {
	Locked   state.Balance  `json:"locked"`
	Unlocked state.Balance  `json:"unlocked"`
	Locks    readPoolAllocs `json:"locks"`
}

//
// smart contract methods
//

// getReadPoolBytes of a client
func (ssc *StorageSmartContract) getReadPoolBytes(clientID datastore.Key,
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(readPoolKey(ssc.ID, clientID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getReadPool of current client
func (ssc *StorageSmartContract) getReadPool(clientID datastore.Key,
	balances chainState.StateContextI) (rp *readPool, err error) {

	var poolb []byte
	if poolb, err = ssc.getReadPoolBytes(clientID, balances); err != nil {
		return
	}
	rp = newReadPool()
	err = rp.Decode(poolb)
	return
}

// newReadPool SC function creates new read pool for a client;
// it saves the read pool
func (ssc *StorageSmartContract) newReadPool(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	_, err = ssc.getReadPoolBytes(t.ClientID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	if err == nil {
		return "", common.NewError("new_read_pool_failed", "already exist")
	}

	var rp = newReadPool()
	rp.Locked.ID = readPoolKey(ssc.ID, t.ClientID) + ":locked"
	rp.Unlocked.ID = readPoolKey(ssc.ID, t.ClientID) + ":unlocked"

	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	return string(rp.Encode()), nil
}

// service method used to check client balance before filling a pool
func (ssc *StorageSmartContract) checkFill(t *transaction.Transaction,
	balances chainState.StateContextI) (err error) {

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

	if t.Value < 0 {
		return errors.New("negative transaction value given")
	}

	return
}

// lock tokens for read pool of transaction's client
func (ssc *StorageSmartContract) readPoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// configs

	var conf *readPoolConfig
	if conf, err = ssc.getReadPoolConfig(balances, true); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// transaction value

	if t.Value < conf.MinLock {
		return "", common.NewError("read_pool_lock_failed",
			"insufficient amount to lock")
	}

	// lock request

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// related allocation

	var alloc *StorageAllocation
	if alloc, err = ssc.getAllocation(lr.AllocationID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get allocation: "+err.Error())
	}

	// validate the request
	if err = lr.validate(conf, alloc); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// user read pool

	var rp *readPool
	if rp, err = ssc.getReadPool(t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get read pool: "+err.Error())
	}

	// check client balance
	if err = ssc.checkFill(t, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// lock

	if _, resp, err = rp.fill(t, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	var re = rp.addLocks(t.CreationDate, state.Balance(t.Value), &lr, alloc)
	// move the rounding error to unlocked
	if re > 0 {
		// move tokens
		_, _, err = rp.Locked.TransferTo(rp.Unlocked, re, nil)
		if err != nil {
			return "", common.NewError("read_pool_lock_failed",
				"unlocking rounding error: "+err.Error())
		}
	}

	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't save read pool: "+err.Error())
	}

	return
}

// unlock tokens of all expire locks
func (ssc *StorageSmartContract) readPoolUnlock(t *transaction.Transaction,
	_ []byte, balances chainState.StateContextI) (resp string, err error) {

	// user read pool

	var rp *readPool
	if rp, err = ssc.getReadPool(t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// the request

	if _, err = rp.update(t.CreationDate); err != nil {
		return "", common.NewError("read_pool_unlock_failed",
			"can't update read pool: "+err.Error())
	}

	if rp.Unlocked.Balance == 0 {
		return "", common.NewError("read_pool_unlock_failed",
			"no tokens to unlock")
	}

	// transfer all unlocked tokens to client

	var transfer *state.Transfer
	transfer, resp, err = rp.Unlocked.EmptyPool(ssc.ID, t.ClientID, nil)
	if err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// save the read pool
	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_unlock_failed",
			"can't save read pool: "+err.Error())
	}

	return
}

//
// stat
//

// statistic for all locked tokens of the read pool
func (ssc *StorageSmartContract) getReadPoolStatHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = datastore.Key(params.Get("client_id"))
		rp       *readPool
	)
	if rp, err = ssc.getReadPool(clientID, balances); err != nil {
		return
	}

	return rp.stat(), nil
}

// statistic for a blobber by its read marker
func (ssc *StorageSmartContract) getReadPoolBlobberHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID     = datastore.Key(params.Get("client_id"))
		allocationID = datastore.Key(params.Get("allocation_id"))
		blobberID    = datastore.Key(params.Get("blobber_id"))
		rp           *readPool
	)
	if rp, err = ssc.getReadPool(clientID, balances); err != nil {
		return
	}

	var stat = rp.stat()
	if stat.Locks == nil {
		return nil, errors.New("no such allocation") // no allocations at all
	}
	var blobbers, ok = stat.Locks[allocationID]
	if !ok {
		return nil, errors.New("no such allocation")
	}
	var locks []*readPoolLock
	if locks, ok = blobbers[blobberID]; !ok {
		return nil, errors.New("no such blobber")
	}

	return locks, nil
}
