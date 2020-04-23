package storagesc

import (
	"sort"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

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
	AllocationID datastore.Key `json:"allocation_id"`
	BlobberID    datastore.Key `json:"blobber_id,omitempty"`
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
	PoolID datastore.Key `json:"pool_id"`
}

func (ur *unlockRequest) decode(input []byte) error {
	return json.Unmarshal(input, ur)
}

//
// blobber read/write pool (expire_at at level above)
//

// blobber pool represents tokens locked for a blobber
type blobberPool struct {
	BlobberID datastore.Key `json:"blobber_id"`
	Balance   state.Balance `json:"balance"`
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
	ExpireAt          common.Timestamp `json:"expire_at"`
	AllocationID      datastore.Key    `json:"allocation_id"`
	Blobbers          blobberPools     `json:"blobbers"`
}

//
// allocation read/write pools (list)
//

// allocationPools is sorted list of read/write pools of allocations
type allocationPools []*allocationPool

func (aps allocationPools) getIndex(allocID string) (i int, ok bool) {
	i = sort.Search(len(aps), func(i int) bool {
		return aps[i].AllocationID >= allocID
	})
	if i == len(aps) {
		return // not found
	}
	if aps[i].AllocationID == allocID {
		return i, true // found
	}
	return // not found
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

func (aps *allocationPools) removeByIndex(i int) {
	(*aps) = append((*aps)[:i], (*aps)[i+1:]...)
}

func (aps *allocationPools) remove(allocID string) (ok bool) {
	var i int
	if i, ok = aps.getIndex(allocID); !ok {
		return // false
	}
	aps.removeByIndex(i)
	return true // removed
}

func (aps *allocationPools) add(ap *allocationPool) {
	if len(*aps) == 0 {
		(*aps) = append((*aps), ap)
		return
	}
	var i = sort.Search(len(*aps), func(i int) bool {
		return (*aps)[i].AllocationID >= ap.AllocationID
	})
	// out of bounds
	if i == len(*aps) {
		(*aps) = append((*aps), ap)
		return
	}
	// insert next after the found one
	(*aps) = append((*aps)[:i], append(allocationPools{ap},
		(*aps)[i:]...)...)
	return
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
	value state.Balance) {

	var cut = aps.allocationCut(allocID)
	cut = removeExpired(cut, until)
	for _, ap := range cut {
		value += ap.Balance
	}
	return
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
	(*aps) = (*aps)[:i]
}

func removeExpired(cut []*allocationPool, now common.Timestamp) (
	clean []*allocationPool) {

	var i int
	for _, arp := range cut {
		if arp.ExpireAt <= now {
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
		if arp.ExpireAt <= now {
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
	BlobberID datastore.Key `json:"blobber_id"`
	Balance   state.Balance `json:"balance"`
}

func (bp *blobberPool) stat() (stat blobberPoolStat) {
	stat.Balance = bp.Balance
	stat.BlobberID = bp.BlobberID
	return
}

// allocation read/write pool represents tokens locked for an allocation;
type allocationPoolStat struct {
	ID           string            `json:"id"`
	Balance      state.Balance     `json:"balance"`
	ExpireAt     common.Timestamp  `json:"expire_at"`
	AllocationID datastore.Key     `json:"allocation_id"`
	Blobbers     []blobberPoolStat `json:"blobbers"`
	Locked       bool              `json:"locked"`
}

func (ap *allocationPool) stat(now common.Timestamp) (stat allocationPoolStat) {

	stat.ID = ap.ID
	stat.Balance = ap.Balance
	stat.ExpireAt = ap.ExpireAt
	stat.AllocationID = ap.AllocationID
	stat.Locked = ap.ExpireAt > now

	stat.Blobbers = make([]blobberPoolStat, 0, len(ap.Blobbers))
	for _, bp := range ap.Blobbers {
		stat.Blobbers = append(stat.Blobbers, bp.stat())
	}

	return
}

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

type untilStat struct {
	PoolID   datastore.Key    `json:"pool_id"`
	Balance  state.Balance    `json:"balance"`
	ExpireAt common.Timestamp `json:"expire_at"`
}
