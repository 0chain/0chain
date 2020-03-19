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

	wp.ClientID = writePoolVal.ClientID

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

// setExpiration of the locked tokens
func (wp *writePool) setExpiration(set common.Timestamp) (err error) {
	if set == 0 {
		return // as is
	}
	tl, ok := wp.TokenLockInterface.(*tokenLock)
	if !ok {
		return fmt.Errorf(
			"invalid write pool state, invalid token lock type: %T",
			wp.TokenLockInterface)
	}
	tl.Duration = time.Duration(set-tl.StartTime) * time.Second
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

// create, fill and save write pool for new allocation
func (ssc *StorageSmartContract) createWritePool(t *transaction.Transaction,
	sa *StorageAllocation, balances chainState.StateContextI) (err error) {

	// create related write_pool expires with the allocation + challenge
	// completion time
	var wp *writePool
	wp, err = ssc.newWritePool(sa.GetKey(ssc.ID), t.ClientID, t.CreationDate,
		sa.Expiration+toSeconds(sa.ChallengeCompletionTime), balances)
	if err != nil {
		return fmt.Errorf("can't create write pool: %v", err)
	}

	// lock required number of tokens

	var minLockDemand = sa.minLockDemandLeft()

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
	if err = wp.save(ssc.ID, sa.ID, balances); err != nil {
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

// update write pool expiration
func (ssc *StorageSmartContract) updateWritePoolExpiration(
	t *transaction.Transaction, allocID string, expiried common.Timestamp,
	balances chainState.StateContextI) (err error) {

	var wp *writePool
	if wp, err = ssc.getWritePool(allocID, balances); err != nil {
		return
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = ssc.checkFill(t, balances); err != nil {
			return
		}
		if _, _, err = wp.fill(t, balances); err != nil {
			return
		}
	}

	if err = wp.setExpiration(expiried); err != nil {
		return
	}

	if err = wp.save(ssc.ID, allocID, balances); err != nil {
		return
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
