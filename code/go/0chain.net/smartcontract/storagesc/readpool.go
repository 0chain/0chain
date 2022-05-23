package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/smartcontract"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//msgp:ignore readPoolRedeem
//go:generate msgp -io=false -tests=false -unexported=true -v

//
// client read pool (consist of allocation pools)
//

func readPoolKey(scKey, clientID string) datastore.Key {
	return scKey + ":readpool:" + clientID
}

// readPool represents client's read pool consist of allocation read pools
type readPool struct {
	Pools allocationPools `json:"pools"`
}

func (rp *readPool) blobberCut(allocID, blobberID string, now common.Timestamp,
) []*allocationPool {

	return rp.Pools.blobberCut(allocID, blobberID, now)
}

func (rp *readPool) removeEmpty(allocID string, ap []*allocationPool) {
	rp.Pools.removeEmpty(allocID, ap)
}

// Encode implements util.Serializable interface.
func (rp *readPool) Encode() []byte {
	var b, err = json.Marshal(rp)
	if err != nil {
		panic(err) // must never happens
	}
	return b
}

// Decode implements util.Serializable interface.
func (rp *readPool) Decode(p []byte) error {
	return json.Unmarshal(p, rp)
}

// save the pool in tree
func (rp *readPool) save(sscKey, clientID string, balances cstate.StateContextI) (
	err error) {

	_, err = balances.InsertTrieNode(readPoolKey(sscKey, clientID), rp)
	return
}

// The readPoolRedeem represents part of response of read markers redeeming.
// A Blobber uses this response for internal read pools cache.
type readPoolRedeem struct {
	PoolID  string        `json:"pool_id"` // read pool ID
	Balance currency.Coin `json:"balance"` // balance reduction
}

func toJson(val interface{}) string {
	var b, err = json.Marshal(val)
	if err != nil {
		panic(err) // must not happen
	}
	return string(b)
}

func (rp *readPool) moveToBlobber(sscKey, allocID, blobID string,
	sp *stakePool, now common.Timestamp, value currency.Coin,
	balances cstate.StateContextI) (resp string, err error) {

	var cut = rp.blobberCut(allocID, blobID, now)

	if len(cut) == 0 {
		return "", fmt.Errorf("no tokens in read pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}

	// all redeems to response at the end
	var redeems []readPoolRedeem
	var moved currency.Coin = 0
	var torm []*allocationPool // to remove later (empty allocation pools)
	for _, ap := range cut {
		if value == moved {
			break // all required tokens has moved to the blobber
		}
		var bi, ok = ap.Blobbers.getIndex(blobID)
		if !ok {
			continue // impossible case, but leave the check here
		}
		var (
			bp   = ap.Blobbers[bi]
			move currency.Coin
		)
		if value >= bp.Balance {
			move, bp.Balance = bp.Balance, 0
		} else {
			move, bp.Balance = value, bp.Balance-value
		}

		ap.Balance -= currency.Coin(value)

		redeems = append(redeems, readPoolRedeem{
			PoolID:  ap.ID,
			Balance: move,
		})

		moved += move
		if bp.Balance == 0 {
			ap.Blobbers.removeByIndex(bi)
		}
		if ap.Balance == 0 {
			torm = append(torm, ap) // remove the allocation pool later
		}
	}

	if moved < value {
		return "", fmt.Errorf("not enough tokens in read pool for "+
			"allocation: %s, blobber: %s", allocID, blobID)
	}

	err = sp.DistributeRewards(value, blobID, spenum.Blobber, balances)
	if err != nil {
		return "", fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	// remove empty allocation pools
	rp.removeEmpty(allocID, torm)

	// return the read redeems for blobbers read pools cache
	return toJson(redeems), nil // ok
}

// take read pool by ID to unlock (the take is get and remove)
func (wp *readPool) take(poolID string, now common.Timestamp) (
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

func (rp *readPool) stat(now common.Timestamp) allocationPoolsStat {
	return rp.Pools.stat(now)
}

//
// smart contract methods
//

// getReadPool of current client
func (ssc *StorageSmartContract) getReadPool(clientID datastore.Key,
	balances cstate.StateContextI) (rp *readPool, err error) {

	rp = new(readPool)
	err = balances.GetTrieNode(readPoolKey(ssc.ID, clientID), rp)
	if err != nil {
		return nil, err
	}
	return rp, nil
}

// newReadPool SC function creates new read pool for a client.
func (ssc *StorageSmartContract) newReadPool(t *transaction.Transaction,
	_ []byte, balances cstate.StateContextI) (resp string, err error) {

	rp := new(readPool)
	err = balances.GetTrieNode(readPoolKey(ssc.ID, t.ClientID), rp)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	if err == nil {
		return "", common.NewError("new_read_pool_failed", "already exist")
	}

	rp = new(readPool)
	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	return string(rp.Encode()), nil
}

// lock tokens for read pool of transaction's client
func (ssc *StorageSmartContract) readPoolLock(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (resp string, err error) {

	// configs

	var conf *readPoolConfig
	if conf, err = ssc.getReadPoolConfig(balances, true); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	if len(lr.TargetId) == 0 {
		lr.TargetId = t.ClientID
	}

	var rp *readPool
	if rp, err = ssc.getReadPool(lr.TargetId, balances); err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
		rp = new(readPool)
	}

	if lr.AllocationID == "" {
		return "", common.NewError("read_pool_lock_failed",
			"missing allocation ID in request")
	}

	if t.Value < conf.MinLock {
		return "", common.NewError("read_pool_lock_failed",
			"insufficient amount to lock")
	}

	if lr.Duration < conf.MinLockPeriod {
		return "", common.NewError("read_pool_lock_failed",
			fmt.Sprintf("duration (%s) is shorter than min lock period (%s)",
				lr.Duration.String(), conf.MinLockPeriod.String()))
	}

	if lr.Duration > conf.MaxLockPeriod {
		return "", common.NewError("read_pool_lock_failed",
			fmt.Sprintf("duration (%s) is longer than max lock period (%v)",
				lr.Duration.String(), conf.MaxLockPeriod.String()))
	}

	// check client balance
	if !lr.MintTokens {
		if err = stakepool.CheckClientBalance(t, balances); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
	}

	// get the allocation object
	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(lr.AllocationID, balances)
	if err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get allocation: "+err.Error())
	}

	var bps blobberPools

	// lock for allocation -> blobber (particular blobber locking)
	if lr.BlobberID != "" {
		if _, ok := alloc.BlobberAllocsMap[lr.BlobberID]; !ok {
			return "", common.NewError("read_pool_lock_failed",
				fmt.Sprintf("no such blobber %s in allocation %s",
					lr.BlobberID, lr.AllocationID))
		}
		bps = append(bps, &blobberPool{
			Balance:   currency.Coin(t.Value),
			BlobberID: lr.BlobberID,
		})
	} else {
		// divide depending read price range for all blobbers of the
		// allocation
		var total float64 // total read price
		for _, b := range alloc.BlobberAllocs {
			total += float64(b.Terms.ReadPrice)
		}
		// calculate (divide)
		for _, b := range alloc.BlobberAllocs {
			var ratio = float64(b.Terms.ReadPrice) / total
			bps.add(&blobberPool{
				Balance:   currency.Coin(float64(t.Value) * ratio),
				BlobberID: b.BlobberID,
			})
		}
	}

	// create and dig allocation pool

	var ap allocationPool
	ap.AllocationID = lr.AllocationID
	ap.ExpireAt = t.CreationDate + toSeconds(lr.Duration)
	ap.Blobbers = bps

	if !lr.MintTokens {
		var transfer *state.Transfer
		if transfer, resp, err = ap.DigPool(t.Hash, t); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
		if err = balances.AddTransfer(transfer); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
	} else {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     currency.Coin(t.Value),
		}); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
		ap.Balance = currency.Coin(t.Value)
		ap.ID = t.Hash
	}

	// remembers who funded the read pool, so tokens get returned to funder on unlock
	if err := ssc.addToFundedPools(t.ClientID, ap.ID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	rp.Pools.add(&ap)
	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) readPoolUnlock(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (resp string, err error) {

	var (
		transfer *state.Transfer
		req      unlockRequest
	)

	if err = req.decode(input); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
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

	var rp *readPool
	if rp, err = ssc.getReadPool(req.PoolOwner, balances); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	var ap *allocationPool
	if ap, err = rp.take(req.PoolID, t.CreationDate); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	transfer, resp, err = ap.EmptyPool(ssc.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// save read pools
	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	return
}

//
// stat
//

// statistic for an allocation/blobber (used by blobbers)
func (ssc *StorageSmartContract) getReadPoolAllocBlobberStatHandler(
	ctx context.Context, params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID  = params.Get("client_id")
		allocID   = params.Get("allocation_id")
		blobberID = params.Get("blobber_id")
		rp        *readPool
	)

	if rp, err = ssc.getReadPool(clientID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get retrieving read pool")
	}

	var (
		cut  = rp.blobberCut(allocID, blobberID, common.Now())
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

// statistic for all locked tokens of the read pool
func (ssc *StorageSmartContract) getReadPoolStatHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = datastore.Key(params.Get("client_id"))
		rp       *readPool
	)

	if rp, err = ssc.getReadPool(clientID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get read pool")
	}

	return rp.stat(common.Now()), nil
}
