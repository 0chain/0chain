package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
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

func (sc *StorageSmartContract) getBlobberBytes(blobberID string,
	balances c_state.StateContextI) (b []byte, err error) {

	var (
		blobber      StorageNode
		blobberBytes util.Serializable
	)

	blobber.ID = blobberID
	blobberBytes, err = balances.GetTrieNode(blobber.GetKey(sc.ID))

	if err != nil {
		return
	}

	return blobberBytes.Encode(), nil
}

func (sc *StorageSmartContract) getBlobber(blobberID string,
	balances c_state.StateContextI) (blobber *StorageNode, err error) {

	var b []byte
	if b, err = sc.getBlobberBytes(blobberID, balances); err != nil {
		return
	}

	blobber = new(StorageNode)
	if err = blobber.Decode(b); err != nil {
		return nil, fmt.Errorf("decoding stored blobber: %v", err)
	}

	return
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

// remove blobber (when a blobber provides capacity = 0)
func (sc *StorageSmartContract) removeBlobber(t *transaction.Transaction,
	blobber *StorageNode, existingBytes util.Serializable, all *StorageNodes,
	balances c_state.StateContextI) (
	existingBlobber *StorageNode, sp *stakePool, err error) {

	// is the blobber exists?
	if existingBytes == nil {
		return nil, nil, errors.New("invalid capacity of blobber: 0")
	}

	existingBlobber = new(StorageNode)
	if err = existingBlobber.Decode(existingBytes.Encode()); err != nil {
		return nil, nil, fmt.Errorf("can't decode existing blobber: %v", err)
	}

	existingBlobber.Capacity = 0 // change it to zero for the removing

	// update stake pool
	if sp, err = sc.getStakePool(existingBlobber.ID, balances); err != nil {
		return nil, nil, fmt.Errorf("can't get related stake pool: %v", err)
	}

	_, err = sp.update(t.CreationDate, existingBlobber, balances)
	if err != nil {
		return nil, nil, fmt.Errorf("can't update related stake pool: v", err)
	}

	if err = sp.save(sc.ID, existingBlobber.ID, balances); err != nil {
		return nil, nil, fmt.Errorf("can't save related stake pool: v", err)
	}

	// remove from the all list, since the blobber can't accept new allocations
	var (
		i  int
		ab *StorageNode
	)
	for i, ab = range all.Nodes {
		if ab.ID == existingBlobber.ID {
			break // found
		}
	}

	// if found
	if ab.ID == existingBlobber.ID {
		all.Nodes = append(all.Nodes[:i], all.Nodes[i+1:]...) // remove from all
	}

	return // removed, opened offers are still opened
}

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	blobber *StorageNode, all *StorageNodes, balances c_state.StateContextI) (
	sp *stakePool, err error) {

	sp = newStakePool(blobber.ID) // create new

	// create stake pool
	var stake state.Balance                                   // required stake
	stake, err = sp.update(t.CreationDate, blobber, balances) //
	if err != nil {
		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	// lock required stake
	if stake > 0 {
		if state.Balance(t.Value) < stake {
			return nil, fmt.Errorf("not enough tokens for stake: %d < %d",
				t.Value, stake)
		}

		if _, _, err = sp.fill(t, balances); err != nil {
			return nil, fmt.Errorf("transferring tokens to stake pool: %v", err)
		}

		// release all tokens over the required stake, moving them to
		// unlocked pool of the stake pool
		_, err = sp.update(t.CreationDate, blobber, balances)
		if err != nil {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
	}

	// add to all
	all.Nodes = append(all.Nodes, blobber)
	return
}

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	blobber *StorageNode, all *StorageNodes, balances c_state.StateContextI) (
	sp *stakePool, err error) {

	if sp, err = sc.getStakePool(blobber.ID, balances); err != nil {
		return nil, fmt.Errorf("can't get related stake pool", err)
	}

	var stake state.Balance
	if stake, err = sp.update(t.CreationDate, blobber, balances); err != nil {
		return nil, fmt.Errorf("updating stake of blobber: %v", err)
	}

	// is more tokens required
	if sp.Locked.Balance < stake {
		if state.Balance(t.Value) < stake-sp.Locked.Balance {
			return nil, fmt.Errorf("not enough tokens for stake: %d < %d",
				t.Value, stake)
		}
		// lock tokens
		if _, _, err = sp.fill(t, balances); err != nil {
			return nil, fmt.Errorf("locking tokens in stake pool: %v", err)
		}
		// unlock all tokens over the required amount
		if _, err = sp.update(t.CreationDate, blobber, balances); err != nil {
			return nil, fmt.Errorf("updating stake pool: %v", err)
		}
	}

	// update in the list, or add to the list if the blobber was removed before
	if !updateBlobberInList(all.Nodes, blobber) {
		all.Nodes = append(all.Nodes, blobber)
	}

	return // success
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

// addBlobber adds, updates or removes a blobber; the blobber should
// provide required tokens for stake pool; otherwise it will not be
// registered; if it provides token, but it's already registered and
// related stake pool has required tokens, then no tokens will be
// transfered; if it provides more tokens then required, then all
// tokens left will be moved to unlocked part of related stake pool;
// the part can be moved back to the blobber anytime or used to
// increase blobber's capacity or write_price next time
func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	// SC configurations
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("add_blobber_failed",
			"can't get config: "+err.Error())
	}

	// all registered active blobbers
	allBlobbersList, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("add_blobber_failed",
			"Failed to get blobber list: "+err.Error())
	}

	// read request
	var newBlobber = new(StorageNode)
	if err = newBlobber.Decode(input); err != nil {
		return "", common.NewError("add_blobber_failed",
			"malformed request: "+err.Error())
	}

	// when capacity is 0, then the blobber want be removed
	if newBlobber.Capacity > 0 {
		// validate the request
		if err = newBlobber.validate(conf); err != nil {
			return "", common.NewError("add_blobber_failed",
				"invalid values in request: "+err.Error())
		}
	}

	// set transaction information
	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey

	// check out stored
	var existb util.Serializable
	existb, err = balances.GetTrieNode(newBlobber.GetKey(sc.ID))

	// unexpected error
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("add_blobber_failed", err.Error())
	}

	var sp *stakePool
	switch {

	// remove blobber case
	case newBlobber.Capacity == 0:
		// use loaded blobber in response
		newBlobber, sp, err = sc.removeBlobber(t, newBlobber, existb,
			allBlobbersList, balances)

	// insert blobber case
	case err == util.ErrValueNotPresent:
		sp, err = sc.insertBlobber(t, newBlobber, allBlobbersList, balances)

	// update blobber case
	default:
		sp, err = sc.updateBlobber(t, newBlobber, allBlobbersList, balances)

	}

	if err != nil {
		return "", common.NewError("add_blobber_failed", err.Error())
	}

	// save related stake pool
	if err = sp.save(sc.ID, newBlobber.ID, balances); err != nil {
		return "", common.NewError("add_blobber_failed",
			"saving related stake pool: "+err.Error())
	}

	// save the all
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbersList)
	if err != nil {
		return "", common.NewError("add_blobber_failed",
			"saving all blobbers: "+err.Error())
	}

	// save the blobber
	_, err = balances.InsertTrieNode(newBlobber.GetKey(sc.ID), newBlobber)
	if err != nil {
		return "", common.NewError("add_blobber_failed",
			"saving blobber: "+err.Error())
	}

	return string(newBlobber.Encode()), nil
}

func (sc *StorageSmartContract) commitBlobberRead(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	// TODO (sfxdx): move tokens: read pool to blobber

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

func (sc *StorageSmartContract) commitBlobberConnection(
	t *transaction.Transaction, input []byte, balances c_state.StateContextI) (
	string, error) {

	// TODO (sfxdx): move tokens: write pool to challenge pool

	var commitConnection BlobberCloseConnection
	err := json.Unmarshal(input, &commitConnection)
	if err != nil {
		return "", err
	}

	if !commitConnection.Verify() {
		return "", common.NewError("invalid_parameters", "Invalid input")
	}

	if commitConnection.WriteMarker.BlobberID != t.ClientID {
		return "", common.NewError("invalid_parameters",
			"Invalid Blobber ID for closing connection. Write marker not for this blobber")
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = commitConnection.WriteMarker.AllocationID
	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))

	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters",
			"Invalid allocation ID")
	}

	err = allocationObj.Decode(allocationBytes.Encode())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters",
			"Invalid allocation ID. Failed to decode from DB")
	}

	if allocationObj.Owner != commitConnection.WriteMarker.ClientID {
		return "", common.NewError("invalid_parameters",
			"Write marker has to be by the same client as owner of the allocation")
	}

	blobberAllocation, ok := allocationObj.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("invalid_parameters",
			"Blobber is not part of the allocation")
	}

	blobberAllocationBytes, err := json.Marshal(blobberAllocation)

	if !commitConnection.WriteMarker.VerifySignature(allocationObj.OwnerPublicKey) {
		return "", common.NewError("invalid_parameters",
			"Invalid signature for write marker")
	}

	if blobberAllocation.AllocationRoot == commitConnection.AllocationRoot &&
		blobberAllocation.LastWriteMarker != nil &&
		blobberAllocation.LastWriteMarker.PreviousAllocationRoot ==
			commitConnection.PrevAllocationRoot {

		return string(blobberAllocationBytes), nil
	}

	if blobberAllocation.AllocationRoot != commitConnection.PrevAllocationRoot {
		return "", common.NewError("invalid_parameters",
			"Previous allocation root does not match the latest allocation root")
	}

	if blobberAllocation.Stats.UsedSize+commitConnection.WriteMarker.Size >
		blobberAllocation.Size {

		return "", common.NewError("invalid_parameters",
			"Size for blobber allocation exceeded maximum")
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
