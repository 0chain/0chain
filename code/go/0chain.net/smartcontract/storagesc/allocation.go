package storagesc

import (
	"encoding/json"
	"math/rand"
	"sort"
	"strconv"

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

func (sc *StorageSmartContract) newAllocationRequest(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
	}

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_creation_failed", "Invalid client in the transaction. No public key found")
	}

	var allocationRequest StorageAllocation

	err = allocationRequest.Decode(input)
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "Failed to create a storage allocation")
	}
	allocationRequest.Payer = t.ClientID
	if allocationRequest.Size > 0 && allocationRequest.DataShards > 0 && len(allocationRequest.OwnerPublicKey) > 0 && len(allocationRequest.Owner) > 0 && len(allocationRequest.Payer) > 0 {
		size := allocationRequest.DataShards + allocationRequest.ParityShards

		if len(allBlobbersList.Nodes) < size {
			return "", common.NewError("not_enough_blobbers", "Not enough blobbers to honor the allocation")
		}

		allocatedBlobbers := make([]*StorageNode, 0)
		allocationRequest.BlobberDetails = make([]*BlobberAllocation, 0)
		allocationRequest.Stats = &StorageAllocationStats{}

		var blobberNodes []*StorageNode
		preferredBlobbersSize := len(allocationRequest.PreferredBlobbers)
		if preferredBlobbersSize > 0 {
			blobberNodes, err = getPreferredBlobbers(allocationRequest.PreferredBlobbers, allBlobbersList.Nodes)
			if err != nil {
				return "", err
			}
		}
		if len(blobberNodes) < size {
			seed, err := strconv.ParseInt(t.Hash[0:8], 16, 64)
			if err != nil {
				return "", common.NewError("allocation_request_failed", "Failed to create seed for randomizeNodes")
			}

			// randomize blobber nodes
			blobberNodes = randomizeNodes(allBlobbersList.Nodes, blobberNodes, size, seed)
		}
		for i := 0; i < size; i++ {
			blobberNode := blobberNodes[i]
			var blobberAllocation BlobberAllocation
			blobberAllocation.Stats = &StorageAllocationStats{}
			blobberAllocation.Size = (allocationRequest.Size + int64(size-1)) / int64(size)
			blobberAllocation.AllocationID = t.Hash
			blobberAllocation.BlobberID = blobberNode.ID

			allocationRequest.BlobberDetails = append(allocationRequest.BlobberDetails, &blobberAllocation)
			allocatedBlobbers = append(allocatedBlobbers, blobberNode)
		}

		sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
			return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
		})

		allocationRequest.Blobbers = allocatedBlobbers
		allocationRequest.ID = t.Hash
		allocationRequest.Payer = t.ClientID

		buff, err := sc.addAllocation(&allocationRequest, balances)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the allocation request")
		}
		return buff, nil
	}
	return "", common.NewError("invalid_allocation_request", "Failed storage allocate")
}

func (sc *StorageSmartContract) updateAllocationRequest(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("allocation_updation_failed", "No Blobbers registered. Failed to update a storage allocation")
	}

	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewError("allocation_updation_failed", "No Blobbers registered. Failed to update a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_updation_failed", "Invalid client in the transaction. No public key found")
	}

	var updatedAllocationInput StorageAllocation

	err = updatedAllocationInput.Decode(input)
	if err != nil {
		return "", common.NewError("allocation_updation_failed", "Failed to update a storage allocation")
	}

	oldAllocations, err := sc.getAllocationsList(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("allocation_updation_failed", "Failed to find existing allocation")
	}

	oldAllocationExists := false
	oldAllocation := &StorageAllocation{}

	for _, oldAllocationID := range oldAllocations.List {
		if updatedAllocationInput.ID == oldAllocationID {
			oldAllocation.ID = oldAllocationID
			oldAllocationExists = true
			break
		}
	}

	if !oldAllocationExists {
		return "", common.NewError("allocation_updation_failed", "Failed to find existing allocation")
	}

	oldAllocationBytes, err := balances.GetTrieNode(oldAllocation.GetKey(sc.ID))
	if err != nil {
		return "", common.NewError("allocation_updation_failed", "Failed to find existing allocation")
	}

	oldAllocation.Decode(oldAllocationBytes.Encode())
	size := oldAllocation.DataShards + oldAllocation.ParityShards

	var updateSize int64
	if updatedAllocationInput.Size > 0 {
		updateSize = (updatedAllocationInput.Size + int64(size-1)) / int64(size)
	} else {
		updateSize = (updatedAllocationInput.Size - int64(size-1)) / int64(size)
	}

	for _, blobberAllocation := range oldAllocation.BlobberDetails {
		blobberAllocation.Size = blobberAllocation.Size + updateSize
	}

	oldAllocation.Size = oldAllocation.Size + updatedAllocationInput.Size
	oldAllocation.Expiration = oldAllocation.Expiration + updatedAllocationInput.Expiration
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
