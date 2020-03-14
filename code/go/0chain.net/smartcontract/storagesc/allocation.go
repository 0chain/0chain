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
	sort.SliceStable(allocationList.List, func(i, j int) bool {
		return allocationList.List[i] < allocationList.List[j]
	})
	return allocationList, nil
}

func (sc *StorageSmartContract) addAllocation(allocation *StorageAllocation,
	balances c_state.StateContextI) (string, error) {

	allocationList, err := sc.getAllocationsList(allocation.Owner, balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed",
			"Failed to get allocation list"+err.Error())
	}
	allAllocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed",
			"Failed to get allocation list"+err.Error())
	}

	allocationBytes, _ := balances.GetTrieNode(allocation.GetKey(sc.ID))
	if allocationBytes == nil {
		allocationList.List = append(allocationList.List, allocation.ID)
		allAllocationList.List = append(allAllocationList.List, allocation.ID)
		clientAllocation := &ClientAllocation{}
		clientAllocation.ClientID = allocation.Owner
		clientAllocation.Allocations = allocationList

		// allAllocationBytes, _ := json.Marshal(allAllocationList)
		balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, allAllocationList)
		balances.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
		balances.InsertTrieNode(allocation.GetKey(sc.ID), allocation)
	}

	buff := allocation.Encode()
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
// (3) save updated blobber, (4) (todo) create blobber's challenge pool
func (sc *StorageSmartContract) addBlobbersOffers(sa *StorageAllocation,
	blobbers []*StorageNodes, balances c_state.StateContextI) (err error) {

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

		// TODO (sfxdx): create challenge pool for the blobber/allocation
	}

	return
}

// update blobbers list in the all blobbers list
func updateBlobbersInAll(all *StorageNodes, update []*StorageNode,
	balances c_state.StateContextI) (err error) {

	// update the blobbers in all blobbers list
	for i, ab := range all.Nodes {
		for j, b := range update {
			if ab.ID == b.ID {
				all.Nodes[i] = b                            // update
				update = append(update[:i], update[i+1]...) // kick
				break
			}
		}
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

// create, fill and save write pool for new allocation
func (sc *StorageSmartContract) createWritePool(t *transaction.Transaction,
	sa *StorageAllocation, balances c_state.StateContextI) (err error) {

	// create related write_pool expires with the allocation + challenge
	// completion time
	var wp *writePool
	wp, err = sc.newWritePool(sa.GetKey(sc.ID), t.ClientID, t.CreationDate,
		sa.Expiration+toSeconds(sa.ChallengeCompletionTime), balances)
	if err != nil {
		return fmt.Errorf("can't create write pool: %v", err)
	}

	// lock required number of tokens

	if t.Value < sa.MinLockDemand {
		return fmt.Errorf("not enough tokens to create allocation: %v < %v",
			t.Value, sa.MinLockDemand)
	}

	if _, _, err = wp.fill(t, balances); err != nil {
		return fmt.Errorf("can't fill write pool: %v", err)
	}

	// save the write pool
	if err = wp.save(sc.ID, sa.ID, balances); err != nil {
		return fmt.Errorf("can't save write pool: %v", err)
	}

	return
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
	if err != nil {
		return "", common.NewError("allocation_creation_failed",
			"No Blobbers registered. Failed to create a storage allocation")
	}

	allBlobbersList = sc.filterHealthyBlobbers(allBlobbersList)

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_creation_failed",
			"No Blobbers registered. Failed to create a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_creation_failed",
			"Invalid client in the transaction. No public key found")
	}

	var request newAllocationRequest
	if err = request.decode(input); err != nil {
		return "", common.NewError("allocation_creation_failed",
			"failed to create a storage allocation")
	}

	var sa = request.storageAllocation()
	sa.Payer = t.ClientID
	if err = sa.validate(conf); err != nil {
		return "", common.NewError("allocation_creation_failed",
			"invalid request: "+err.Error())
	}

	var (
		// number of blobbers required
		size = sa.DataShards + sa.ParityShards
		// size of allocation for a blobber
		bsize = (sa.Size + int64(size-1)) / int64(size)
		// filtered list
		list = sa.filterBlobbers(allBlobbersList.Nodes, t.CreationDate, bsize)
	)

	if len(list) < size {
		return "", common.NewError("not_enough_blobbers",
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

		balloc.MinLockDemand = int64(math.Ceil(
			float64(b.Terms.WritePrice) * gbSize * b.Terms.MinLockDemand,
		))

		// add to overall min lock demand
		sa.MinLockDemand += balloc.MinLockDemand

		if b.Terms.ChallengeCompletionTime > sa.ChallengeCompletionTime {
			sa.ChallengeCompletionTime = b.Terms.ChallengeCompletionTime
		}
	}

	sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
		return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
	})

	// TODO (sfxdx): why it saves blobbers in allocation?
	sa.Blobbers = allocatedBlobbers
	sa.ID = t.Hash
	sa.StartTime = t.CreationDate // offer start time

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

	buff, err := sc.addAllocation(sa, balances)
	if err != nil {
		return "", common.NewError("allocation_request_failed",
			"failed to store the allocation request")
	}

	return buff, nil
}

// update allocation request
type updateAllocationRequest struct {
	ID         string           `json:"id"`              // allocation id
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
	balances c_state.StateContextI) (blobbers []*StorageNode, err error) {

	blobbers = make([]*StorageNode, 0, len(alloc.BlobberDetails))

	for _, details := range alloc.BlobberDetails {
		var blobber *StorageNode
		blobber, err = sc.getBlobber(details.BlobberID, balances)
		if err != nil {
			return fmt.Errorf("can't get blobber %q: %v", details.BlobberID,
				err)
		}
		blobbers = append(blobbers, blobber)
	}

	return
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

	if !clist.find(request.ID) {
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

	// change terms to new, if the update allocation request
	// increases size or expiration time

	var (
		bsize  = request.getNewBlobbersSize(alloc) // new blobber size
		gbSize = sizeInGB(bsize)                   // in GB

		// new values

		newChallengeCompletionTime time.Duration
		newMinLockDemand           int64
	)

	// check blobbers and calculate new values
	for i, ba := range alloc.BlobberDetails {
		// related blobber with probably new terms
		var nb = blobbers[i]
		// blobber has been removed, can't extend the allocation
		if nb.Capacity == 0 {
			return "", common.NewError("allocation_updating_failed",
				fmt.Sprintf("blobber %s no longer provides its service", nb.ID))
		}
		// extend expiration time
		if request.Expiration > 0 {
			// can't extend, blobber doesn't accept too long offer
			if alloc.Expiration+request.Expiration >
				t.CreationDate+toSeconds(nb.Terms.MaxOfferDuration) {

				return "", common.NewError("allocation_updating_failed",
					fmt.Sprintf("blobber %s doesn't allow too long offer",
						nb.ID))
			}
		}
		// change terms
		if request.Expiration > 0 || request.Size > 0 {
			ba.Terms = nb.Terms // new terms
			// get new value only if terms has changed
			if ba.Terms.ChallengeCompletionTime > newChallengeCompletionTime {
				newChallengeCompletionTime = ba.Terms.ChallengeCompletionTime
			}
			// we don't reduce the min lock demand, even if size has reduced
			// check required free space in the blobber
			if nb.Capacity-nb.Used < bsize {
				return "", common.NewError("allocation_updating_failed",
					fmt.Sprintf("blobber %s doesn't have enough space", nb.ID))
			}
			ba.MinLockDemand = int64(math.Ceil(
				float64(b.Terms.WritePrice) * gbSize * b.Terms.MinLockDemand,
			))
			// add to overall min lock demand
			newMinLockDemand += ba.MinLockDemand
		}

	}

	// determine actual challenge completion time
	if newChallengeCompletionTime == 0 {
		newChallengeCompletionTime = alloc.ChallengeCompletionTime // the same
	}

	// set new values
	alloc.ChallengeCompletionTime = newChallengeCompletionTime
	alloc.Expiration = alloc.Expiration + request.Expiration

	// update blobbers locks (stake pool & capacity used)
	for i, ba := range alloc.BlobberDetails {
		var nb = blobbers[i]
		// adjust blobber's capacity used
		if request.Size != 0 {
			nb.Used += bsize - ba.Size // += (new size - old size)
		}
		// update blobbers' related offer
		var sp *stakePool
		if sp, err = sc.getStakePool(nb.ID, balances); err != nil {
			return "", common.NewError("allocation_updating_failed",
				fmt.Sprintf("can't get blobber's %s stake pool", nb.ID))
		}
		if err = sp.extendOffer(alloc, ba); err != nil {
			return "", common.NewError("allocation_updating_failed",
				err.Error())
		}
		if err = sp.save(sc.ID, nb.ID, balances); err != nil {
			return "", common.NewError("allocation_updating_failed",
				fmt.Sprintf("can't save blobber's %s stake pool: %v", nb.ID,
					err))
		}
	}

	// do we need to lock more tokens to extend the allocation?
	var lack int64
	if newMinLockDemand > alloc.MinLockDemand {
		lack = newMinLockDemand - alloc.MinLockDemand
		alloc.MinLockDemand = newMinLockDemand // use new value
	}

	// expiration difference
	var expireDiff = toSeconds(newChallengeCompletionTime) + request.Expiration

	// - if the min lock demand has increased, then we need to make sure related
	//   write pool have enough tokens (of fill it by this transactions)
	// - if the expiration has changed, then we need to adjust lock of the
	//   related write pool
	// - add tokens to write pool if the transaction provides them
	if lack > 0 || request.Expiration != 0 || t.Value > 0 {
		var wp *writePool
		if wp, err = sc.getWritePool(alloc.ID, balances); err != nil {
			return "", common.NewError("allocation_updating_failed",
				"can't get related write pool: "+err.Error())
		}
		if err = wp.extend(expireDiff); err != nil {
			return "", common.NewError("allocation_updating_failed",
				"can't extend related write pool: "+err.Error())
		}
		if t.Value > 0 {
			if _, _, err = wp.fill(t, balances); err != nil {
				return "", common.NewError("allocation_updating_failed",
					"can't add tokens to related write pool: "+err.Error())
			}
		}
		if lack > 0 {
			if wp.Balance < state.Balance(lack) {
				return "", common.NewError("allocation_updating_failed",
					fmt.Sprintf("not enough tokens to extend allocation: "+
						"%v < %v", wp.Balance, newMinLockDemand))
			}
		}
	}

	return
}

// udpateAllocation with the same blobbers increasing or reducing
// size or expiration; if terms of a blobber has changed, then the terms
// will be moved to the allocation only if the request increases
// expiration of the allocation
func (sc *StorageSmartContract) updateAllocationRequestStub(
	t *transaction.Transaction, input []byte, balances c_state.StateContextI) (
	string, error) {

	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get all blobbers list: "+err.Error())
	}

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_updating_failed",
			"all blobbers list is empty")
	}

	if t.ClientID == "" {
		return "", common.NewError("allocation_updating_failed",
			"missing client id in transaction")
	}

	var req StorageAllocation
	if err = req.Decode(input); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"Failed to update a storage allocation")
	}

	oldAllocations, err := sc.getAllocationsList(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("allocation_updating_failed",
			"Failed to find existing allocation")
	}

	var (
		found bool
		alloc StorageAllocation
	)

	for _, id := range oldAllocations.List {
		if req.ID == id {
			alloc.ID, found = id, true
			break
		}
	}

	if !found {
		return "", common.NewError("allocation_updating_failed",
			"Failed to find existing allocation")
	}

	allocBytes, err := balances.GetTrieNode(alloc.GetKey(sc.ID))
	if err != nil {
		return "", common.NewError("allocation_updating_failed",
			"Failed to find existing allocation")
	}

	if err = alloc.Decode(allocBytes.Encode()); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"malformed db response, can't decode allocation: "+err.Error())
	}
	var (
		size       = alloc.DataShards + alloc.ParityShards
		updateSize int64
	)
	if req.Size > 0 {
		updateSize = (req.Size + int64(size-1)) / int64(size)
	} else {
		updateSize = (req.Size - int64(size-1)) / int64(size)
	}

	// min 'max offer duration' of blobbers
	var (
		offerDuration time.Duration              // min offer duration limit
		demand        int64                      // additional lock demand
		gbUpdateSize  = float64(updateSize) / GB // in GB
	)

	for _, b := range alloc.BlobberDetails {
		if offerDuration == 0 || offerDuration > b.Terms.MaxOfferDuration {
			offerDuration = b.Terms.MaxOfferDuration
		}
		b.Size = b.Size + updateSize

		// blobber demand difference (+/-)
		var bdemand = int64(math.Ceil(
			float64(b.Terms.WritePrice) * gbUpdateSize * b.Terms.MinLockDemand,
		))

		b.MinLockDemand += bdemand // add or sub

		// add to overall demand
		demand += demand
	}

	// can we extend the allocation by the offer duration?
	var dur = common.ToTime(alloc.Expiration + req.Expiration).
		Sub(common.ToTime(alloc.StartTime))

	if offerDuration < dur {
		return "", common.NewError("allocation_updating_failed",
			"can't extend allocation time due to offer expiration")
	}

	// TODO (sfxdx): check out blobbers' stake to not exceed it

	// extend
	alloc.Size = alloc.Size + req.Size
	alloc.Expiration = alloc.Expiration + req.Expiration

	// TODO (sfxdx): adjust blobbers Used (used capacity)

	// write pool:
	//     1. lock additional tokens if it extends size (?)
	//     2. extend write pool expiration if it extends the expiration

	wp, err := sc.getWritePool(alloc.ID, balances)
	if err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get related write pool: "+err.Error())
	}

	if err = wp.extend(req.Expiration); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't change write pool lock duration: "+err.Error())
	}

	// lock additional tokens
	if demand > 0 {
		// TODO (sfxdx): should the user have enough tokens here?
	}

	// lock some tokens if user provides them
	if t.Value > 0 {
		if _, _, err = wp.fill(t, balances); err != nil {
			return "", common.NewError("write_pool_lock_failed",
				"can't lock tokens in write pool: "+err.Error())
		}
	}

	// save the write pool
	if err = wp.save(sc.ID, req.ID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't save write pool: "+err.Error())
	}

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), &alloc)
	if err != nil {
		return "", common.NewError("allocation_updating_failed",
			"Failed to update existing allocation")
	}

	return string(alloc.Encode()), nil
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
