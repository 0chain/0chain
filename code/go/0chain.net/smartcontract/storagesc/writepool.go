package storagesc

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/core/logging"
	"0chain.net/core/util"

	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool"

	"encoding/json"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

//
// client write pool (consist of allocation pools)
//

func writePoolKey(scKey, clientID string) datastore.Key {
	return datastore.Key(scKey + ":writepool:" + clientID)
}

// writePool represents client's write pool consist of allocation write pools
type writePool struct {
	Pools allocationPools `json:"pools"` // tokens locked for a period
}

func (wp *writePool) blobberCut(allocID, blobberID string, now common.Timestamp,
) []*allocationPool {

	return wp.Pools.blobberCut(allocID, blobberID, now)
}

// Encode implements util.Serializable interface.
func (wp *writePool) Encode() []byte {
	var b, err = json.Marshal(wp)
	if err != nil {
		panic(err) // must never happens
	}
	return b
}

// Decode implements util.Serializable interface.
func (wp *writePool) Decode(p []byte) error {
	return json.Unmarshal(p, wp)
}

// save the pool in tree
func (wp *writePool) save(sscKey, clientID string,
	balances chainState.StateContextI) (err error) {

	r, err := balances.InsertTrieNode(writePoolKey(sscKey, clientID), wp)
	logging.Logger.Debug("write pool safe", zap.String("root", r))

	return
}

// take write pool by ID to unlock (the take is get and remove)
func (wp *writePool) take(poolID string, now common.Timestamp) (
	took *allocationPool, err error) {

	var i int
	for _, ap := range wp.Pools {
		if ap.ID == poolID {
			if ap.ExpireAt >= now {
				return nil, errors.New("the pool is not expired yet")
			}
			took = ap
			continue // delete
		}
		wp.Pools[i], i = ap, i+1
	}
	wp.Pools = wp.Pools[:i]

	if took == nil {
		return nil, errors.New("pool not found")
	}
	return
}

func (wp *writePool) getPool(poolID string) *allocationPool {
	for _, ap := range wp.Pools {
		if ap.ID == poolID {
			return ap
		}
	}
	return nil
}

func (wp *writePool) allocPool(allocID string, until common.Timestamp) (
	ap *allocationPool) {

	var zero *allocationPool
	for _, ap := range wp.Pools.allocationCut(allocID) {
		if ap.ExpireAt == until {
			return ap
		}
		if ap.ExpireAt == 0 {
			zero = ap
		}
	}
	return zero
}

func (wp *writePool) stat(now common.Timestamp) (aps allocationPoolsStat) {
	aps = wp.Pools.stat(now)
	return
}

func makeCopyAllocationBlobbers(alloc StorageAllocation, value int64) blobberPools {
	var bps blobberPools
	var total float64
	for _, b := range alloc.BlobberAllocs {
		total += float64(b.Terms.WritePrice)
	}
	for _, b := range alloc.BlobberAllocs {
		var ratio = float64(b.Terms.WritePrice) / total
		bps.add(&blobberPool{
			Balance:   currency.Coin(float64(value) * ratio),
			BlobberID: b.BlobberID,
		})
	}
	return bps
}

func (wp *writePool) allocUntil(allocID string, until common.Timestamp) (
	value currency.Coin) {

	return wp.Pools.allocUntil(allocID, until)
}

//
// smart contract methods
//

// getWritePool of current client
func (ssc *StorageSmartContract) getWritePool(clientID datastore.Key,
	balances chainState.StateContextI) (wp *writePool, err error) {
	wp = new(writePool)
	err = balances.GetTrieNode(writePoolKey(ssc.ID, clientID), wp)
	if err != nil {
		return nil, err
	}

	return wp, nil
}

func (ssc *StorageSmartContract) createEmptyWritePool(
	txn *transaction.Transaction,
	alloc *StorageAllocation,
	balances chainState.StateContextI,
) (err error) {
	var wp *writePool
	wp, err = ssc.getWritePool(alloc.Owner, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return fmt.Errorf("getting client write pool: %v", err)
	}
	if err == util.ErrValueNotPresent {
		wp = new(writePool)
	}

	var ap = allocationPool{
		AllocationID: alloc.ID,
		ExpireAt:     alloc.Until(),
		Blobbers:     makeCopyAllocationBlobbers(*alloc, txn.ValueZCN),
	}
	ap.TokenPool.ID = txn.Hash
	alloc.addWritePoolOwner(alloc.Owner)
	wp.Pools.add(&ap)

	if err = wp.save(ssc.ID, alloc.Owner, balances); err != nil {
		return fmt.Errorf("saving write pool: %v", err)
	}

	return
}

func (ssc *StorageSmartContract) createWritePool(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	mintNewTokens bool,
	balances chainState.StateContextI,
) (err error) {
	var wp *writePool
	wp, err = ssc.getWritePool(alloc.Owner, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return fmt.Errorf("getting client write pool: %v", err)
	}

	if err == util.ErrValueNotPresent {
		wp = new(writePool)
	}

	var mld = alloc.restMinLockDemand()
	if t.ValueZCN < int64(mld) || t.ValueZCN <= 0 {
		return fmt.Errorf("not enough tokens to honor the min lock demand"+
			" (%d < %d)", t.ValueZCN, mld)
	}

	if t.ValueZCN > 0 {
		var until = alloc.Until()
		ap, err := newAllocationPool(t, alloc, until, mintNewTokens, balances)
		if err != nil {
			return err
		}
		wp.Pools.add(ap)
	}

	if err = wp.save(ssc.ID, alloc.Owner, balances); err != nil {
		return fmt.Errorf("saving write pool: %v", err)
	}

	return
}

// lock tokens for write pool of transaction's client
func (ssc *StorageSmartContract) writePoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	var conf *writePoolConfig
	if conf, err = ssc.getWritePoolConfig(balances, true); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// lock request & user balance

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if len(lr.TargetId) == 0 {
		lr.TargetId = t.ClientID
	}

	var wp *writePool
	if wp, err = ssc.getWritePool(lr.TargetId, balances); err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("write_pool_lock_failed", err.Error())
		}
		wp = new(writePool)
	}

	if lr.AllocationID == "" {
		return "", common.NewError("write_pool_lock_failed",
			"missing allocation ID in request")
	}

	txnSAS, err := currency.ParseZCN(float64(t.ValueZCN))
	if txnSAS < conf.MinLock || t.ValueZCN <= 0 {
		return "", common.NewError("write_pool_lock_failed",
			"insufficient amount to lock")
	}

	if lr.Duration < conf.MinLockPeriod {
		return "", common.NewError("write_pool_lock_failed",
			fmt.Sprintf("duration (%s) is shorter than min lock period (%s)",
				lr.Duration.String(), conf.MinLockPeriod.String()))
	}

	if lr.Duration > conf.MaxLockPeriod {
		return "", common.NewError("write_pool_lock_failed",
			fmt.Sprintf("duration (%s) is longer than max lock period (%v)",
				lr.Duration.String(), conf.MaxLockPeriod.String()))
	}

	// check client balance
	if err = stakepool.CheckClientBalance(t, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// get the allocation object
	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(lr.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get allocation: "+err.Error())
	}

	var bps blobberPools

	// lock for allocation -> blobber (particular blobber locking)
	if lr.BlobberID != "" {
		if _, ok := alloc.BlobberAllocsMap[lr.BlobberID]; !ok {
			return "", common.NewError("write_pool_lock_failed",
				fmt.Sprintf("no such blobber %s in allocation %s",
					lr.BlobberID, lr.AllocationID))
		}
		bps = append(bps, &blobberPool{
			Balance:   currency.Coin(t.ValueZCN),
			BlobberID: lr.BlobberID,
		})
	} else {
		// divide depending write price range for all blobbers of the
		// allocation
		var total float64 // total write price
		for _, b := range alloc.BlobberAllocs {
			total += float64(b.Terms.WritePrice)
		}
		// calculate (divide)
		for _, b := range alloc.BlobberAllocs {
			var ratio = float64(b.Terms.WritePrice) / total
			bps.add(&blobberPool{
				Balance:   currency.Coin(float64(t.ValueZCN) * ratio),
				BlobberID: b.BlobberID,
			})
		}
	}

	// create and dig allocation pool

	var (
		ap       allocationPool
		transfer *state.Transfer
	)
	if transfer, resp, err = ap.DigPool(t.Hash, t); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// set fields
	ap.AllocationID = lr.AllocationID
	ap.ExpireAt = t.CreationDate + toSeconds(lr.Duration)
	ap.Blobbers = bps

	// add and save
	alloc.addWritePoolOwner(t.ClientID)
	wp.Pools.add(&ap)
	if err = wp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// remembers who funded the write pool, so tokens get returned to funder on unlock
	if err := ssc.addToFundedPools(t.ClientID, ap.ID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// save new linked allocation pool
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("write_pool_lock_failed",
			"saving allocation: %v", err)
	}
	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) writePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	var (
		transfer *state.Transfer
		req      unlockRequest
	)

	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	if len(req.PoolOwner) == 0 {
		req.PoolOwner = t.ClientID
	}

	isFunded, err := ssc.isFundedPool(t.ClientID, req.PoolID, balances)
	if err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}
	if !isFunded {
		return "", common.NewErrorf("read_pool_unlock_failed",
			"%s did not fund pool %s", t.ClientID, req.PoolID)
	}

	var wp *writePool
	if wp, err = ssc.getWritePool(req.PoolOwner, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// don't unlock over min lock demand left
	var ap = wp.getPool(req.PoolID)
	if ap == nil {
		return "", common.NewError("write_pool_unlock_failed",
			"no such write pool")
	}

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(ap.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_unlock_failed",
			"can't get related allocation: "+err.Error())
	}

	if !alloc.Finalized && !alloc.Canceled {
		var (
			want  = alloc.restMinLockDemand()
			unitl = alloc.Until()
			leave = wp.allocUntil(ap.AllocationID, unitl) - ap.Balance
		)
		if leave < want && ap.ExpireAt >= unitl {
			return "", common.NewError("write_pool_unlock_failed",
				"can't unlock, because min lock demand is not paid yet")
		}
	}

	if ap, err = wp.take(req.PoolID, t.CreationDate); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	transfer, resp, err = ap.EmptyPool(ssc.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// save write pools
	if err = wp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	return
}

//
// stat
//
