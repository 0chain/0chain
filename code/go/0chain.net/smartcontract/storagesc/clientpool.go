package storagesc

import (
	"0chain.net/smartcontract"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//
// client read/write pool
//

func clientPoolKey(scKey, clientID string) datastore.Key {
	return datastore.Key(scKey + ":clientpool:" + clientID)
}

type clientPool struct {
	Allocations allocationPools `json:"allocations"`
}

// Encode pool
// Implements util.Serializable interface.
func (cPool *clientPool) Encode() []byte {
	var b, err = json.Marshal(cPool)
	if err != nil {
		panic(err) // must never happens
	}
	return b
}

// Decode pool
// Implements util.Serializable interface
func (cPool *clientPool) Decode(p []byte) error {
	return json.Unmarshal(p, cPool)
}

// Save pool in tree
func (cPool *clientPool) save(ssContractKey, clientID string, balances cstate.StateContextI,
) (err error) {
	_, err = balances.InsertTrieNode(clientPoolKey(ssContractKey, clientID), cPool)
	return
}

// todo: add description, rename it
func (cPool *clientPool) blobberCut(aPoolID, bPoolID string, eTimestamp common.Timestamp,
) []*allocationPool {
	return cPool.Allocations.blobberCut(aPoolID, bPoolID, eTimestamp)
}

// Former removeEmpty
// todo: add description
// todo: clarify and rename - to remove allocation or allocations?
func (cPool *clientPool) removeEmptyAllocation(aPoolID string, aPools []*allocationPool) {
	cPool.Allocations.removeEmpty(aPoolID, aPools)
}

// For read operations only
// todo: better rename it, adding "Read" in
// todo: something probably wrong with it.
// todo: clientPool entity should be used insted of some params to be passed?
func (cPool *clientPool) moveBlobberCharge(ssContractKey string, sPool *stakePool,
	aPool *allocationPool, value state.Balance, balances cstate.StateContextI,
) (err error) {
	if value == 0 {
		return // avoid insufficient transfer
	}

	var (
		dw = sPool.Settings.DelegateWallet
		transfer *state.Transfer
	)
	transfer, _, err = aPool.DrainPool(ssContractKey, dw, value, nil)
	if err != nil {
		return fmt.Errorf("transferring tokens client_pool() -> "+
			"blobber_charge(%s): %v", dw, err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return fmt.Errorf("adding transfer: %v", err)
	}

	// blobber service charge
	sPool.Rewards.Charge += value
	return
}

// For read operations only
// todo: better rename it, adding "Read" in
func (cPool *clientPool) movePartToBlobber(ssContractKey string, aPool *allocationPool,
	sPool *stakePool, value state.Balance, balances cstate.StateContextI,
) (err error) {
	var blobberCharge state.Balance
	blobberCharge = state.Balance(sPool.Settings.ServiceCharge * float64(value))
	err = cPool.moveBlobberCharge(ssContractKey, sPool, aPool, blobberCharge, balances)
	if err != nil {
		return
	}

	value = value - blobberCharge // left for stake holders

	if value == 0 {
		return // avoid insufficient transfer
	}

	var stake = float64(sPool.stake())
	for _, dp := range sPool.orderedPools() {
		var ratio float64
		if stake == 0.0 {
			ratio = float64(dp.Balance) / float64(len(sPool.Pools))
		} else {
			ratio = float64(dp.Balance) / stake
		}

		var (
			move = state.Balance(float64(value) * ratio)
			transfer *state.Transfer
		)
		transfer, _, err = aPool.DrainPool(ssContractKey, dp.DelegateID, move, nil)
		if err != nil {
			return fmt.Errorf("transferring tokens client_pool() -> "+
				"stake_pool_holder(%s): %v", dp.DelegateID, err)
		}
		if err = balances.AddTransfer(transfer); err != nil {
			return fmt.Errorf("adding transfer: %v", err)
		}
		// stat
		dp.Rewards += move         // add to stake_pool_holder rewards
		sPool.Rewards.Blobber += move // add to total blobber rewards
	}

	return
}

// For write operations only
// todo: better rename it, adding "Write" in
func (cPool *clientPool) moveToChallenge(allocID, blobID string,
	chPool *challengePool, now common.Timestamp, value state.Balance,
) (err error) {
	if value == 0 {
		return // nothing to move, ok
	}

	var aPools = cPool.blobberCut(allocID, blobID, now)

	if len(aPools) == 0 {
		return fmt.Errorf("no tokens in client pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	var torm []*allocationPool // to remove later (empty allocation pools)
	for _, ap := range aPools {
		if value == 0 {
			break // all required tokens has moved to the blobber
		}
		var bi, ok = ap.Blobbers.getIndex(blobID)
		if !ok {
			continue // impossible case, but leave the check here
		}
		var (
			bp   = ap.Blobbers[bi]
			move state.Balance
		)
		if value >= bp.Balance {
			move, bp.Balance = bp.Balance, 0
		} else {
			move, bp.Balance = value, bp.Balance-value
		}
		if _, _, err = ap.TransferTo(chPool, move, nil); err != nil {
			return // transferring error
		}
		value -= move
		if bp.Balance == 0 {
			ap.Blobbers.removeByIndex(bi)
		}
		if ap.Balance == 0 {
			torm = append(torm, ap) // remove the allocation pool later
		}
	}

	if value != 0 {
		return fmt.Errorf("not enough tokens in client pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	// remove empty allocation pool(s?)
	cPool.removeEmptyAllocation(allocID, torm)
	return
}

// The clientPoolReadRedeem represents part of response of read markers redeeming.
// A Blobber uses this response for internal client pools cache.
// Former readPoolRedeem
// todo: should it be reafactored?
type clientPoolReadRedeem struct {
	PoolID  string        `json:"pool_id"` // client pool ID
	Balance state.Balance `json:"balance"` // balance reduction
}

// For read operations only
// todo: better rename it, adding "Read" in
func (cPool *clientPool) moveToBlobber(ssContractKey, allocID, blobID string,
	sPool *stakePool, now common.Timestamp, value state.Balance,
	balances cstate.StateContextI,
) (resp string, err error) {
	var cut = cPool.blobberCut(allocID, blobID, now)

	if len(cut) == 0 {
		return "", fmt.Errorf("no tokens in client pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	// all redeems to response at the end
	var redeems []clientPoolReadRedeem

	var torm []*allocationPool // to remove later (empty allocation pools)
	for _, ap := range cut {
		if value == 0 {
			break // all required tokens has moved to the blobber
		}

		var bIndex, ok = ap.Blobbers.getIndex(blobID)
		if !ok {
			continue // impossible case, but leave the check here
		}

		var (
			bp = ap.Blobbers[bIndex]
			move state.Balance
		)
		if value >= bp.Balance {
			move, bp.Balance = bp.Balance, 0
		} else {
			move, bp.Balance = value, bp.Balance-value
		}

		err = cPool.movePartToBlobber(ssContractKey, ap, sPool, move, balances)
		if err != nil {
			return // fatal, can't move, can't continue, rollback all
		}

		redeems = append(redeems, clientPoolReadRedeem{
			PoolID:  ap.ID,
			Balance: move,
		})

		value -= move
		sPool.Rewards.Blobber += value
		if bp.Balance == 0 {
			ap.Blobbers.removeByIndex(bIndex)
		}
		if ap.Balance == 0 {
			torm = append(torm, ap) // remove the allocation pool later
		}
	}

	if value != 0 {
		return "", fmt.Errorf("not enough tokens in client pool for "+
			"allocation: %s, blobber: %s", allocID, blobID)
	}

	// remove empty allocation pool(s)
	cPool.removeEmptyAllocation(allocID, torm)

	// return the read redeems for blobbers read pools cache
	var respb []byte
	respb, err = json.Marshal(redeems)
	if err != nil {
		panic(err) // must not happen / from the very legacy code
	}

	return string(respb), nil
}

// take pool by ID to unlock (the take is get and remove)
// Former readPool.take, etc
// todo: investigate, refactor
// todo: poor quality. poor naming.
func (cPool *clientPool) takeAllocationPool(aPoolID string, eTimestamp common.Timestamp,
) (aPool *allocationPool, err error) {
	var i int
	for _, ap := range cPool.Allocations {
		if ap.ID == aPoolID {
			if ap.ExpireAt >= eTimestamp {
				return nil, errors.New("client pool is not expired yet")
			}
			aPool = ap
			continue // delete
		}
		cPool.Allocations[i], i = ap, i + 1
	}
	cPool.Allocations = cPool.Allocations[:i]

	if aPool == nil {
		return nil, errors.New("pool not found")
	}

	return
}

func (cPool *clientPool) fill(t *transaction.Transaction, alloc *StorageAllocation,
	until common.Timestamp, balances cstate.StateContextI,
) (resp string, err error) {
	var bPools blobberPools
	if err = checkFill(t, balances); err != nil {
		return
	}
	var total float64
	for _, b := range alloc.BlobberDetails {
		total += float64(b.Terms.WritePrice)
	}
	for _, b := range alloc.BlobberDetails {
		var ratio = float64(b.Terms.WritePrice) / total
		bPools.add(&blobberPool{
			Balance:   state.Balance(float64(t.Value) * ratio),
			BlobberID: b.BlobberID,
		})
	}
	var (
		aPool allocationPool
		transfer *state.Transfer
	)
	if transfer, resp, err = aPool.DigPool(t.Hash, t); err != nil {
		return "", fmt.Errorf("digging client pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer to client pool: %v", err)
	}

	// set fields
	aPool.AllocationID = alloc.ID
	aPool.ExpireAt = until
	aPool.Blobbers = bPools

	// add the allocation pool
	cPool.Allocations.add(&aPool)
	return
}

// Former writePool.getPool
// todo: investigate, refactor
func (cPool *clientPool) getPool(poolID string) *allocationPool {
	for _, aPool := range cPool.Allocations {
		if aPool.ID == poolID {
			return aPool
		}
	}
	return nil
}

// Former writePool.allocPool
// todo: investigate, refactor
func (cPool *clientPool) allocPool(allocID string, until common.Timestamp,
) (aPool *allocationPool) {
	for _, ap := range cPool.Allocations.allocationCut(allocID) {
		if ap.ExpireAt == until {
			return ap
		}
		if ap.ExpireAt == 0 {
			aPool = ap
		}
	}
	return
}

func (cPool *clientPool) allocUntil(allocID string, until common.Timestamp,
) (value state.Balance) {
	return cPool.Allocations.allocUntil(allocID, until)
}

func (cPool *clientPool) stat(now common.Timestamp) allocationPoolsStat {
	return cPool.Allocations.stat(now)
}

//
// smart contract methods
//

// Get encoded client pool for a client
func (ssContract *StorageSmartContract) getClientPoolBytes(clientID datastore.Key,
	balances cstate.StateContextI,
) (b []byte, err error) {
	var val util.Serializable
	val, err = balances.GetTrieNode(clientPoolKey(ssContract.ID, clientID))
	if err != nil {
		return
	}

	return val.Encode(), nil
}

// Get client pool for a client
func (ssContract *StorageSmartContract) getClientPool(clientID datastore.Key,
	balances cstate.StateContextI,
) (cPool *clientPool, err error) {
	var cPoolBytes []byte
	if cPoolBytes, err = ssContract.getClientPoolBytes(clientID, balances); err != nil {
		return
	}

	cPool = new(clientPool)
	err = cPool.Decode(cPoolBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}

	return
}

// Create new pool for a client
// Former "newReadPool", "createWritePool"
// todo: or addClientPool?
func (ssContract *StorageSmartContract) createClientPool(t *transaction.Transaction,
	balances cstate.StateContextI,
) (cPoolEncoded string, err error) {
	_, err = ssContract.getClientPoolBytes(t.ClientID, balances)

	if err == nil {
		return "", common.NewError("new_client_pool_failed", "already exists")
	}

	if err != util.ErrValueNotPresent {
		return "", common.NewError("new_client_pool_failed", err.Error())
	}

	var cPool = new(clientPool)
	if err = cPool.save(ssContract.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("new_client_pool_failed", err.Error())
	}

	return string(cPool.Encode()), nil

	// todo: implement
	// derived from WritePool
	// if err == util.ErrValueNotPresent {
	// 	wp = new(writePool)
	// }

	// var mld = alloc.restMinLockDemand()
	// if t.Value < int64(mld) {
	// 	return fmt.Errorf("not enough tokens to honor the min lock demand"+
	// 		" (%d < %d)", t.Value, mld)
	// }

	// if t.Value > 0 {
	// 	var until = alloc.Until()
	// 	if _, err = wp.fill(t, alloc, until, balances); err != nil {
	// 		return
	// 	}
	// }

	// if err = wp.save(ssContract.ID, alloc.Owner, balances); err != nil {
	// 	return fmt.Errorf("saving write pool: %v", err)
	// }

	// return
}

// Lock tokens for the pool of transaction's client
func (ssContract *StorageSmartContract) lockClientPool(t *transaction.Transaction,
	lockRequestBytes []byte, balances cstate.StateContextI,
) (cPoolEncoded string, err error) {
	// get config

	var conf *clientPoolConfig
	if conf, err = ssContract.getClientPoolConfig(balances, true); err != nil {
		return "", common.NewError("client_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// get client pool

	var cPool *clientPool
	if cPool, err = ssContract.getClientPool(t.ClientID, balances); err != nil {
		return "", common.NewError("client_pool_lock_failed", err.Error())
	}

	// get lock request & user balance

	var lRequest lockRequest
	if err = lRequest.decode(lockRequestBytes); err != nil {
		return "", common.NewError("client_pool_lock_failed", err.Error())
	}

	// check

	if lRequest.AllocationID == "" {
		return "", common.NewError("client_pool_lock_failed",
			"missing allocation ID in request")
	}

	if t.Value < conf.MinLock {
		return "", common.NewError("client_pool_lock_failed",
			"insufficient amount to lock")
	}

	if lRequest.Duration < conf.MinLockPeriod {
		return "", common.NewError("client_pool_lock_failed",
			fmt.Sprintf("duration (%s) is shorter than min lock period (%s)",
				lRequest.Duration.String(), conf.MinLockPeriod.String()))
	}

	if lRequest.Duration > conf.MaxLockPeriod {
		return "", common.NewError("client_pool_lock_failed",
			fmt.Sprintf("duration (%s) is longer than max lock period (%v)",
				lRequest.Duration.String(), conf.MaxLockPeriod.String()))
	}

	// check client balance
	if err = checkFill(t, balances); err != nil {
		return "", common.NewError("client_pool_lock_failed", err.Error())
	}

	// get the allocation object
	var alloc *StorageAllocation
	alloc, err = ssContract.getAllocation(lRequest.AllocationID, balances)
	if err != nil {
		return "", common.NewError("client_pool_lock_failed",
			"can't get allocation: "+err.Error())
	}

	var bPools blobberPools

	// lock for allocation -> blobber (particular blobber locking)
	if lRequest.BlobberID != "" {
		if _, ok := alloc.BlobberMap[lRequest.BlobberID]; !ok {
			return "", common.NewError("client_pool_lock_failed",
				fmt.Sprintf("no such blobber %s in allocation %s",
					lRequest.BlobberID, lRequest.AllocationID))
		}
		bPools = append(bPools, &blobberPool{
			Balance:   state.Balance(t.Value),
			BlobberID: lRequest.BlobberID,
		})
	} else {
		// divide depending read price range for all blobbers of the
		// allocation
		var total float64 // total read price
		for _, b := range alloc.BlobberDetails {
			total += float64(b.Terms.ReadPrice)
		}
		// calculate (divide)
		for _, b := range alloc.BlobberDetails {
			var ratio = float64(b.Terms.ReadPrice) / total
			bPools.add(&blobberPool{
				Balance:   state.Balance(float64(t.Value) * ratio),
				BlobberID: b.BlobberID,
			})
		}
	}

	// create and dig allocation pool

	var aPool allocationPool
	var transfer *state.Transfer

	if transfer, cPoolEncoded, err = aPool.DigPool(t.Hash, t); err != nil {
		return "", common.NewError("client_pool_lock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("client_pool_lock_failed", err.Error())
	}

	// set fields
	aPool.AllocationID = lRequest.AllocationID
	aPool.ExpireAt = t.CreationDate + toSeconds(lRequest.Duration)
	aPool.Blobbers = bPools

	// add and save

	cPool.Allocations.add(&aPool)
	if err = cPool.save(ssContract.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("client_pool_lock_failed", err.Error())
	}

	return
}

// Unlock tokens the pool of transaction's client if expired (what's expired?)
func (ssContract *StorageSmartContract) unlockClientPool(t *transaction.Transaction,
	unlockRequestBytes []byte, balances cstate.StateContextI,
) (cPoolEncoded string, err error) {
	// get pool

	var cPool *clientPool
	if cPool, err = ssContract.getClientPool(t.ClientID, balances); err != nil {
		return "", common.NewError("client_pool_unlock_failed", err.Error())
	}

	// get unlock request

	var (
		transfer *state.Transfer
		uRequest unlockRequest
	)

	if err = uRequest.decode(unlockRequestBytes); err != nil {
		return "", common.NewError("client_pool_unlock_failed", err.Error())
	}

	// todo: implement
	// derived from WritePool
	// // don't unlock over min lock demand left
	// var ap = wp.getPool(req.PoolID)
	// if ap == nil {
	// 	return "", common.NewError("write_pool_unlock_failed",
	// 		"no such write pool")
	// }

	// var alloc *StorageAllocation
	// alloc, err = ssc.getAllocation(ap.AllocationID, balances)
	// if err != nil {
	// 	return "", common.NewError("write_pool_unlock_failed",
	// 		"can't get related allocation: "+err.Error())
	// }

	// if !alloc.Finalized && !alloc.Canceled {
	// 	var (
	// 		want  = alloc.restMinLockDemand()
	// 		unitl = alloc.Until()
	// 		leave = wp.allocUntil(ap.AllocationID, unitl) - ap.Balance
	// 	)
	// 	if leave < want && ap.ExpireAt >= unitl {
	// 		return "", common.NewError("write_pool_unlock_failed",
	// 			"can't unlock, because min lock demand is not paid yet")
	// 	}
	// }

	// if ap, err = wp.takeAllocationPool(req.PoolID, t.CreationDate); err != nil {
	// 	return "", common.NewError("write_pool_unlock_failed", err.Error())
	// }

	var aPool *allocationPool
	if aPool, err = cPool.takeAllocationPool(uRequest.PoolID, t.CreationDate); err != nil {
		return "", common.NewError("client_pool_unlock_failed", err.Error())
	}

	transfer, cPoolEncoded, err = aPool.EmptyPool(ssContract.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("client_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("client_pool_unlock_failed", err.Error())
	}

	// save pool

	if err = cPool.save(ssContract.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("client_pool_unlock_failed", err.Error())
	}

	return
}

//
// Statistics
//

// Get statistic for allocation/blobber of a client pools (used by blobbers)
func (ssContract *StorageSmartContract) getClientPoolAllocBlobberStatHandler(
	ctx context.Context, params url.Values, balances cstate.StateContextI,
) (resp interface{}, err error) {
	var (
		clientID  = params.Get("client_id")
		allocID   = params.Get("allocation_id")
		blobberID = params.Get("blobber_id")
		cPool     *clientPool
	)

	if cPool, err = ssContract.getClientPool(clientID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true,
			"can't get retrieving client pool")
	}

	var (
		aPools  = cPool.blobberCut(allocID, blobberID, common.Now())
		stat []untilStat
	)

	for _, aPool := range aPools {
		var _, ok = aPool.Blobbers.get(blobberID)
		if !ok {
			continue
		}
		stat = append(stat, untilStat{
			PoolID:   aPool.ID,
			Balance:  aPool.Balance,
			ExpireAt: aPool.ExpireAt,
		})
	}

	return &stat, nil
}

// Get statistics of all locked tokens of a client pool
func (ssContract *StorageSmartContract) getClientPoolStatHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI,
) (resp interface{}, err error) {
	var (
		clientID = datastore.Key(params.Get("client_id"))
		cPool    *clientPool
	)

	if cPool, err = ssContract.getClientPool(clientID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true,
			"can't get client pool")
	}

	return cPool.stat(common.Now()), nil
}
