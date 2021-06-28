package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

// getAllocation by ID
func (sc *StorageSmartContract) getAllocation(allocID string,
	balances chainstate.StateContextI) (alloc *StorageAllocation, err error) {

	alloc = new(StorageAllocation)
	alloc.ID = allocID
	var allocb util.Serializable
	if allocb, err = balances.GetTrieNode(alloc.GetKey(sc.ID)); err != nil {
		return nil, err
	}
	err = alloc.Decode(allocb.Encode())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return
}

func (sc *StorageSmartContract) getAllocationsList(clientID string,
	balances chainstate.StateContextI) (*Allocations, error) {

	allocationList := &Allocations{}
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	allocationListBytes, err := balances.GetTrieNode(clientAlloc.GetKey(sc.ID))
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), &clientAlloc)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, "failed to retrieve existing allocations list")
	}
	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) getAllAllocationsList(
	balances chainstate.StateContextI) (*Allocations, error) {

	allocationList := &Allocations{}

	allocationListBytes, err := balances.GetTrieNode(ALL_ALLOCATIONS_KEY)
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), allocationList)
	if err != nil {
		return nil, common.NewError("getAllAllocationsList_failed",
			"Failed to retrieve existing allocations list")
	}
	return allocationList, nil
}

func (sc *StorageSmartContract) addAllocation(alloc *StorageAllocation,
	balances chainstate.StateContextI) (string, error) {

	clients, err := sc.getAllocationsList(alloc.Owner, balances)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"Failed to get allocation list: %v", err)
	}
	all, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"Failed to get allocation list: %v", err)
	}

	if _, err = balances.GetTrieNode(alloc.GetKey(sc.ID)); err == nil {
		return "", common.NewErrorf("add_allocation_failed",
			"allocation id already used in trie: %v", alloc.GetKey(sc.ID))
	}
	if err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_allocation_failed",
			"unexpected error: %v", err)
	}

	clients.List.add(alloc.ID)
	all.List.add(alloc.ID)

	clientAllocation := &ClientAllocation{}
	clientAllocation.ClientID = alloc.Owner
	clientAllocation.Allocations = clients

	if _, err = balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, all); err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving all allocations list: %v", err)
	}

	_, err = balances.InsertTrieNode(clientAllocation.GetKey(sc.ID),
		clientAllocation)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving client allocations list (client: %s): %v",
			alloc.Owner, err)
	}
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving new allocation: %v", err)
	}

	buff := alloc.Encode()
	return string(buff), nil
}

type newAllocationRequest struct {
	DataShards                 int              `json:"data_shards"`
	ParityShards               int              `json:"parity_shards"`
	Size                       int64            `json:"size"`
	Expiration                 common.Timestamp `json:"expiration_date"`
	Owner                      string           `json:"owner_id"`
	OwnerPublicKey             string           `json:"owner_public_key"`
	PreferredBlobbers          []string         `json:"preferred_blobbers"`
	ReadPriceRange             PriceRange       `json:"read_price_range"`
	WritePriceRange            PriceRange       `json:"write_price_range"`
	MaxChallengeCompletionTime time.Duration    `json:"max_challenge_completion_time"`
}

// storageAllocation from the request
func (nar *newAllocationRequest) storageAllocation() (sa *StorageAllocation) {
	sa = new(StorageAllocation)
	sa.DataShards = nar.DataShards
	sa.ParityShards = nar.ParityShards
	sa.Size = nar.Size
	sa.Expiration = nar.Expiration
	sa.Owner = nar.Owner
	sa.OwnerPublicKey = nar.OwnerPublicKey
	sa.PreferredBlobbers = nar.PreferredBlobbers
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.MaxChallengeCompletionTime = nar.MaxChallengeCompletionTime
	return
}

func (nar *newAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

func (nar *newAllocationRequest) encode() ([]byte, error) {
	return json.Marshal(nar)
}

// (1) adjust blobber capacity used, (2) add offer (stake lock boundary),
// (3) save updated blobber
func (sc *StorageSmartContract) addBlobbersOffers(sa *StorageAllocation,
	blobbers []*StorageNode, balances chainstate.StateContextI) (err error) {

	// update blobbers' stakes and capacity used
	for i, b := range blobbers {
		b.Used += sa.BlobberDetails[i].Size // adjust used size
		var sp *stakePool
		if sp, err = sc.getStakePool(b.ID, balances); err != nil {
			return fmt.Errorf("can't get blobber's stake pool: %v", err)
		}
		sp.addOffer(sa, sa.BlobberDetails[i])

		// save blobber
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return fmt.Errorf("can't save blobber: %v", err)
		}
		// save its stake pool
		if err = sp.save(sc.ID, b.ID, balances); err != nil {
			return fmt.Errorf("can't save blobber's stake pool: %v", err)
		}
	}

	return
}

// update blobbers list in the all blobbers list
func updateBlobbersInAll(all *StorageNodes, update []*StorageNode,
	balances chainstate.StateContextI) (err error) {

	// update the blobbers in all blobbers list
	for _, b := range update {
		all.Nodes.update(b)
		// don't replace if blobber has removed from the all blobbers list;
		// for example, if the blobber has removed, then it shouldn't be
		// in the all blobbers list
	}

	// save
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, all)
	if err != nil {
		return fmt.Errorf("can't save all blobber list: %v", err)
	}

	return
}

// convert time.Duration to common.Timestamp truncating to seconds
func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

// size in gigabytes
func sizeInGB(size int64) float64 {
	return float64(size) / GB
}

// exclude blobbers with not enough token in stake pool to fit the size
func (sc *StorageSmartContract) filterBlobbersByFreeSpace(now common.Timestamp,
	size int64, balances chainstate.StateContextI) (filter filterBlobberFunc) {

	return filterBlobberFunc(func(b *StorageNode) (kick bool) {
		var sp, err = sc.getStakePool(b.ID, balances)
		if err != nil {
			return true // kick off
		}
		if b.Terms.WritePrice == 0 {
			return false // keep, ok or already filtered by bid
		}
		// clean capacity (without delegate pools want to 'unstake')
		var free = sp.cleanCapacity(now, b.Terms.WritePrice)
		return free < size // kick off if it hasn't enough free space
	})
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequest(
	t *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (string, error) {
	var conf *scConfig
	var err error
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"can't get config: %v", err)
	}

	resp, err := sc.newAllocationRequestInternal(t, input, conf, false, balances)
	if err != nil {
		return "", err
	}

	return resp, err
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequestInternal(
	t *transaction.Transaction,
	input []byte,
	conf *scConfig,
	mintNewTokens bool,
	balances chainstate.StateContextI,
) (resp string, err error) {
	var allBlobbersList *StorageNodes
	allBlobbersList, err = sc.getBlobbersList(balances)
	if err != nil || len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_creation_failed",
			"No Blobbers registered. Failed to create a storage allocation")
	}

	if t.ClientID == "" {
		return "", common.NewError("allocation_creation_failed",
			"Invalid client in the transaction. No client id in transaction")
	}

	var request newAllocationRequest
	if err = request.decode(input); err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"malformed request: %v", err)
	}

	var sa = request.storageAllocation() // (set fields, including expiration)
	sa.TimeUnit = conf.TimeUnit          // keep the initial time unit

	if err = sa.validate(t.CreationDate, conf); err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"invalid request: %v", err)
	}

	var (
		// number of blobbers required
		size = sa.DataShards + sa.ParityShards
		// size of allocation for a blobber
		bsize = (sa.Size + int64(size-1)) / int64(size)
		// filtered list
		list = sa.filterBlobbers(allBlobbersList.Nodes.copy(), t.CreationDate,
			bsize, filterHealthyBlobbers(t.CreationDate),
			sc.filterBlobbersByFreeSpace(t.CreationDate, bsize, balances))
	)

	if len(list) < size {
		return "", common.NewError("allocation_creation_failed",
			"Not enough blobbers to honor the allocation")
	}

	allocatedBlobbers := make([]*StorageNode, 0)
	sa.BlobberDetails = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	var blobberNodes []*StorageNode
	preferredBlobbersSize := len(sa.PreferredBlobbers)
	if preferredBlobbersSize > 0 {
		blobberNodes, err = getPreferredBlobbers(sa.PreferredBlobbers, list)
		if err != nil {
			return "", common.NewError("allocation_creation_failed",
				err.Error())
		}
	}

	// randomize blobber nodes
	if len(blobberNodes) < size {
		var seed int64
		if seed, err = strconv.ParseInt(t.Hash[0:8], 16, 64); err != nil {
			return "", common.NewError("allocation_creation_failed",
				"Failed to create seed for randomizeNodes")
		}
		blobberNodes = randomizeNodes(list, blobberNodes, size, seed)
	}

	blobberNodes = blobberNodes[:size]

	var gbSize = sizeInGB(bsize) // size in gigabytes

	for _, b := range blobberNodes {
		var balloc BlobberAllocation
		balloc.Stats = &StorageAllocationStats{}
		balloc.Size = bsize
		balloc.Terms = b.Terms
		balloc.AllocationID = t.Hash
		balloc.BlobberID = b.ID

		sa.BlobberDetails = append(sa.BlobberDetails, &balloc)
		allocatedBlobbers = append(allocatedBlobbers, b)

		// the Expiration and TimeUnit are already set for the 'sa' and we c
		// an use the restDurationInTimeUnits method here
		balloc.MinLockDemand = b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(t.CreationDate))

		if b.Terms.ChallengeCompletionTime > sa.ChallengeCompletionTime {
			sa.ChallengeCompletionTime = b.Terms.ChallengeCompletionTime
		}
	}

	sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
		return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
	})

	sa.Blobbers = allocatedBlobbers
	sa.ID = t.Hash
	sa.StartTime = t.CreationDate
	sa.Tx = t.Hash

	if err = sc.addBlobbersOffers(sa, blobberNodes, balances); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	err = updateBlobbersInAll(allBlobbersList, blobberNodes, balances)
	if err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	// create write pool and lock tokens
	if err = sc.createWritePool(t, sa, mintNewTokens, balances); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	if err = sc.createChallengePool(t, sa, balances); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	if resp, err = sc.addAllocation(sa, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "%v", err)
	}

	return resp, err
}

// update allocation request
type updateAllocationRequest struct {
	ID         string           `json:"id"`              // allocation id
	OwnerID    string           `json:"owner_id"`        // Owner of the allocation
	Size       int64            `json:"size"`            // difference
	Expiration common.Timestamp `json:"expiration_date"` // difference
}

func (uar *updateAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, uar)
}

// validate request
func (uar *updateAllocationRequest) validate(conf *scConfig,
	alloc *StorageAllocation) (err error) {

	if uar.Size == 0 && uar.Expiration == 0 {
		return errors.New("update allocation changes nothing")
	}

	if ns := alloc.Size + uar.Size; ns < conf.MinAllocSize {
		return fmt.Errorf("new allocation size is too small: %d < %d",
			ns, conf.MinAllocSize)
	}

	if len(alloc.BlobberDetails) == 0 {
		return errors.New("invalid allocation for updating: no blobbers")
	}

	return
}

// calculate size difference for every blobber of the allocations
func (uar *updateAllocationRequest) getBlobbersSizeDiff(
	alloc *StorageAllocation) (diff int64) {

	var size = alloc.DataShards + alloc.ParityShards
	if uar.Size > 0 {
		diff = (uar.Size + int64(size-1)) / int64(size)
	} else if uar.Size < 0 {
		diff = (uar.Size - int64(size-1)) / int64(size)
	}
	// else -> (0), no changes, avoid unnecessary calculation

	return
}

// new size of blobbers' allocation
func (uar *updateAllocationRequest) getNewBlobbersSize(
	alloc *StorageAllocation) (newSize int64) {

	return alloc.BlobberDetails[0].Size + uar.getBlobbersSizeDiff(alloc)
}

// getAllocationBlobbers loads blobbers of an allocation from store
func (sc *StorageSmartContract) getAllocationBlobbers(alloc *StorageAllocation,
	balances chainstate.StateContextI) (blobbers []*StorageNode, err error) {

	blobbers = make([]*StorageNode, 0, len(alloc.BlobberDetails))

	for _, details := range alloc.BlobberDetails {
		var blobber *StorageNode
		blobber, err = sc.getBlobber(details.BlobberID, balances)
		if err != nil {
			return nil, fmt.Errorf("can't get blobber %q: %v",
				details.BlobberID, err)
		}
		blobbers = append(blobbers, blobber)
	}

	return
}

// closeAllocation making it expired; the allocation will be alive the
// challenge_completion_time and be closed then
func (sc *StorageSmartContract) closeAllocation(t *transaction.Transaction,
	alloc *StorageAllocation, balances chainstate.StateContextI) (
	resp string, err error) {

	if alloc.Expiration-t.CreationDate <
		toSeconds(alloc.ChallengeCompletionTime) {
		return "", common.NewError("allocation_closing_failed",
			"doesn't need to close allocation is about to expire")
	}

	// mark as expired, but it will be alive at least chellenge_competion_time
	alloc.Expiration = t.CreationDate

	// stake pool (offers)

	for _, ba := range alloc.BlobberDetails {
		if err = sc.updateSakePoolOffer(ba, alloc, balances); err != nil {
			return "", common.NewError("allocation_closing_failed",
				err.Error())
		}
	}

	// save allocation

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("allocation_closing_failed",
			"can't save allocation: "+err.Error())
	}

	return string(alloc.Encode()), nil // closing
}

func (sc *StorageSmartContract) saveUpdatedAllocation(all *StorageNodes,
	alloc *StorageAllocation, blobbers []*StorageNode,
	balances chainstate.StateContextI) (err error) {

	// save all
	if err = updateBlobbersInAll(all, blobbers, balances); err != nil {
		return
	}

	// save related blobbers
	for _, b := range blobbers {
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return
		}
	}

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return
	}

	return
}

// allocation period used to calculate weighted average prices
type allocPeriod struct {
	read   state.Balance    // read price
	write  state.Balance    // write price
	period common.Timestamp // period (duration)
	size   int64            // size for period
}

func (ap *allocPeriod) weight() float64 {
	return float64(ap.period) * float64(ap.size)
}

// returns weighted average read and write prices
func (ap *allocPeriod) join(np *allocPeriod) (avgRead, avgWrite state.Balance) {
	var (
		apw, npw = ap.weight(), np.weight() // weights
		ws       = apw + npw                // weights sum
		rp, wp   float64                    // read sum, write sum (weighted)
	)

	rp = (float64(ap.read) * apw) + (float64(np.read) * npw)
	wp = (float64(ap.write) * apw) + (float64(np.write) * npw)

	avgRead = state.Balance(rp / ws)
	avgWrite = state.Balance(wp / ws)
	return
}

func weightedAverage(prev, next *Terms, tx, pexp, expDiff common.Timestamp,
	psize, sizeDiff int64) (avg Terms) {

	// allocation periods
	var left, added allocPeriod
	left.read, left.write = prev.ReadPrice, prev.WritePrice   // } prices
	added.read, added.write = next.ReadPrice, next.WritePrice // }
	left.size, added.size = psize, psize+sizeDiff             // sizes
	left.period, added.period = pexp-tx, pexp+expDiff-tx      // periods
	// join
	avg.ReadPrice, avg.WritePrice = left.join(&added)

	// just copy from next
	avg.MinLockDemand = next.MinLockDemand
	avg.MaxOfferDuration = next.MaxOfferDuration
	avg.ChallengeCompletionTime = next.ChallengeCompletionTime
	return
}

// The adjustChallengePool moves more or moves some tokens back from or to
// challenge pool during allocation extending or reducing.
func (sc *StorageSmartContract) adjustChallengePool(alloc *StorageAllocation,
	wp *writePool, odr, ndr common.Timestamp, oterms []Terms,
	now common.Timestamp, balances chainstate.StateContextI) (err error) {

	var (
		changes = alloc.challengePoolChanges(odr, ndr, oterms)
		cp      *challengePool
	)

	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	var changed bool

	for i, ch := range changes {
		var blobID = alloc.BlobberDetails[i].BlobberID
		switch {
		case ch > 0:
			err = wp.moveToChallenge(alloc.ID, blobID, cp, now, ch)
			changed = true
		case ch < 0:
			// only if the challenge pool has the tokens; all the tokens
			// can be moved back already, or moved to a blobber due to
			// challenge process
			if cp.Balance >= -ch {
				err = cp.moveToWritePool(alloc.ID, blobID, alloc.Until(), wp,
					-ch)
				changed = true
			}
		default:
			// no changes for the blobber
		}
		if err != nil {
			return fmt.Errorf("adjust_challenge_pool: %v", err)
		}
	}

	if changed {
		err = cp.save(sc.ID, alloc.ID, balances)
	}

	return
}

// extendAllocation extends size or/and expiration (one of them can be reduced);
// here we use new terms of blobbers
func (sc *StorageSmartContract) extendAllocation(
	t *transaction.Transaction,
	all *StorageNodes,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	uar *updateAllocationRequest,
	mintTokens bool,
	balances chainstate.StateContextI,
) (
	resp string, err error) {

	var (
		diff   = uar.getBlobbersSizeDiff(alloc) // size difference
		size   = uar.getNewBlobbersSize(alloc)  // blobber size
		gbSize = sizeInGB(size)                 // blobber size in GB
		cct    time.Duration                    // new challenge_completion_time

		// keep original terms to adjust challenge pool value
		oterms = make([]Terms, 0, len(alloc.BlobberDetails))
		// original allocation duration remains
		odr = alloc.Expiration - t.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	var prevExpiration = alloc.Expiration
	alloc.Expiration += uar.Expiration // new expiration
	alloc.Size += uar.Size             // new size

	// 1. update terms
	for i, details := range alloc.BlobberDetails {
		oterms = append(oterms, details.Terms) // keep original terms will be changed

		var b = blobbers[i]
		if b.Capacity == 0 {
			return "", common.NewErrorf("allocation_extending_failed",
				"blobber %s no longer provides its service", b.ID)
		}
		if uar.Size > 0 {
			if b.Capacity-b.Used-diff < 0 {
				return "", common.NewErrorf("allocation_extending_failed",
					"blobber %s doesn't have enough free space", b.ID)
			}
		}

		b.Used += diff // new capacity used

		// update terms using weighted average
		details.Terms = weightedAverage(&details.Terms, &b.Terms,
			t.CreationDate, prevExpiration, alloc.Expiration, details.Size,
			diff)

		details.Size = size // new size

		if uar.Expiration > toSeconds(b.Terms.MaxOfferDuration) {
			return "", common.NewErrorf("allocation_extending_failed",
				"blobber %s doesn't allow so long offers", b.ID)
		}

		if b.Terms.ChallengeCompletionTime > cct {
			cct = b.Terms.ChallengeCompletionTime // seek max CCT
		}

		// since, new terms is weighted average based on previous terms and
		// past allocation time and new terms and new allocation time; then
		// we can easily recalculate new min_lock_demand value from allocation
		// start to its new end using the new weighted average terms; but, we
		// can't reduce the min_lock_demand_value; that's all;

		// new blobber's min lock demand (alloc.Expiration is already updated
		// and we can use restDurationInTimeUnits method here)
		var nbmld = details.Terms.minLockDemand(gbSize,
			alloc.restDurationInTimeUnits(alloc.StartTime))

		// min_lock_demand can be increased only
		if nbmld > details.MinLockDemand {
			details.MinLockDemand = nbmld
		}
	}

	// update max challenge_completion_time
	alloc.ChallengeCompletionTime = cct

	// extend offers after alloc.challenge_completion_time is known
	for _, ba := range alloc.BlobberDetails {
		if err = sc.updateSakePoolOffer(ba, alloc, balances); err != nil {
			return "", common.NewError("allocation_extending_failed",
				err.Error())
		}
	}

	// get related write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return "", common.NewErrorf("allocation_extending_failed",
			"can't get write pool: %v", err)
	}

	var until = alloc.Until()

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if !mintTokens {
			if err = checkFill(t, balances); err != nil {
				return "", common.NewError("allocation_extending_failed",
					err.Error())
			}
		}
		if _, err = wp.fill(t, alloc, until, mintTokens, balances); err != nil {
			return "", common.NewErrorf("allocation_extending_failed",
				"write pool filling: %v", err)
		}
	}

	// is it about size increasing? if so, we should make sure the write
	// pool has enough tokens
	if diff > 0 {
		if mldLeft := alloc.restMinLockDemand(); mldLeft > 0 {
			if wp.allocUntil(alloc.ID, until) < mldLeft {
				return "", common.NewError("allocation_extending_failed",
					"not enough tokens in write pool to extend allocation")
			}
		}
	}

	// add more tokens to related challenge pool, or move some tokens back
	var ndr = alloc.Expiration - t.CreationDate
	err = sc.adjustChallengePool(alloc, wp, odr, ndr, oterms, t.CreationDate,
		balances)
	if err != nil {
		return "", common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	// save the write pool
	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return "", common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	// save all

	err = sc.saveUpdatedAllocation(all, alloc, blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	return string(alloc.Encode()), nil
}

// reduceAllocation reduces size or/and expiration (no one can be increased);
// here we use the same terms of related blobbers
func (sc *StorageSmartContract) reduceAllocation(t *transaction.Transaction,
	all *StorageNodes, alloc *StorageAllocation, blobbers []*StorageNode,
	uar *updateAllocationRequest, balances chainstate.StateContextI) (
	resp string, err error) {

	var (
		diff = uar.getBlobbersSizeDiff(alloc) // size difference
		size = uar.getNewBlobbersSize(alloc)  // blobber size

		// original allocation duration remains
		odr = alloc.Expiration - t.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	alloc.Expiration += uar.Expiration
	alloc.Size += uar.Size

	// 1. update terms
	for i, ba := range alloc.BlobberDetails {
		var b = blobbers[i]
		b.Used += diff // new capacity used
		ba.Size = size // new size
		// update stake pool
		if err = sc.updateSakePoolOffer(ba, alloc, balances); err != nil {
			return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
	}

	// get related write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return "", common.NewErrorf("allocation_reducing_failed",
			"can't get write pool: %v", err)
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = checkFill(t, balances); err != nil {
			return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
		var until = alloc.Until()
		if _, err = wp.fill(t, alloc, until, false, balances); err != nil {
			return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
	}

	// new allocation duration remains
	var ndr = alloc.Expiration - t.CreationDate
	err = sc.adjustChallengePool(alloc, wp, odr, ndr, nil, t.CreationDate,
		balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	// save the write pool
	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	// save all

	err = sc.saveUpdatedAllocation(all, alloc, blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	return string(alloc.Encode()), nil
}

// update allocation allows to change allocation size or expiration;
// if expiration reduced or unchanged, then existing terms of blobbers used,
// otherwise new terms used; also, it locks additional tokens if size is
// extended and it checks blobbers for required stake;
func (sc *StorageSmartContract) updateAllocationRequest(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get SC configurations: "+err.Error())
	}
	return sc.updateAllocationRequestInternal(txn, input, conf, false, balances)
}

func (sc *StorageSmartContract) updateAllocationRequestInternal(
	t *transaction.Transaction,
	input []byte,
	conf *scConfig,
	mintTokens bool,
	balances chainstate.StateContextI,
) (resp string, err error) {

	var all *StorageNodes // all blobbers list
	if all, err = sc.getBlobbersList(balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get all blobbers list: "+err.Error())
	}

	if len(all.Nodes) == 0 {
		return "", common.NewError("allocation_updating_failed",
			"empty blobbers list")
	}

	if t.ClientID == "" {
		return "", common.NewError("allocation_updating_failed",
			"missing client_id in transaction")
	}

	var request updateAllocationRequest
	if err = request.decode(input); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"invalid request: "+err.Error())
	}

	if request.OwnerID == "" {
		request.OwnerID = t.ClientID
		// return "", common.NewError("allocation_updating_failed",
		//	"invalid request: missing owner_id")
	}

	var clist *Allocations // client allocations list
	if clist, err = sc.getAllocationsList(request.OwnerID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get client's allocations list: "+err.Error())
	}

	if !clist.has(request.ID) {
		return "", common.NewErrorf("allocation_updating_failed",
			"can't find allocation in client's allocations list: %s (%d)",
			request.ID, len(clist.List))
	}

	var alloc *StorageAllocation
	if alloc, err = sc.getAllocation(request.ID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get existing allocation: "+err.Error())
	}

	if err = request.validate(conf, alloc); err != nil {
		return "", common.NewError("allocation_updating_failed", err.Error())
	}

	// can't update expired allocation
	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("allocation_updating_failed",
			"can't update expired allocation")
	}

	// get blobber of the allocation to update them
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			err.Error())
	}

	// adjust expiration
	var newExpiration = alloc.Expiration + request.Expiration

	// update allocation transaction hash
	alloc.Tx = t.Hash

	// close allocation now
	if newExpiration <= t.CreationDate {
		return sc.closeAllocation(t, alloc, balances)
	}

	// an allocation can't be shorter than configured in SC
	// (prevent allocation shortening for entire period)
	if request.Expiration < 0 &&
		newExpiration-t.CreationDate < toSeconds(conf.MinAllocDuration) {

		return "", common.NewError("allocation_updating_failed",
			"allocation duration becomes too short")
	}

	if request.Size < 0 && alloc.Size+request.Size < conf.MinAllocSize {
		return "", common.NewError("allocation_updating_failed",
			"allocation size becomes too small")
	}

	// if size or expiration increased, then we use new terms
	// otherwise, we use the same terms
	if request.Size > 0 || request.Expiration > 0 {
		return sc.extendAllocation(t, all, alloc, blobbers, &request, mintTokens, balances)
	}

	if mintTokens {
		return "", common.NewError("allocation_updating_failed",
			"cannot reduce when minting tokens")
	}

	return sc.reduceAllocation(t, all, alloc, blobbers, &request, balances)
}

func getPreferredBlobbers(preferredBlobbers []string, allBlobbers []*StorageNode) (selectedBlobbers []*StorageNode, err error) {
	blobberMap := make(map[string]*StorageNode)
	for _, storageNode := range allBlobbers {
		blobberMap[storageNode.BaseURL] = storageNode
	}
	for _, blobberURL := range preferredBlobbers {
		selectedBlobber, ok := blobberMap[blobberURL]
		if !ok {
			err = errors.New("invalid preferred blobber URL")
			return
		}
		selectedBlobbers = append(selectedBlobbers, selectedBlobber)
	}
	return
}

func randomizeNodes(in []*StorageNode, out []*StorageNode, n int, seed int64) []*StorageNode {
	nOut := minInt(len(in), n)
	nOut = maxInt(1, nOut)
	randGen := rand.New(rand.NewSource(seed))
	for {
		i := randGen.Intn(len(in))
		if !checkExists(in[i], out) {
			out = append(out, in[i])
		}
		if len(out) >= nOut {
			break
		}
	}
	return out
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func checkExists(c *StorageNode, sl []*StorageNode) bool {
	for _, s := range sl {
		if s.ID == c.ID {
			return true
		}
	}
	return false
}

func (sc *StorageSmartContract) finalizedPassRates(alloc *StorageAllocation) ([]float64, error) {
	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}
	var failed, succesful int64 = 0, 0
	var passRates = make([]float64, 0, len(alloc.BlobberDetails))
	for _, blobber := range alloc.BlobberDetails {
		if blobber.Stats == nil {
			blobber.Stats = new(StorageAllocationStats)
		}
		blobber.Stats.FailedChallenges += blobber.Stats.OpenChallenges
		blobber.Stats.OpenChallenges = 0
		blobber.Stats.TotalChallenges = blobber.Stats.FailedChallenges + blobber.Stats.SuccessChallenges
		if blobber.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		passRates = append(passRates, float64(blobber.Stats.SuccessChallenges)/float64(blobber.Stats.TotalChallenges))
		succesful += blobber.Stats.SuccessChallenges
		failed += blobber.Stats.FailedChallenges
	}
	alloc.Stats.SuccessChallenges = succesful
	alloc.Stats.FailedChallenges = failed
	alloc.Stats.TotalChallenges = alloc.Stats.FailedChallenges + alloc.Stats.FailedChallenges
	alloc.Stats.OpenChallenges = 0
	return passRates, nil
}

// a blobber can not send a challenge response, thus we have to check out
// challenge requests and their expiration
func (sc *StorageSmartContract) canceledPassRates(alloc *StorageAllocation,
	now common.Timestamp, balances chainstate.StateContextI) (
	passRates []float64, err error) {

	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}
	passRates = make([]float64, 0, len(alloc.BlobberDetails))
	var failed, succesful int64 = 0, 0
	// range over all related blobbers
	for _, d := range alloc.BlobberDetails {
		// check out blobber challenges
		var bc *BlobberChallenge
		bc, err = sc.getBlobberChallenge(d.BlobberID, balances)
		if err != nil && err != util.ErrValueNotPresent {
			return nil, fmt.Errorf("getting blobber challenge: %v", err)
		}
		// no blobber challenges, no failures
		if err == util.ErrValueNotPresent || len(bc.Challenges) == 0 {
			passRates, err = append(passRates, 1.0), nil
			continue // no challenges for the blobber
		}
		if d.Stats == nil {
			d.Stats = new(StorageAllocationStats) // make sure
		}
		// all expired open challenges are failed, all other
		// challenges we are treating as successful
		for _, c := range bc.Challenges {
			if c.Response != nil {
				continue // already accepted, already rewarded/penalized
			}
			var expire = c.Created + toSeconds(d.Terms.ChallengeCompletionTime)
			if expire < now {
				d.Stats.FailedChallenges++
			} else {
				d.Stats.SuccessChallenges++
			}
		}
		d.Stats.OpenChallenges = 0
		d.Stats.TotalChallenges = d.Stats.SuccessChallenges + d.Stats.FailedChallenges
		if d.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		// success rate for the blobber allocation
		//fmt.Println("pass rate i", i, "successful", d.Stats.SuccessChallenges, "failed", d.Stats.FailedChallenges)
		passRates = append(passRates, float64(d.Stats.SuccessChallenges)/float64(d.Stats.TotalChallenges))
		succesful += d.Stats.SuccessChallenges
		failed += d.Stats.FailedChallenges
	}
	alloc.Stats.SuccessChallenges = succesful
	alloc.Stats.FailedChallenges = failed
	alloc.Stats.TotalChallenges = alloc.Stats.FailedChallenges + alloc.Stats.FailedChallenges
	alloc.Stats.OpenChallenges = 0
	return passRates, nil
}

// If blobbers doesn't provide their services, then user can use this
// cancel_allocation transaction to close allocation and unlock all tokens
// of write pool back to himself. The cacnel_allocation doesn't pays min_lock
// demand to blobbers.
func (sc *StorageSmartContract) cancelAllocationRequest(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var req lockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}

	if alloc.Owner != t.ClientID {
		return "", common.NewError("alloc_cancel_failed",
			"only owner can cancel an allocation")
	}

	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("alloc_cancel_failed",
			"trying to cancel expired allocation")
	}

	var passRates []float64
	passRates, err = sc.canceledPassRates(alloc, t.CreationDate, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"can't get SC configurations: "+err.Error())
	}

	if fctc := conf.FailedChallengesToCancel; fctc > 0 {
		if alloc.Stats == nil || alloc.Stats.FailedChallenges < int64(fctc) {
			return "", common.NewError("alloc_cancel_failed",
				"not enough failed challenges of allocation to cancel")
		}
	}

	// can cancel
	// new values
	alloc.Expiration, alloc.ChallengeCompletionTime = t.CreationDate, 0

	var sps = []*stakePool{}
	for _, d := range alloc.BlobberDetails {
		var sp *stakePool
		if sp, err = sc.getStakePool(d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err = sp.extendOffer(alloc, d); err != nil {
			return "", common.NewError("alloc_cacnel_failed",
				"removing stake pool offer for "+d.BlobberID+": "+err.Error())
		}
		sps = append(sps, sp)
	}

	err = sc.finishAllocation(t, alloc, passRates, sps, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}

	alloc.Finalized, alloc.Canceled = true, true
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"saving allocation: "+err.Error())
	}

	return "canceled", nil
}

//
// finalize an allocation (after expire + challenge completion time)
//

// 1. challenge pool                  -> blobbers or write pool
// 2. write pool min_lock_demand left -> blobbers
// 3. remove offer from blobber (stake pool)
// 4. update blobbers used and in all blobbers list too
// 5. write pool                      -> client
func (sc *StorageSmartContract) finalizeAllocation(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var req lockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	// should be owner or one of blobbers of the allocation
	if !alloc.IsValidFinalizer(t.ClientID) {
		return "", common.NewError("fini_alloc_failed",
			"not allowed, unknown finalization initiator")
	}

	// should not be finalized
	if alloc.Finalized {
		return "", common.NewError("fini_alloc_failed",
			"allocation already finalized")
	}

	// should be expired
	if alloc.Until() > t.CreationDate {
		return "", common.NewError("fini_alloc_failed",
			"allocation is not expired yet, or waiting a challenge completion")
	}

	var passRates []float64
	passRates, err = sc.finalizedPassRates(alloc)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	var sps = []*stakePool{}
	for _, d := range alloc.BlobberDetails {
		var sp *stakePool
		if sp, err = sc.getStakePool(d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		sps = append(sps, sp)
	}

	err = sc.finishAllocation(t, alloc, passRates, sps, balances)
	if err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	alloc.Finalized = true
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"saving allocation: "+err.Error())
	}

	return "finalized", nil
}

func (sc *StorageSmartContract) finishAllocation(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	passRates []float64,
	sps []*stakePool,
	balances chainstate.StateContextI,
) (err error) {
	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get SC configurations: "+err.Error())
	}

	// write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get user's write pools: "+err.Error())
	}

	// challenge pool
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get related challenge pool: "+err.Error())
	}

	// blobbers
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"invalid state: can't get related blobbers: "+err.Error())
	}

	var allb *StorageNodes
	if allb, err = sc.getBlobbersList(balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get all blobbers list: "+err.Error())
	}

	// we can use the i for the blobbers list above because of algorithm
	// of the getAllocationBlobbers method; also, we can use the i in the
	// passRates list above because of algorithm of the adjustChallenges
	var cpLeft = cp.Balance // tokens left in related challenge pool
	for i, d := range alloc.BlobberDetails {
		// min lock demand rest
		var fctrml = conf.FailedChallengesToRevokeMinLock
		if d.Stats == nil || d.Stats.FailedChallenges < int64(fctrml) {
			if lack := d.MinLockDemand - d.Spent; lack > 0 {
				if _, err := moveReward(sc.ID, *cp.ZcnPool, sps[i], lack, balances); err != nil {
					return common.NewError("alloc_cancel_failed",
						"paying min_lock for "+d.BlobberID+": "+err.Error())
				}
				d.Spent += lack
				d.FinalReward += lack
				cpLeft -= lack
			}
		}
	}
	var passPayments state.Balance = 0
	for i, d := range alloc.BlobberDetails {
		var b = blobbers[i] // related blobber
		if alloc.UsedSize > 0 && cpLeft > 0 && passRates[i] > 0 && d.Stats != nil {
			var (
				ratio = float64(d.Stats.UsedSize) / float64(alloc.UsedSize)
				move  = state.Balance(float64(cpLeft) * ratio * passRates[i])
			)
			var reward state.Balance
			if reward, err = moveReward(sc.ID, *cp.ZcnPool, sps[i], move, balances); err != nil {
				return common.NewError("fini_alloc_failed",
					"moving tokens to stake pool of "+d.BlobberID+": "+
						err.Error())
			}
			sps[i].Rewards.Blobber += reward
			d.Spent += move
			d.FinalReward += move
			passPayments += move
		}
		var info *stakePoolUpdateInfo
		info, err = sps[i].update(conf, sc.ID, t.CreationDate, balances)
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"updating stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err = sps[i].save(sc.ID, d.BlobberID, balances); err != nil {
			return common.NewError("fini_alloc_failed",
				"saving stake pool of "+d.BlobberID+": "+err.Error())
		}
		conf.Minted += info.minted
		// update the blobber
		b.Used -= d.Size
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return common.NewError("fini_alloc_failed",
				"saving blobber "+d.BlobberID+": "+err.Error())
		}
		// update the blobber in all (replace with existing one)
		allb.Nodes.update(b)
	}
	cp.Balance = cpLeft - passPayments
	// move challenge pool rest to write pool
	alloc.MovedBack += cp.Balance
	err = cp.moveToWritePool(alloc.ID, "", alloc.Until(), wp, cp.Balance)
	if err != nil {
		return common.NewError("fini_alloc_failed",
			"moving challenge pool rest back to write pool: "+err.Error())
	}

	// save all blobbers list
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allb)
	if err != nil {
		return common.NewError("fini_alloc_failed",
			"saving all blobbers list: "+err.Error())
	}

	// save all rest and remove allocation from all allocations list

	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving challenge pool: "+err.Error())
	}

	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving write pool: "+err.Error())
	}

	alloc.Finalized = true

	var all *Allocations
	if all, err = sc.getAllAllocationsList(balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"getting all allocations list: "+err.Error())
	}

	if !all.List.remove(alloc.ID) {
		return common.NewError("fini_alloc_failed",
			"invalid state: allocation not found in all allocations list")
	}

	_, err = balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, all)
	if err != nil {
		return common.NewError("fini_alloc_failed",
			"saving all allocations list: "+err.Error())
	}

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(sc.ID), conf)
	if err != nil {
		return common.NewError("fini_alloc_failed",
			"saving configurations: "+err.Error())
	}

	return nil
}

type transferAllocationInput struct {
	AllocationId      string `json:"allocation_id"`
	NewOwnerId        string `json:"new_owner_id"`
	NewOwnerPublicKey string `json:"new_owner_public_key"`
}

func (aci *transferAllocationInput) decode(input []byte) error {
	return json.Unmarshal(input, aci)
}

func (sc *StorageSmartContract) curatorTransferAllocation(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (string, error) {
	var tai transferAllocationInput
	if err := tai.decode(input); err != nil {
		return "", common.NewError("curator_transfer_allocation_failed",
			"error unmarshalling input: "+err.Error())
	}

	alloc, err := sc.getAllocation(tai.AllocationId, balances)
	if err != nil {
		return "", common.NewError("curator_transfer_allocation_failed", err.Error())
	}

	if !alloc.isCurator(txn.ClientID) {
		return "", common.NewError("curator_transfer_allocation_failed",
			"only curators can transfer allocations; "+txn.ClientID+" is not a curator")
	}

	alloc.Owner = tai.NewOwnerId
	alloc.OwnerPublicKey = tai.NewOwnerPublicKey

	if !alloc.hasWritePool(sc, tai.NewOwnerId, balances) {
		if err = sc.createEmptyWritePool(txn, alloc, balances); err != nil {
			return "", common.NewError("curator_transfer_allocation_failed",
				"error creating write pool: "+err.Error())
		}
	}

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("curator_transfer_allocation_failed",
			"saving new allocation: %v", err)
	}

	// txn.Hash is the id of the new token pool
	return txn.Hash, nil
}

type addCuratorInput struct {
	CuratorId    string `json:"curator_id"`
	AllocationId string `json:"allocation_id"`
}

func (aci *addCuratorInput) decode(input []byte) error {
	return json.Unmarshal(input, aci)
}

func (sa StorageAllocation) isCurator(id string) bool {
	for _, curator := range sa.Curators {
		if curator == id {
			return true
		}
	}
	return false
}
func (sa StorageAllocation) hasWritePool(
	ssc *StorageSmartContract,
	id string,
	balances chainstate.StateContextI,
) bool {
	wp, err := ssc.getWritePool(sa.Owner, balances)
	if err != nil {
		return false
	}
	for _, pool := range wp.Pools {
		if pool.AllocationID == sa.ID {
			return true
		}
	}
	return false
}

func (sc *StorageSmartContract) addCurator(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (err error) {
	var aci addCuratorInput
	if err = aci.decode(input); err != nil {
		return common.NewError("add_curator_failed",
			"error unmarshalling input: "+err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(aci.AllocationId, balances)
	if err != nil {
		return common.NewError("alloc_cancel_failed", err.Error())
	}

	if alloc.Owner != txn.ClientID {
		return common.NewError("add_curator_failed",
			"only owner can add a curator")
	}

	if alloc.isCurator(aci.CuratorId) {
		return common.NewError("add_curator_failed",
			"already a curator: "+aci.CuratorId)
	}

	alloc.Curators = append(alloc.Curators, aci.CuratorId)

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return common.NewError("add_curator_failed",
			"cannot save allocation"+err.Error())
	}

	return nil
}
