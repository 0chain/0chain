package storagesc

import (
	"0chain.net/smartcontract"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

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

func (wp *writePool) removeEmpty(allocID string, ap []*allocationPool) {
	wp.Pools.removeEmpty(allocID, ap)
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

	_, err = balances.InsertTrieNode(writePoolKey(sscKey, clientID), wp)
	return
}

func (wp *writePool) moveToChallenge(allocID, blobID string,
	cp *challengePool, now common.Timestamp, value state.Balance) (err error) {

	if value == 0 {
		return // nothing to move, ok
	}

	var cut = wp.blobberCut(allocID, blobID, now)

	if len(cut) == 0 {
		return fmt.Errorf("no tokens in write pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	var torm []*allocationPool // to remove later (empty allocation pools)
	for _, ap := range cut {
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
		if _, _, err = ap.TransferTo(cp, move, nil); err != nil {
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
		return fmt.Errorf("not enough tokens in write pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	// remove empty allocation pools
	wp.removeEmpty(allocID, torm)
	return
}

func (wp *writePool) movePartToStake(sscKey string, ap *allocationPool,
	sp *stakePool, value state.Balance, balances chainState.StateContextI) (
	moved state.Balance, err error) {

	var stake = float64(sp.stake())
	for _, dp := range sp.orderedPools() {
		var ratio float64
		if stake == 0.0 {
			ratio = 1.0 / float64(len(sp.Pools))
		} else {
			ratio = float64(dp.Balance) / stake
		}
		var (
			move     = state.Balance(float64(value) * ratio)
			transfer *state.Transfer
		)
		transfer, _, err = ap.DrainPool(sscKey, dp.DelegateID, move, nil)
		if err != nil {
			return 0, fmt.Errorf("transferring tokens"+
				" write_pool/alloc_pool(%s) -> stake_pool_holder(%s): %v",
				ap.ID, dp.DelegateID, err)
		}
		if err = balances.AddTransfer(transfer); err != nil {
			return 0, fmt.Errorf("adding transfer: %v", err)
		}
		// stat
		dp.Rewards += move           // add to stake_pool_holder rewards
		sp.Rewards.Validator += move // add to total blobber rewards
		moved += move
	}

	return
}

func (wp *writePool) moveToStake(sscKey, allocID, blobID string,
	sp *stakePool, now common.Timestamp, value state.Balance,
	balances chainState.StateContextI) (err error) {

	var cut = wp.blobberCut(allocID, blobID, now)

	if len(cut) == 0 {
		return fmt.Errorf("no tokens in write pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	var torm []*allocationPool // to remove later (empty allocation pools)
	for _, ap := range cut {
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
		_, err = wp.movePartToStake(sscKey, ap, sp, move, balances)
		if err != nil {
			return
		}
		sp.Rewards.Blobber += move
		value -= move
		if bp.Balance == 0 {
			ap.Blobbers.removeByIndex(bi)
		}
		if ap.Balance == 0 {
			torm = append(torm, ap) // remove the allocation pool later
		}
	}

	if value != 0 {
		return fmt.Errorf("not enough tokens in write pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	// remove empty allocation pools
	wp.removeEmpty(allocID, torm)
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

func (wp *writePool) fill(t *transaction.Transaction, alloc *StorageAllocation,
	until common.Timestamp, balances chainState.StateContextI) (
	resp string, err error) {

	var bps blobberPools
	if err = checkFill(t, balances); err != nil {
		return
	}
	var total float64
	for _, b := range alloc.BlobberDetails {
		total += float64(b.Terms.WritePrice)
	}
	for _, b := range alloc.BlobberDetails {
		var ratio = float64(b.Terms.WritePrice) / total
		bps.add(&blobberPool{
			Balance:   state.Balance(float64(t.Value) * ratio),
			BlobberID: b.BlobberID,
		})
	}
	var (
		ap       allocationPool
		transfer *state.Transfer
	)
	if transfer, resp, err = ap.DigPool(t.Hash, t); err != nil {
		return "", fmt.Errorf("digging write pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer to write pool: %v", err)
	}

	// set fields
	ap.AllocationID = alloc.ID
	ap.ExpireAt = until
	ap.Blobbers = bps

	// add the allocation pool
	wp.Pools.add(&ap)
	return
}

func (wp *writePool) allocUntil(allocID string, until common.Timestamp) (
	value state.Balance) {

	return wp.Pools.allocUntil(allocID, until)
}

//
// smart contract methods
//

// getWritePoolBytes of a client
func (ssc *StorageSmartContract) getWritePoolBytes(clientID datastore.Key,
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(writePoolKey(ssc.ID, clientID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getWritePool of current client
func (ssc *StorageSmartContract) getWritePool(clientID datastore.Key,
	balances chainState.StateContextI) (wp *writePool, err error) {

	var poolb []byte
	if poolb, err = ssc.getWritePoolBytes(clientID, balances); err != nil {
		return
	}
	wp = new(writePool)
	err = wp.Decode(poolb)
	return
}

func (ssc *StorageSmartContract) createWritePool(t *transaction.Transaction,
	alloc *StorageAllocation, balances chainState.StateContextI) (err error) {

	var wp *writePool
	wp, err = ssc.getWritePool(alloc.Owner, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return fmt.Errorf("getting client write pool: %v", err)
	}

	if err == util.ErrValueNotPresent {
		wp = new(writePool)
	}

	var mld = alloc.restMinLockDemand()
	if t.Value < int64(mld) {
		return fmt.Errorf("not enough tokens to honor the min lock demand"+
			" (%d < %d)", t.Value, mld)
	}

	if t.Value > 0 {
		var until = alloc.Until()
		if _, err = wp.fill(t, alloc, until, balances); err != nil {
			return
		}
	}

	if err = wp.save(ssc.ID, alloc.Owner, balances); err != nil {
		return fmt.Errorf("saving write pool: %v", err)
	}

	return
}

// lock tokens for write pool of transaction's client
func (ssc *StorageSmartContract) writePoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// configs

	var conf *writePoolConfig
	if conf, err = ssc.getWritePoolConfig(balances, true); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// user write pools

	var wp *writePool
	if wp, err = ssc.getWritePool(t.ClientID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// lock request & user balance

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// check

	if lr.AllocationID == "" {
		return "", common.NewError("write_pool_lock_failed",
			"missing allocation ID in request")
	}

	if t.Value < conf.MinLock {
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
	if err = checkFill(t, balances); err != nil {
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
		if _, ok := alloc.BlobberMap[lr.BlobberID]; !ok {
			return "", common.NewError("write_pool_lock_failed",
				fmt.Sprintf("no such blobber %s in allocation %s",
					lr.BlobberID, lr.AllocationID))
		}
		bps = append(bps, &blobberPool{
			Balance:   state.Balance(t.Value),
			BlobberID: lr.BlobberID,
		})
	} else {
		// divide depending write price range for all blobbers of the
		// allocation
		var total float64 // total write price
		for _, b := range alloc.BlobberDetails {
			total += float64(b.Terms.WritePrice)
		}
		// calculate (divide)
		for _, b := range alloc.BlobberDetails {
			var ratio = float64(b.Terms.WritePrice) / total
			bps.add(&blobberPool{
				Balance:   state.Balance(float64(t.Value) * ratio),
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

	wp.Pools.add(&ap)
	if err = wp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) writePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// user write pool

	var wp *writePool
	if wp, err = ssc.getWritePool(t.ClientID, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// the request

	var (
		transfer *state.Transfer
		req      unlockRequest
	)

	if err = req.decode(input); err != nil {
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

// statistic for an allocation/blobber (used by blobbers)
func (ssc *StorageSmartContract) getWritePoolAllocBlobberStatHandler(
	ctx context.Context, params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID  = params.Get("client_id")
		allocID   = params.Get("allocation_id")
		blobberID = params.Get("blobber_id")
		wp        *writePool
	)

	if wp, err = ssc.getWritePool(clientID, balances); err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingWritePoolErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}

	var (
		cut  = wp.blobberCut(allocID, blobberID, common.Now())
		stat []untilStat
	)

	for _, ap := range cut {
		var bp, ok = ap.Blobbers.get(blobberID)
		if !ok {
			continue
		}
		stat = append(stat, untilStat{
			PoolID:   ap.ID,
			Balance:  bp.Balance,
			ExpireAt: ap.ExpireAt,
		})
	}

	return &stat, nil
}

// statistic for all locked tokens of the write pool
func (ssc *StorageSmartContract) getWritePoolStatHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = datastore.Key(params.Get("client_id"))
		wp       *writePool
	)

	if wp, err = ssc.getWritePool(clientID, balances); err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingWritePoolErr, err)
		return nil, smartcontract.WrapErrNoResource(err)
	}

	return wp.stat(common.Now()), nil
}
