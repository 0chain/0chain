package storagesc

import (
	"encoding/json"
	"sort"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getBlobbersList(balances c_state.StateContextI) (*StorageNodes, error) {
	allBlobbersList := &StorageNodes{}
	allBlobbersBytes, err := balances.GetTrieNode(ALL_BLOBBERS_KEY)
	if allBlobbersBytes == nil {
		return allBlobbersList, nil
	}
	err = json.Unmarshal(allBlobbersBytes.Encode(), allBlobbersList)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	sort.SliceStable(allBlobbersList.Nodes, func(i, j int) bool {
		return allBlobbersList.Nodes[i].ID < allBlobbersList.Nodes[j].ID
	})
	return allBlobbersList, nil
}

func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("add_blobber_failed", "Failed to get blobber list"+err.Error())
	}
	newBlobber := &StorageNode{}
	err = newBlobber.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey
	blobberBytes, _ := balances.GetTrieNode(newBlobber.GetKey(sc.ID))
	if blobberBytes == nil {
		allBlobbersList.Nodes = append(allBlobbersList.Nodes, newBlobber)
		// allBlobbersBytes, _ := json.Marshal(allBlobbersList)
		balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
		balances.InsertTrieNode(newBlobber.GetKey(sc.ID), newBlobber)
	}

	buff := newBlobber.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) commitBlobberRead(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	commitRead := &ReadConnection{}
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	lastBlobberClientReadBytes, err := balances.GetTrieNode(commitRead.GetKey(sc.ID))
	lastCommittedRM := &ReadConnection{}
	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes.Encode())
	}

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Invalid read marker."+err.Error())
	}
	balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	return "success", nil
}

func (sc *StorageSmartContract) commitBlobberConnection(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	var commitConnection BlobberCloseConnection
	err := json.Unmarshal(input, &commitConnection)
	if err != nil {
		return "", err
	}

	if !commitConnection.Verify() {
		return "", common.NewError("invalid_parameters", "Invalid input")
	}

	if commitConnection.WriteMarker.BlobberID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Invalid Blobber ID for closing connection. Write marker not for this blobber")
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = commitConnection.WriteMarker.AllocationID
	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))

	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
	}

	err = allocationObj.Decode(allocationBytes.Encode())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID. Failed to decode from DB")
	}

	if allocationObj.Owner != commitConnection.WriteMarker.ClientID {
		return "", common.NewError("invalid_parameters", "Write marker has to be by the same client as owner of the allocation")
	}

	blobberAllocation, ok := allocationObj.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation")
	}

	blobberAllocationBytes, err := json.Marshal(blobberAllocation)

	if !commitConnection.WriteMarker.VerifySignature(allocationObj.OwnerPublicKey) {
		return "", common.NewError("invalid_parameters", "Invalid signature for write marker")
	}

	if blobberAllocation.AllocationRoot == commitConnection.AllocationRoot && blobberAllocation.LastWriteMarker != nil && blobberAllocation.LastWriteMarker.PreviousAllocationRoot == commitConnection.PrevAllocationRoot {
		return string(blobberAllocationBytes), nil
	}

	if blobberAllocation.AllocationRoot != commitConnection.PrevAllocationRoot {
		return "", common.NewError("invalid_parameters", "Previous allocation root does not match the latest allocation root")
	}

	if blobberAllocation.Stats.UsedSize+commitConnection.WriteMarker.Size > blobberAllocation.Size {
		return "", common.NewError("invalid_parameters", "Size for blobber allocation exceeded maximum")
	}

	blobberAllocation.AllocationRoot = commitConnection.AllocationRoot
	blobberAllocation.LastWriteMarker = commitConnection.WriteMarker
	blobberAllocation.Stats.UsedSize += commitConnection.WriteMarker.Size
	blobberAllocation.Stats.NumWrites++

	allocationObj.Stats.UsedSize += commitConnection.WriteMarker.Size
	allocationObj.Stats.NumWrites++
	balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)

	blobberAllocationBytes, err = json.Marshal(blobberAllocation.LastWriteMarker)
	return string(blobberAllocationBytes), err
}
