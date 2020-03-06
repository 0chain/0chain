package storagesc

import (
	"encoding/json"
	"sort"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
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

func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("add_blobber_failed",
			"Failed to get blobber list: "+err.Error())
	}
	var newBlobber StorageNode
	if err = newBlobber.Decode(input); err != nil {
		return "", common.NewError("add_blobber_failed",
			"malformed request: "+err.Error())
	}

	if err = newBlobber.validate(); err != nil {
		return "", common.NewError("add_blobber_failed",
			"invalid values in request: "+err.Error())
	}

	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey
	blobberBytes, err := balances.GetTrieNode(newBlobber.GetKey(sc.ID))

	// errors handling
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("add_blobber_failed", err.Error())
	}

	// already have
	if err == nil {
		return string(blobberBytes.Encode()), nil
	}

	// create new (util.ErrValueNotPresent)

	allBlobbersList.Nodes = append(allBlobbersList.Nodes, &newBlobber)
	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	balances.InsertTrieNode(newBlobber.GetKey(sc.ID), &newBlobber)

	return string(newBlobber.Encode()), nil
}

//
// TODO (sfxdx): remove this, use addBlobber to update a blobber
//
// updateBlobber terms and capacity
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("update_blobber_failed",
			"failed to get blobber list: "+err.Error())
	}

	var updateBlobber StorageNode
	if err = updateBlobber.Decode(input); err != nil {
		return "", common.NewError("update_blobber_failed",
			"malformed request: "+err.Error())
	}

	if err = updateBlobber.validate(); err != nil {
		return "", common.NewError("update_blobber_failed",
			"invalid values in request: "+err.Error())
	}

	updateBlobber.ID = t.ClientID
	updateBlobber.PublicKey = t.PublicKey
	_, err = balances.GetTrieNode(updateBlobber.GetKey(sc.ID))

	// errors handling
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("update_blobber_failed", err.Error())
	}

	// already have
	if err == util.ErrValueNotPresent {
		return "", common.NewError("update_blobber_failed", "no such blobber")
	}

	// update existing blobber

	var found bool
	for i, b := range allBlobbersList.Nodes {
		if b.ID == t.ClientID {
			allBlobbersList.Nodes[i], found = &updateBlobber, true
			break
		}
	}

	if !found {
		// invalid DB state (impossible case?)
		return "", common.NewError("update_blobber_failed",
			"blobber not found in all blobbers list")
	}

	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	balances.InsertTrieNode(updateBlobber.GetKey(sc.ID), &updateBlobber)

	return string(updateBlobber.Encode()), nil
}

func (sc *StorageSmartContract) commitBlobberRead(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {

	// TODO (sfxdx): move token from readMarker.OwnerID read pool to the blobber

	commitRead := &ReadConnection{}
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	lastBlobberClientReadBytes, err := balances.GetTrieNode(commitRead.GetKey(sc.ID))
	lastCommittedRM := &ReadConnection{}
	lastKnownCtr := int64(0)
	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes.Encode())
		lastKnownCtr = lastCommittedRM.ReadMarker.ReadCounter
	}

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Invalid read marker."+err.Error())
	}
	balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	sc.newRead(balances, commitRead.ReadMarker.ReadCounter-lastKnownCtr)
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
	sc.newWrite(balances, commitConnection.WriteMarker.Size)
	return string(blobberAllocationBytes), err
}
