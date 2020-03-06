package storagesc

import (
	"encoding/json"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getAllocationsList(clientID string, balances c_state.StateContextI) (*Allocations, error) {
	allocationList := &Allocations{}
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	allocationListBytes, err := balances.GetTrieNode(clientAlloc.GetKey(sc.ID))
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), &clientAlloc)
	if err != nil {
		return nil, common.NewError("getAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) getAllAllocationsList(balances c_state.StateContextI) (*Allocations, error) {
	allocationList := &Allocations{}

	allocationListBytes, err := balances.GetTrieNode(ALL_ALLOCATIONS_KEY)
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), allocationList)
	if err != nil {
		return nil, common.NewError("getAllAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	sort.SliceStable(allocationList.List, func(i, j int) bool {
		return allocationList.List[i] < allocationList.List[j]
	})
	return allocationList, nil
}

func (sc *StorageSmartContract) addAllocation(allocation *StorageAllocation, balances c_state.StateContextI) (string, error) {
	allocationList, err := sc.getAllocationsList(allocation.Owner, balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}
	allAllocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
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

func (sc *StorageSmartContract) newAllocationRequest(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("allocation_creation_failed",
			"No Blobbers registered. Failed to create a storage allocation")
	}

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_creation_failed",
			"No Blobbers registered. Failed to create a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_creation_failed",
			"Invalid client in the transaction. No public key found")
	}

	var req StorageAllocation
	if err = req.Decode(input); err != nil {
		return "", common.NewError("allocation_creation_failed",
			"failed to create a storage allocation")
	}

	req.Payer = t.ClientID
	if err = req.validate(); err != nil {
		return "", common.NewError("allocation_creation_failed",
			"invalid request: "+err.Error())
	}

	var (
		size = req.DataShards + req.ParityShards
		// size of allocation for a blobber
		bsize = (req.Size + int64(size-1)) / int64(size)
		// filtered list
		list = req.filterBlobbers(allBlobbersList.Nodes, bsize)
	)

	if len(list) < size {
		return "", common.NewError("not_enough_blobbers",
			"Not enough blobbers to honor the allocation")
	}

	allocatedBlobbers := make([]*StorageNode, 0)
	req.BlobberDetails = make([]*BlobberAllocation, 0)
	req.Stats = &StorageAllocationStats{}

	var blobberNodes []*StorageNode
	preferredBlobbersSize := len(req.PreferredBlobbers)
	if preferredBlobbersSize > 0 {
		blobberNodes, err = getPreferredBlobbers(req.PreferredBlobbers, list)
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

	var (
		demand                  int64                 // overall min lock demand
		gbSize                  = float64(bsize) / GB // size in gigabytes
		challengeCompletionTime time.Duration         // max
	)

	for i := 0; i < size; i++ {
		b := blobberNodes[i]
		var balloc BlobberAllocation
		balloc.Stats = &StorageAllocationStats{}
		balloc.Size = bsize
		balloc.Terms = b.Terms
		balloc.AllocationID = t.Hash
		balloc.BlobberID = b.ID

		req.BlobberDetails = append(req.BlobberDetails, &balloc)
		allocatedBlobbers = append(allocatedBlobbers, b)

		balloc.MinLockDemand = int64(math.Ceil(
			float64(b.Terms.WritePrice) * gbSize * b.Terms.MinLockDemand,
		))

		// add to overall min lock demand
		demand += balloc.MinLockDemand

		if b.Terms.ChallengeCompletionTime > challengeCompletionTime {
			challengeCompletionTime = b.Terms.ChallengeCompletionTime
		}

		// TODO (sfxdx): adjust blobbers' CapUsed
	}

	sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
		return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
	})

	req.Blobbers = allocatedBlobbers
	req.ChallengeCompletionTime = challengeCompletionTime
	req.MinLockDemand = demand
	req.ID = t.Hash

	// create related write_pool expires with the allocation + challenge
	// completion time
	wp, err := sc.newWritePool(req.GetKey(sc.ID), t.ClientID, t.CreationDate,
		req.Expiration+
			common.Timestamp(challengeCompletionTime.Truncate(time.Second)),
		balances)
	if err != nil {
		return "", common.NewError("allocation_request_failed",
			"can't create write pool: "+err.Error())
	}

	// lock tokens if user provides them
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

	// TODO (sfxdx):
	//   1. create challenge pool
	//   2. move the min lock demand to the challenge pool?

	buff, err := sc.addAllocation(&req, balances)
	if err != nil {
		return "", common.NewError("allocation_request_failed",
			"failed to store the allocation request")
	}

	return buff, nil
}

func (sc *StorageSmartContract) updateAllocationRequest(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("allocation_updation_failed",
			"No Blobbers registered. Failed to update a storage allocation")
	}

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_updation_failed",
			"No Blobbers registered. Failed to update a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_updation_failed",
			"Invalid client in the transaction. No public key found")
	}

	var req StorageAllocation
	if err = req.Decode(input); err != nil {
		return "", common.NewError("allocation_updation_failed",
			"Failed to update a storage allocation")
	}

	oldAllocations, err := sc.getAllocationsList(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("allocation_updation_failed",
			"Failed to find existing allocation")
	}

	oldAllocationExists := false
	oldAllocation := &StorageAllocation{}

	for _, oldAllocationID := range oldAllocations.List {
		if req.ID == oldAllocationID {
			oldAllocation.ID = oldAllocationID
			oldAllocationExists = true
			break
		}
	}

	if !oldAllocationExists {
		return "", common.NewError("allocation_updation_failed",
			"Failed to find existing allocation")
	}

	oldAllocationBytes, err := balances.GetTrieNode(oldAllocation.GetKey(sc.ID))
	if err != nil {
		return "", common.NewError("allocation_updation_failed",
			"Failed to find existing allocation")
	}

	oldAllocation.Decode(oldAllocationBytes.Encode())
	size := oldAllocation.DataShards + oldAllocation.ParityShards

	var updateSize int64
	if req.Size > 0 {
		updateSize = (req.Size + int64(size-1)) / int64(size)
	} else {
		updateSize = (req.Size - int64(size-1)) / int64(size)
	}

	for _, blobberAllocation := range oldAllocation.BlobberDetails {
		blobberAllocation.Size = blobberAllocation.Size + updateSize
	}

	oldAllocation.Size = oldAllocation.Size + req.Size
	oldAllocation.Expiration = oldAllocation.Expiration + req.Expiration
	_, err = balances.InsertTrieNode(oldAllocation.GetKey(sc.ID), oldAllocation)
	if err != nil {
		return "", common.NewError("allocation_updation_failed", "Failed to update existing allocation")
	}

	buff := oldAllocation.Encode()
	return string(buff), nil
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
