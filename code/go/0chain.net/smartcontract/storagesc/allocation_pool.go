package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"0chain.net/chaincore/currency"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
)

//msgp:ignore lockRequest unlockRequest
//go:generate msgp -io=false -tests=false -unexported=true -v

//
// SC / API requests
//

// lock request

// request to lock tokens creating a read pool;
// the allocation_id is required, if blobber_id provided, then
// it locks tokens for allocation -> {blobber}, otherwise
// all tokens divided for all blobbers of the allocation
// automatically
type lockRequest struct {
	Duration     time.Duration `json:"duration"`
	AllocationID string        `json:"allocation_id"`
	BlobberID    string        `json:"blobber_id,omitempty"`
	TargetId     string        `json:"target_id,omitempty"`
	MintTokens   bool          `json:"mint_tokens,omitempty"`
}

func (lr *lockRequest) decode(input []byte) (err error) {
	if err = json.Unmarshal(input, lr); err != nil {
		return
	}
	if lr.AllocationID == "" {
		return errors.New("missing allocation_id in request")
	}
	return // ok
}

// unlock request used to unlock all tokens of a read pool
type unlockRequest struct {
	PoolOwner string `json:"pool_owner,omitempty"`
	PoolID    string `json:"pool_id"`
}

func (ur *unlockRequest) decode(input []byte) error {
	return json.Unmarshal(input, ur)
}

//
// blobber read/write pool (expire_at at level above)
//

// blobber pool represents tokens locked for a blobber
type blobberPool struct {
	BlobberID string        `json:"blobber_id"`
	Balance   currency.Coin `json:"balance"`
}

//
// blobber read/write pools (list)
//

// blobberPools is sorted list of blobber read/write pools sorted by blobber ID
type blobberPools []*blobberPool

func (bps blobberPools) getIndex(blobberID string) (i int, ok bool) {
	i = sort.Search(len(bps), func(i int) bool {
		return bps[i].BlobberID >= blobberID
	})
	if i == len(bps) {
		return // not found
	}
	if bps[i].BlobberID == blobberID {
		return i, true // found
	}
	return // not found
}

func (bps blobberPools) get(blobberID string) (
	bp *blobberPool, ok bool) {

	var i = sort.Search(len(bps), func(i int) bool {
		return bps[i].BlobberID >= blobberID
	})
	if i == len(bps) {
		return // not found
	}
	if bps[i].BlobberID == blobberID {
		return bps[i], true // found
	}
	return // not found
}

func (bps *blobberPools) removeByIndex(i int) {
	(*bps) = append((*bps)[:i], (*bps)[i+1:]...)
}

func (bps *blobberPools) remove(blobberID string) (ok bool) {
	var i int
	if i, ok = bps.getIndex(blobberID); !ok {
		return // false
	}
	bps.removeByIndex(i)
	return true // removed
}

func (bps *blobberPools) add(bp *blobberPool) (ok bool) {
	if len(*bps) == 0 {
		(*bps) = append((*bps), bp)
		return true // added
	}
	var i = sort.Search(len(*bps), func(i int) bool {
		return (*bps)[i].BlobberID >= bp.BlobberID
	})
	// out of bounds
	if i == len(*bps) {
		(*bps) = append((*bps), bp)
		return true // added
	}
	// the same
	if (*bps)[i].BlobberID == bp.BlobberID {
		(*bps)[i] = bp // replace
		return false   // already have
	}
	// next
	(*bps) = append((*bps)[:i], append([]*blobberPool{bp},
		(*bps)[i:]...)...)
	return true // added
}

//
// allocation read/write pool
//

// allocation read/write pool represents tokens locked for an allocation;
type allocationPool struct {
	tokenpool.ZcnPool `json:"pool"`
	ExpireAt          common.Timestamp `json:"expire_at"`     // inclusive
	AllocationID      string           `json:"allocation_id"` //
	Blobbers          blobberPools     `json:"blobbers"`      //
}

func newAllocationPool(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	until common.Timestamp,
	mintNewTokens bool,
	balances chainState.StateContextI,
) (*allocationPool, error) {
	var err error
	if !mintNewTokens {
		if err = stakepool.CheckClientBalance(t, balances); err != nil {
			return nil, err
		}
	}

	var ap allocationPool
	var transfer *state.Transfer
	if transfer, _, err = ap.DigPool(t.Hash, t); err != nil {
		return nil, fmt.Errorf("digging write pool: %v", err)
	}
	if mintNewTokens {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     t.Value,
		}); err != nil {
			return nil, fmt.Errorf("minting tokens for write pool: %v", err)
		}
	} else {
		if err = balances.AddTransfer(transfer); err != nil {
			return nil, fmt.Errorf("adding transfer to write pool: %v", err)
		}
	}

	// set fields
	ap.AllocationID = alloc.ID
	ap.ExpireAt = until
	ap.Blobbers, err = makeCopyAllocationBlobbers(*alloc, t.Value)
	if err != nil {
		return nil, fmt.Errorf("error creating blobber pools: %v", err)
	}

	// add the allocation pool
	alloc.addWritePoolOwner(alloc.Owner)
	return &ap, nil
}

//
// allocation read/write pools (list)
//

// allocationPools is sorted list of read/write pools of allocations
type allocationPools []*allocationPool

func (aps allocationPools) getIndex(allocID string) (i int, ok bool) {
	var ap *allocationPool
	for i, ap = range aps {
		if ap.AllocationID == allocID {
			return i, true
		}
	}
	return 0, false
}

func (aps allocationPools) get(allocID string) (
	ap *allocationPool, ok bool) {

	var i = sort.Search(len(aps), func(i int) bool {
		return aps[i].AllocationID >= allocID
	})
	if i == len(aps) {
		return // not found
	}
	if aps[i].AllocationID == allocID {
		return aps[i], true // found
	}
	return // not found
}

func (aps *allocationPools) add(ap *allocationPool) {
	if len(*aps) == 0 {
		*aps = append(*aps, ap)
		return
	}
	var i = sort.Search(len(*aps), func(i int) bool {
		return (*aps)[i].AllocationID >= ap.AllocationID
	})
	// out of bounds
	if i == len(*aps) {
		*aps = append(*aps, ap)
		return
	}
	// insert next after the found one
	*aps = append((*aps)[:i], append(allocationPools{ap},
		(*aps)[i:]...)...)
}

func (aps allocationPools) allocationCut(allocID string) (
	cut []*allocationPool) {

	var i, ok = aps.getIndex(allocID)
	if !ok {
		return // nil
	}

	var j = i + 1
	for ; j < len(aps) && aps[j].AllocationID == allocID; j++ {
	}

	if len(aps[i:j]) == 0 {
		return // nil
	}

	cut = make([]*allocationPool, len(aps[i:j]))
	copy(cut, aps[i:j])
	return
}

func (aps allocationPools) blobberCut(allocID, blobberID string,
	now common.Timestamp) (cut []*allocationPool) {

	cut = aps.allocationCut(allocID)
	cut = removeBlobberExpired(cut, blobberID, now)
	sortExpireAt(cut)
	return
}

func (aps allocationPools) allocUntil(allocID string, until common.Timestamp) (
	value currency.Coin) {

	var cut = aps.allocationCut(allocID)
	cut = removeExpired(cut, until)
	for _, ap := range cut {
		value += ap.Balance //810
	}
	return
}

func (aps allocationPools) sortExpiry() {
	sort.Slice(aps, func(i, j int) bool {
		return aps[i].ExpireAt < aps[j].ExpireAt
	})
}

func isInTOMRList(torm []*allocationPool, ax *allocationPool) bool {
	for _, tr := range torm {
		if tr == ax {
			return true
		}
	}
	return false
}

// remove empty pools of an allocation (all given pools should belongs to
// one allocation)
func (aps *allocationPools) removeEmpty(allocID string,
	torm []*allocationPool) {

	if len(torm) == 0 {
		return // nothing to remove
	}

	var i, ok = aps.getIndex(allocID)
	if !ok {
		return // not found, impossible case, but keep it here
	}
Outer:
	for _, ax := range (*aps)[i:] {
		if ax.AllocationID == allocID {
			if isInTOMRList(torm, ax) {
				continue Outer
			}
		}
		(*aps)[i], i = ax, i+1
	}
	*aps = (*aps)[:i]
}

func (aps *allocationPools) moveToChallenge(
	allocID, blobID string,
	cp *challengePool,
	now common.Timestamp,
	value currency.Coin,
) (err error) {
	if value == 0 {
		return // nothing to move, ok
	}

	var cut = aps.blobberCut(allocID, blobID, now)

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
			move currency.Coin
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
	aps.removeEmpty(allocID, torm)
	return
}

func removeExpired(cut []*allocationPool, now common.Timestamp) (
	clean []*allocationPool) {

	var i int
	for _, arp := range cut {
		if arp.ExpireAt < now {
			continue
		}
		if arp.Balance == 0 {
			continue // no tokens for this blobber
		}
		cut[i], i = arp, i+1
	}
	return cut[:i]
}

func removeBlobberExpired(cut []*allocationPool, blobberID string,
	now common.Timestamp) (clean []*allocationPool) {

	var i int
	for _, arp := range cut {
		if arp.ExpireAt < now {
			continue
		}
		var bp, ok = arp.Blobbers.get(blobberID)
		if !ok {
			continue // no pool for this blobber
		}
		if bp.Balance == 0 {
			continue // no tokens for this blobber
		}
		cut[i], i = arp, i+1
	}
	return cut[:i]
}

func sortExpireAt(cut []*allocationPool) {
	sort.Slice(cut, func(i, j int) bool {
		return cut[i].ExpireAt < cut[j].ExpireAt
	})
}

//
// stat
//

// blobber pool represents tokens locked for a blobber
type blobberPoolStat struct {
	BlobberID string        `json:"blobber_id"`
	Balance   currency.Coin `json:"balance"`
}

func (bp *blobberPool) stat() (stat blobberPoolStat) {
	stat.Balance = bp.Balance
	stat.BlobberID = bp.BlobberID
	return
}

// allocation read/write pool represents tokens locked for an allocation;
type allocationPoolStat struct {
	ID           string            `json:"id"`
	Balance      currency.Coin     `json:"balance"`
	ExpireAt     common.Timestamp  `json:"expire_at"`
	AllocationID string            `json:"allocation_id"`
	Blobbers     []blobberPoolStat `json:"blobbers"`
	Locked       bool              `json:"locked"`
}

func (ap *allocationPool) stat(now common.Timestamp) (stat allocationPoolStat) {

	stat.ID = ap.ID
	stat.Balance = ap.Balance
	stat.ExpireAt = ap.ExpireAt
	stat.AllocationID = ap.AllocationID
	stat.Locked = ap.ExpireAt >= now

	stat.Blobbers = make([]blobberPoolStat, 0, len(ap.Blobbers))
	for _, bp := range ap.Blobbers {
		stat.Blobbers = append(stat.Blobbers, bp.stat())
	}

	return
}

// swagger:model allocationPoolsStat
type allocationPoolsStat struct {
	Pools []allocationPoolStat `json:"pools"`
}

func (aps allocationPools) stat(now common.Timestamp) (
	stat allocationPoolsStat) {

	stat.Pools = make([]allocationPoolStat, 0, len(aps))
	for _, ap := range aps {
		stat.Pools = append(stat.Pools, ap.stat(now))
	}
	return
}

//
// until stat
//
// swagger:model untilStat
type untilStat struct {
	PoolID   string           `json:"pool_id"`
	Balance  currency.Coin    `json:"balance"`
	ExpireAt common.Timestamp `json:"expire_at"`
}
