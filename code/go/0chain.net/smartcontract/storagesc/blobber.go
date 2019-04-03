package storagesc

import (
	"encoding/json"
	"sort"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getBlobbersList() ([]StorageNode, error) {
	var allBlobbersList = make([]StorageNode, 0)
	allBlobbersBytes, err := sc.DB.GetNode(ALL_BLOBBERS_KEY)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	if allBlobbersBytes == nil {
		return allBlobbersList, nil
	}
	err = json.Unmarshal(allBlobbersBytes, &allBlobbersList)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	sort.SliceStable(allBlobbersList, func(i, j int) bool {
		return allBlobbersList[i].ID < allBlobbersList[j].ID
	})
	return allBlobbersList, nil
}

func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction, input []byte) (string, error) {
	allBlobbersList, err := sc.getBlobbersList()
	if err != nil {
		return "", common.NewError("add_blobber_failed", "Failed to get blobber list"+err.Error())
	}
	var newBlobber StorageNode
	err = newBlobber.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey
	blobberBytes, _ := sc.DB.GetNode(newBlobber.GetKey())
	if blobberBytes == nil {
		allBlobbersList = append(allBlobbersList, newBlobber)
		allBlobbersBytes, _ := json.Marshal(allBlobbersList)
		sc.DB.PutNode(ALL_BLOBBERS_KEY, allBlobbersBytes)
		sc.DB.PutNode(newBlobber.GetKey(), newBlobber.Encode())
	}

	buff := newBlobber.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) commitBlobberRead(t *transaction.Transaction, input []byte) (string, error) {
	var commitRead ReadConnection
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	lastBlobberClientReadBytes, err := sc.DB.GetNode(commitRead.GetKey())
	if err != nil {
		return "", common.NewError("rm_read_error", "Error reading the read marker for the blobber and client")
	}
	lastCommittedRM := &ReadConnection{}
	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes)
	}

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Invalid read marker."+err.Error())
	}
	sc.DB.PutNode(commitRead.GetKey(), input)
	return "success", nil
}

func (sc *StorageSmartContract) commitBlobberConnection(t *transaction.Transaction, input []byte) (string, error) {
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
	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())

	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
	}

	err = allocationObj.Decode(allocationBytes)
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
	sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())

	blobberAllocationBytes, err = json.Marshal(blobberAllocation.LastWriteMarker)
	return string(blobberAllocationBytes), err
}
