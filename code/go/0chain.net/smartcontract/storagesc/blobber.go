package storagesc

import (
	"encoding/json"
	"sort"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

const blobberHealthTime = 60 * 60 // 1 Hour

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

func updateBlobberInList(list []*StorageNode, update *StorageNode) (ok bool) {
	for i, b := range list {
		if b.ID == update.ID {
			list[i], ok = update, true
			return
		}
	}
	return
}

func (sc *StorageSmartContract) filterHealthyBlobbers(blobbersList *StorageNodes) *StorageNodes {
	healthyBlobbersList := &StorageNodes{}
	for _, blobberNode := range blobbersList.Nodes {
		if blobberNode.LastHealthCheck > (time.Now().Unix() - blobberHealthTime) {
			healthyBlobbersList.Nodes = append(healthyBlobbersList.Nodes, blobberNode)
		}
	}
	return healthyBlobbersList
}

func (sc *StorageSmartContract) blobberHealthCheck(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("blobberHealthCheck_failed", "Failed to get blobber list"+err.Error())
	}
	existingBlobber := &StorageNode{}
	err = existingBlobber.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	existingBlobber.ID = t.ClientID
	existingBlobber.PublicKey = t.PublicKey
	existingBlobber.LastHealthCheck = time.Now().Unix()
	blobberBytes, _ := balances.GetTrieNode(existingBlobber.GetKey(sc.ID))
	if blobberBytes == nil {
		return "", common.NewError("blobberHealthCheck_failed", "Blobber doesn't exists"+err.Error())
	}
	for i := 0; i < len(allBlobbersList.Nodes); i++ {
		if allBlobbersList.Nodes[i].ID == existingBlobber.ID {
			allBlobbersList.Nodes[i].LastHealthCheck = time.Now().Unix()
			balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
			break
		}
	}
	balances.InsertTrieNode(existingBlobber.GetKey(sc.ID), existingBlobber)
	buff := existingBlobber.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("add_blobber_failed",
			"can't get config: "+err.Error())
	}

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

	if err = newBlobber.validate(conf); err != nil {
		return "", common.NewError("add_blobber_failed",
			"invalid values in request: "+err.Error())
	}

	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey

	_, err = balances.GetTrieNode(newBlobber.GetKey(sc.ID))

	// unexpected error
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("add_blobber_failed", err.Error())
	}

	// insert or update blobber
	if err == util.ErrValueNotPresent {
		allBlobbersList.Nodes = append(allBlobbersList.Nodes, &newBlobber)
	} else {
		if !updateBlobberInList(allBlobbersList.Nodes, &newBlobber) {
			return "", common.NewError("add_blobber_failed",
				"blobber not found in all blobbers list")
		}
	}

	balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	balances.InsertTrieNode(newBlobber.GetKey(sc.ID), &newBlobber)

	return string(newBlobber.Encode()), nil
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
