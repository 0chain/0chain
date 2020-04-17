package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

// getAllocation by ID
func (sc *StorageSmartContract) getAllocation(allocID string,
	balances c_state.StateContextI) (alloc *StorageAllocation, err error) {

	alloc = new(StorageAllocation)
	alloc.ID = allocID
	var allocb util.Serializable
	if allocb, err = balances.GetTrieNode(alloc.GetKey(sc.ID)); err != nil {
		return nil, err
	}
	err = alloc.Decode(allocb.Encode())
	return
}

func (sc *StorageSmartContract) getAllocationsList(clientID string,
	balances c_state.StateContextI) (*Allocations, error) {

	allocationList := &Allocations{}
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	allocationListBytes, err := balances.GetTrieNode(clientAlloc.GetKey(sc.ID))
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), &clientAlloc)
	if err != nil {
		return nil, common.NewError("getAllocationsList_failed",
			"Failed to retrieve existing allocations list")
	}
	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) getAllAllocationsList(
	balances c_state.StateContextI) (*Allocations, error) {

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
	balances c_state.StateContextI) (string, error) {

	clients, err := sc.getAllocationsList(alloc.Owner, balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed",
			"Failed to get allocation list"+err.Error())
	}
	all, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed",
			"Failed to get allocation list"+err.Error())
	}

	if _, err = balances.GetTrieNode(alloc.GetKey(sc.ID)); err == nil {
		return "", common.NewError("add_allocation_failed",
			"allocation id already used in trie: "+alloc.GetKey(sc.ID))
	}
	if err != util.ErrValueNotPresent {
		return "", common.NewError("add_allocation_failed",
			"unexpected error: "+err.Error())
	}

	clients.List.add(alloc.ID)
	all.List.add(alloc.ID)

	clientAllocation := &ClientAllocation{}
	clientAllocation.ClientID = alloc.Owner
	clientAllocation.Allocations = clients

	balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, all)
	balances.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
	balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)

	buff := alloc.Encode()
	return string(buff), nil
}

type newAllocationRequest struct {
	DataShards        int              `json:"data_shards"`
	ParityShards      int              `json:"parity_shards"`
	Size              int64            `json:"size"`
	Expiration        common.Timestamp `json:"expiration_date"`
	Owner             string           `json:"owner_id"`
	OwnerPublicKey    string           `json:"owner_public_key"`
	PreferredBlobbers []string         `json:"preferred_blobbers"`
	ReadPriceRange    PriceRange       `json:"read_price_range"`
	WritePriceRange   PriceRange       `json:"write_price_range"`
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
	return
}

func (nar *newAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

// (1) adjust blobber capacity used, (2) add offer (stake lock boundary),
// (3) save updated blobber
func (sc *StorageSmartContract) addBlobbersOffers(sa *StorageAllocation,
	blobbers []*StorageNode, balances c_state.StateContextI) (err error) {

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
	balances c_state.StateContextI) (err error) {

	// update the blobbers in all blobbers list
	for _, b := range update {
		var i, ok = all.Nodes.getIndex(b.ID)
		if ok {
			all.Nodes[i] = b // replace only it found
		}
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

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequest(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("allocation_creation_failed",
			"can't get config: "+err.Error())
	}

	allBlobbersList, err := sc.getBlobbersList(balances)
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
		return "", common.NewError("allocation_creation_failed",
			"malformed request: "+err.Error())
	}

	var sa = request.storageAllocation()
	sa.Owner = t.ClientID
	if err = sa.validate(t.CreationDate, conf); err != nil {
		return "", common.NewError("allocation_creation_failed",
			"invalid request: "+err.Error())
	}

	var (
		// number of blobbers required
		size = sa.DataShards + sa.ParityShards
		// size of allocation for a blobber
		bsize = (sa.Size + int64(size-1)) / int64(size)
		// filtered list
		list = sa.filterBlobbers(allBlobbersList.Nodes, t.CreationDate, bsize,
			filterHealthyBlobbers(t.CreationDate))
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
			return "", err
		}
	}

	// randomize blobber nodes
	if len(blobberNodes) < size {
		seed, err := strconv.ParseInt(t.Hash[0:8], 16, 64)
		if err != nil {
			return "", common.NewError("allocation_request_failed",
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

		balloc.MinLockDemand = state.Balance(
			float64(b.Terms.WritePrice) * gbSize * b.Terms.MinLockDemand,
		)

		if b.Terms.ChallengeCompletionTime > sa.ChallengeCompletionTime {
			sa.ChallengeCompletionTime = b.Terms.ChallengeCompletionTime
		}
	}

	sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
		return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
	})

	sa.Blobbers = allocatedBlobbers
	sa.ID = t.Hash
	sa.StartTime = t.CreationDate // offer start time
	sa.Tx = t.Hash                // keep

	if err = sc.addBlobbersOffers(sa, blobberNodes, balances); err != nil {
		return "", common.NewError("allocation_request_failed", err.Error())
	}

	err = updateBlobbersInAll(allBlobbersList, blobberNodes, balances)
	if err != nil {
		return "", common.NewError("allocation_request_failed", err.Error())
	}

	// create write pool and lock tokens
	if err = sc.createWritePool(t, sa, balances); err != nil {
		return "", common.NewError("allocation_request_failed", err.Error())
	}

	// create challenge pool
	if err = sc.createChallengePool(t, sa, balances); err != nil {
		return "", common.NewError("allocation_request_failed", err.Error())
	}

	// save
	buff, err := sc.addAllocation(sa, balances)
	if err != nil {
		return "", common.NewError("allocation_request_failed",
			"failed to store the allocation request")
	}

	return buff, nil
}

// update allocation request
type updateAllocationRequest struct {
	ID         string           `json:"id"`         // allocation id
	Size       int64            `json:"size"`       // difference
	Expiration common.Timestamp `json:"expiration"` // difference
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
	balances c_state.StateContextI) (blobbers []*StorageNode, err error) {

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
	alloc *StorageAllocation, balances c_state.StateContextI) (
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
	balances c_state.StateContextI) (err error) {

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

// extendAllocation extends size or/and expiration (one of them can be reduced);
// here we use new terms of blobbers
func (sc *StorageSmartContract) extendAllocation(t *transaction.Transaction,
	all *StorageNodes, alloc *StorageAllocation, blobbers []*StorageNode,
	uar *updateAllocationRequest, balances c_state.StateContextI) (resp string,
	err error) {

	var (
		diff   = uar.getBlobbersSizeDiff(alloc) // size difference
		size   = uar.getNewBlobbersSize(alloc)  // blobber size
		gbSize = sizeInGB(size)                 // blobber size in GB
		cct    time.Duration                    // new challenge_completion_time
	)

	// adjust the expiration if changed, boundaries has already checked
	var prevExpiration = alloc.Expiration
	alloc.Expiration += uar.Expiration // new expiration
	alloc.Size += uar.Size             // new size

	// 1. update terms
	for i, ba := range alloc.BlobberDetails {
		var b = blobbers[i]
		if b.Capacity == 0 {
			return "", common.NewError("allocation_extending_failed",
				"blobber "+b.ID+" no longer provides its service")
		}
		if uar.Size > 0 {
			if b.Capacity-b.Used-diff < 0 {
				return "", common.NewError("allocation_extending_failed",
					"blobber "+b.ID+" doesn't have enough free space")
			}
		}

		b.Used += diff // new capacity used

		// update terms using weighted average
		ba.Terms = weightedAverage(&ba.Terms, &b.Terms, t.CreationDate,
			prevExpiration, alloc.Expiration, ba.Size, diff)

		ba.Size = size // new size

		if uar.Expiration > toSeconds(b.Terms.MaxOfferDuration) {
			return "", common.NewError("allocation_extending_failed",
				"blobber "+b.ID+" doesn't allow too long offers")
		}

		if b.Terms.ChallengeCompletionTime > cct {
			cct = b.Terms.ChallengeCompletionTime
		}
		// new blobber's min lock demand
		var nbmld = state.Balance(math.Ceil(
			float64(ba.Terms.WritePrice) * gbSize * ba.Terms.MinLockDemand,
		))
		// min_lock_demand can be increased only
		if nbmld > ba.MinLockDemand {
			ba.MinLockDemand = nbmld
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
	if wp, err = sc.getWritePool(alloc.ID, balances); err != nil {
		return "", common.NewError("allocation_extending_failed",
			"can't get write pool: "+err.Error())
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = sc.checkFill(t, balances); err != nil {
			return "", common.NewError("allocation_extending_failed",
				err.Error())
		}
		if _, _, err = wp.fill(t, balances); err != nil {
			return "", common.NewError("allocation_extending_failed",
				"write pool filling: "+err.Error())
		}
	}

	// is it about size increasing? if so, we should make sure the write
	// pool has enough tokens
	if diff > 0 {
		if mldLeft := alloc.minLockDemandLeft(); mldLeft > 0 {
			if wp.Balance < mldLeft {
				return "", common.NewError("allocation_extending_failed",
					"not enough tokens in write pool to extend allocation")
			}
		}
	}

	// save the write pool
	if err = wp.save(sc.ID, alloc.ID, balances); err != nil {
		return "", common.NewError("allocation_extending_failed",
			err.Error())
	}

	// save all

	err = sc.saveUpdatedAllocation(all, alloc, blobbers, balances)
	if err != nil {
		return "", common.NewError("allocation_extending_failed",
			err.Error())
	}

	return string(alloc.Encode()), nil
}

// reduceAllocation reduces size or/and expiration (no one can be increased);
// here we use the same terms of related blobbers
func (sc *StorageSmartContract) reduceAllocation(t *transaction.Transaction,
	all *StorageNodes, alloc *StorageAllocation, blobbers []*StorageNode,
	uar *updateAllocationRequest, balances c_state.StateContextI) (
	resp string, err error) {

	var (
		diff = uar.getBlobbersSizeDiff(alloc) // size difference
		size = uar.getNewBlobbersSize(alloc)  // blobber size
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
			return "", common.NewError("allocation_reducing_failed",
				err.Error())
		}
	}

	// get related write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.ID, balances); err != nil {
		return "", common.NewError("allocation_reducing_failed",
			"can't get write pool: "+err.Error())
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = sc.checkFill(t, balances); err != nil {
			return "", common.NewError("allocation_reducing_failed",
				err.Error())
		}
		if _, _, err = wp.fill(t, balances); err != nil {
			return "", common.NewError("allocation_reducing_failed",
				err.Error())
		}
	}

	// save the write pool
	if err = wp.save(sc.ID, alloc.ID, balances); err != nil {
		return "", common.NewError("allocation_reducing_failed",
			err.Error())
	}

	// save all

	err = sc.saveUpdatedAllocation(all, alloc, blobbers, balances)
	if err != nil {
		return "", common.NewError("allocation_reducing_failed",
			err.Error())
	}

	return string(alloc.Encode()), nil
}

// update allocation allows to change allocation size or expiration;
// if expiration reduced or unchanged, then existing terms of blobbers used,
// otherwise new terms used; also, it locks additional tokens if size is
// extended and it checks blobbers for required stake;
func (sc *StorageSmartContract) updateAllocationRequest(
	t *transaction.Transaction, input []byte, balances c_state.StateContextI) (
	resp string, err error) {

	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get SC configurations: "+err.Error())
	}

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

	var clist *Allocations // client allocations list
	if clist, err = sc.getAllocationsList(t.ClientID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get client's allocations list: "+err.Error())
	}

	if !clist.has(request.ID) {
		return "", common.NewError("allocation_updating_failed",
			"can't find allocation in client's allocations list")
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

	// an allocation can't be shorter then configured in SC
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
		return sc.extendAllocation(t, all, alloc, blobbers, &request, balances)
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
			err = common.NewError("allocation_request_failed", "Invalid preferred blobber URL")
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

// If blobbers doesn't provide their services, then user can use this
// cancel_allocation transaction to close allocation and unlock all tokens
// of write pool back to himself. The cacnel_allocation doesn't pays min_lock
// demand to blobbers.
func (sc *StorageSmartContract) cacnelAllocationRequest(
	t *transaction.Transaction, input []byte, balances c_state.StateContextI) (
	resp string, err error) {

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("alloc_cacnel_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed", err.Error())
	}

	if alloc.Owner != t.ClientID {
		return "", common.NewError("alloc_cacnel_failed",
			"only owner can cancel an allocation")
	}

	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("alloc_cacnel_failed",
			"allocation is expired or going to expire soon")
	}

	alloc.Expiration = t.CreationDate // now
	alloc.ChallengeCompletionTime = 0 // no challenges wait
	for _, details := range alloc.BlobberDetails {
		details.MinLockDemand = 0 // reset
	}

	alloc.Cancelled = true

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"can't save allocation: "+err.Error())
	}

	return sc.finalizeAllocation(t, input, balances)
}
