package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

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

	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return nil, nil, common.NewError("fini_alloc_failed",
			"can't get SC configurations: "+err.Error())
	}

	// update stake pool
	if sp, err = sc.getStakePool(existingBlobber.ID, balances); err != nil {
		return nil, nil, fmt.Errorf("can't get related stake pool: %v", err)
	}

	_, err = sp.update(conf, sc.ID, t.CreationDate, existingBlobber, balances)
	if err != nil {
		return nil, nil, fmt.Errorf("can't update related stake pool: %v", err)
	}

	if err = sp.save(sc.ID, existingBlobber.ID, balances); err != nil {
		return nil, nil, fmt.Errorf("can't save related stake pool: %v", err)
	}

	// remove from the all list, since the blobber can't accept new allocations
	all.Nodes.remove(existingBlobber.ID)
	return // removed, opened offers are still opened
}

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	blobber *StorageNode, all *StorageNodes, balances c_state.StateContextI) (
	sp *stakePool, err error) {

	// the stake pool can be created by related validator
	sp, err = sc.getOrCreateStakePool(blobber.ID, balances)
	if err != nil {
		return
	}

	blobber.LastHealthCheck = t.CreationDate

	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return nil, common.NewError("fini_alloc_failed",
			"can't get SC configurations: "+err.Error())
	}

	// create stake pool
	var (
		stake state.Balance // required stake
		info  *stakePoolUpdateInfo
	)
	info, err = sp.update(conf, sc.ID, t.CreationDate, blobber, balances) //
	if err != nil {
		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	stake = info.stake

	// lock required stake
	if stake > 0 {
		if state.Balance(t.Value) < stake {
			return nil, fmt.Errorf("not enough tokens for stake: %d < %d",
				t.Value, stake)
		}

		if err = checkFill(t, balances); err != nil {
			return nil, err
		}

		if _, _, err = sp.fill(t, balances); err != nil {
			return nil, fmt.Errorf("transferring tokens to stake pool: %v", err)
		}

		// release all tokens over the required stake,
		// moving them back to blobber
		_, err = sp.update(conf, sc.ID, t.CreationDate, blobber, balances)
		if err != nil {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
	}

	all.Nodes.add(blobber) // add to all
	return
}

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	blobber *StorageNode, existingBytes util.Serializable, all *StorageNodes,
	balances c_state.StateContextI) (sp *stakePool, err error) {

	var existingBlobber StorageNode
	if err = existingBlobber.Decode(existingBytes.Encode()); err != nil {
		return nil, fmt.Errorf("can't decode existing blobber: %v", err)
	}

	blobber.Used = existingBlobber.Used      // copy
	blobber.LastHealthCheck = t.CreationDate // health

	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return nil, common.NewError("fini_alloc_failed",
			"can't get SC configurations: "+err.Error())
	}

	if sp, err = sc.getStakePool(blobber.ID, balances); err != nil {
		return nil, fmt.Errorf("can't get related stake pool: %v", err)
	}

	var (
		lack state.Balance
		info *stakePoolUpdateInfo
	)
	info, err = sp.update(conf, sc.ID, t.CreationDate, blobber, balances)
	if err != nil {
		return nil, fmt.Errorf("updating stake of blobber: %v", err)
	}

	lack = info.lack

	// is more tokens required
	if lack > 0 {
		if state.Balance(t.Value) < lack {
			return nil, fmt.Errorf("not enough tokens for stake: %d < %d",
				t.Value, lack)
		}
		// check blobber's tokens
		if err = checkFill(t, balances); err != nil {
			return nil, err
		}
		// lock tokens
		if _, _, err = sp.fill(t, balances); err != nil {
			return nil, fmt.Errorf("locking tokens in stake pool: %v", err)
		}
		// unlock all tokens over the required amount
		_, err = sp.update(conf, sc.ID, t.CreationDate, blobber, balances)
		if err != nil {
			return nil, fmt.Errorf("updating stake pool: %v", err)
		}
	}

	// update in the list, or add to the list if the blobber was removed before
	all.Nodes.add(blobber)
	return // success
}

func filterHealthyBlobbers(now common.Timestamp) filterBlobberFunc {
	return filterBlobberFunc(func(b *StorageNode) bool {
		return b.LastHealthCheck <= (now - blobberHealthTime)
	})
}

func (sc *StorageSmartContract) blobberHealthCheck(t *transaction.Transaction,
	_ []byte, balances c_state.StateContextI) (string, error) {

	all, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"Failed to get blobber list: "+err.Error())
	}

	var existingBlobber *StorageNode
	if existingBlobber, err = sc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	existingBlobber.LastHealthCheck = t.CreationDate

	var i, ok = all.Nodes.getIndex(t.ClientID)
	// if blobber has been removed, then it shouldn't send the health check
	// transactions
	if !ok {
		return "", common.NewError("blobber_health_check_failed", "blobber "+
			t.ClientID+" not found in all blobbers list")
	}
	var found = all.Nodes[i]
	found.LastHealthCheck = t.CreationDate
	if _, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, all); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save all blobbers list: "+err.Error())
	}

	_, err = balances.InsertTrieNode(existingBlobber.GetKey(sc.ID),
		existingBlobber)
	if err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}

	return string(existingBlobber.Encode()), nil
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
		sp, err = sc.updateBlobber(t, newBlobber, existb, allBlobbersList,
			balances)

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

	commitRead := &ReadConnection{}
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	if commitRead.ReadMarker == nil {
		return "", errors.New("malformed request: missing read_marker")
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
		return "", common.NewError("commit_read_failed",
			"Invalid read marker."+err.Error())
	}

	// move tokens to blobber's stake pool from client's read pool
	alloc, err := sc.getAllocation(commitRead.ReadMarker.AllocationID, balances)
	if err != nil {
		return "", common.NewError("commit_read_failed",
			"can't get related allocation: "+err.Error())
	}

	var details *BlobberAllocation
	for _, d := range alloc.BlobberDetails {
		if d.BlobberID == commitRead.ReadMarker.BlobberID {
			details = d
			break
		}
	}

	if details == nil {
		return "", common.NewError("commit_read_failed",
			"blobber doesn't belong to allocation")
	}

	const CHUNK_SIZE = 64 * KB

	// one read is one 64 KB block
	var (
		numReads = commitRead.ReadMarker.ReadCounter - lastKnownCtr
		sizeRead = sizeInGB(numReads * CHUNK_SIZE)
		value    = state.Balance(float64(details.Terms.ReadPrice) * sizeRead)
		userID   = commitRead.ReadMarker.ClientID
	)

	// move tokens from read pool to blobber
	rp, err := sc.getReadPool(userID, balances)
	if err != nil {
		return "", common.NewError("commit_read_failed",
			"can't get related read pool: "+err.Error())
	}

	var sp *stakePool
	sp, err = sc.getStakePool(commitRead.ReadMarker.BlobberID, balances)
	if err != nil {
		return "", common.NewError("commit_read_failed",
			"can't get related stake pool: "+err.Error())
	}

	err = rp.moveToBlobber(commitRead.ReadMarker.AllocationID,
		commitRead.ReadMarker.BlobberID, sp, t.CreationDate, value)
	if err != nil {
		return "", common.NewError("commit_read_failed",
			"can't transfer tokens from read pool to stake pool: "+err.Error())
	}
	sp.BlobberReward += value   // rewards statistic
	sp.Rewards += value         // blobber rewards (not a stake)
	details.ReadReward += value // stat
	details.Spent += value      // reduce min lock demand left

	// save pools
	err = sp.save(sc.ID, commitRead.ReadMarker.BlobberID, balances)
	if err != nil {
		return "", common.NewError("commit_read_failed",
			"can't save stake pool: "+err.Error())
	}

	if err = rp.save(sc.ID, userID, balances); err != nil {
		return "", common.NewError("commit_read_failed",
			"can't save read pool: "+err.Error())
	}

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("commit_read_failed",
			"can't save allocation: "+err.Error())
	}

	// save read marker
	balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	sc.newRead(balances, numReads)
	return "success", nil
}

func sizePrice(size int64, price state.Balance) float64 {
	return sizeInGB(size) * float64(price)
}

// (expire - last_challenge_time) /  (allocation duration)
func allocLeftRatio(start, expire, last common.Timestamp) float64 {
	return float64(expire-last) / float64(expire-start)
}

// commitMoveTokens moves tokens on connection commit (on write marker),
// if data written (size > 0) -- from write pool to challenge pool
func (sc *StorageSmartContract) commitMoveTokens(alloc *StorageAllocation,
	size int64, details *BlobberAllocation, balances c_state.StateContextI) (
	err error) {

	if size == 0 {
		return errors.New("zero size write marker given")
	}

	if size < 0 {
		return // data has deleted, nothing to do here
	}

	// write pool
	wp, err := sc.getWritePool(alloc.Owner, balances)
	if err != nil {
		return errors.New("can't get related write pool")
	}

	// challenge pool
	cp, err := sc.getChallengePool(alloc.ID, balances)
	if err != nil {
		return errors.New("can't get related challenge pool")
	}

	var (
		until = alloc.Until()
		value state.Balance
	)

	if size > 0 {
		// write
		value = state.Balance(float64(details.Terms.WritePrice) *
			sizeInGB(size))

		err = wp.moveToChallenge(alloc.ID, details.BlobberID, cp, until, value)
		if err != nil {
			return fmt.Errorf("can't move tokens to challenge pool: %v", err)
		}
		alloc.MovedToChallenge += value
		details.Spent += value
	}

	// save pools
	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return fmt.Errorf("can't save write pool: %v", err)
	}
	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save challenge pool: %v", err)
	}

	return
}

func (sc *StorageSmartContract) commitBlobberConnection(
	t *transaction.Transaction, input []byte, balances c_state.StateContextI) (
	string, error) {

	var commitConnection BlobberCloseConnection
	err := json.Unmarshal(input, &commitConnection)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"malformed input: "+err.Error())
	}

	if commitConnection.WriteMarker == nil {
		return "", common.NewError("commit_connection_failed",
			"invalid input: missing write_marker")
	}

	if !commitConnection.Verify() {
		return "", common.NewError("commit_connection_failed", "Invalid input")
	}

	if commitConnection.WriteMarker.BlobberID != t.ClientID {
		return "", common.NewError("commit_connection_failed",
			"Invalid Blobber ID for closing connection. Write marker not for this blobber")
	}

	alloc, err := sc.getAllocation(commitConnection.WriteMarker.AllocationID,
		balances)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"can't get allocation: "+err.Error())
	}

	if alloc.Owner != commitConnection.WriteMarker.ClientID {
		return "", common.NewError("commit_connection_failed", "write marker has"+
			" to be by the same client as owner of the allocation")
	}

	details, ok := alloc.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("commit_connection_failed",
			"Blobber is not part of the allocation")
	}

	detailsBytes, err := json.Marshal(details)

	if !commitConnection.WriteMarker.VerifySignature(alloc.OwnerPublicKey) {
		return "", common.NewError("commit_connection_failed",
			"Invalid signature for write marker")
	}

	if details.AllocationRoot == commitConnection.AllocationRoot &&
		details.LastWriteMarker != nil &&
		details.LastWriteMarker.PreviousAllocationRoot ==
			commitConnection.PrevAllocationRoot {

		return string(detailsBytes), nil
	}

	if details.AllocationRoot != commitConnection.PrevAllocationRoot {
		return "", common.NewError("commit_connection_failed",
			"Previous allocation root does not match the latest allocation root")
	}

	if details.Stats.UsedSize+commitConnection.WriteMarker.Size >
		details.Size {

		return "", common.NewError("commit_connection_failed",
			"Size for blobber allocation exceeded maximum")
	}

	details.AllocationRoot = commitConnection.AllocationRoot
	details.LastWriteMarker = commitConnection.WriteMarker
	details.Stats.UsedSize += commitConnection.WriteMarker.Size
	details.Stats.NumWrites++

	alloc.Stats.UsedSize += commitConnection.WriteMarker.Size
	alloc.Stats.NumWrites++

	// check time boundaries
	if commitConnection.WriteMarker.Timestamp < alloc.StartTime {
		return "", common.NewError("commit_connection_failed",
			"write marker time is before allocation created")
	}

	if commitConnection.WriteMarker.Timestamp > alloc.Expiration {

		return "", common.NewError("commit_connection_failed",
			"write marker time is after allocation expires")
	}

	err = sc.commitMoveTokens(alloc, commitConnection.WriteMarker.Size, details,
		balances)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"moving tokens: "+err.Error())
	}

	// save allocation object
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"saving allocation object: "+err.Error())
	}

	detailsBytes, err = json.Marshal(details.LastWriteMarker)
	sc.newWrite(balances, commitConnection.WriteMarker.Size)
	return string(detailsBytes), err
}
