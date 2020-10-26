package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

const blobberHealthTime = 60 * 60 // 1 Hour

func (sc *StorageSmartContract) getBlobbersList(balances cstate.StateContextI) (*StorageNodes, error) {
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
	balances cstate.StateContextI) (b []byte, err error) {

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
	balances cstate.StateContextI) (blobber *StorageNode, err error) {

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
	existingBytes util.Serializable, all *StorageNodes) (
	existingBlobber *StorageNode, err error) {

	// is the blobber exists?
	if existingBytes == nil {
		return nil, errors.New("invalid capacity of blobber: 0")
	}

	existingBlobber = new(StorageNode)
	if err = existingBlobber.Decode(existingBytes.Encode()); err != nil {
		return nil, fmt.Errorf("can't decode existing blobber: %v", err)
	}

	existingBlobber.Capacity = 0 // change it to zero for the removing

	// remove from the all list, since the blobber can't accept new allocations
	all.Nodes.remove(existingBlobber.ID)

	// statistic
	sc.statIncr(statRemoveBlobber)
	sc.statDecr(statNumberOfBlobbers)
	return // removed, opened offers are still opened
}

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	blobber *StorageNode, existingBytes util.Serializable, all *StorageNodes) (
	err error) {

	var existingBlobber StorageNode
	if err = existingBlobber.Decode(existingBytes.Encode()); err != nil {
		return fmt.Errorf("can't decode existing blobber: %v", err)
	}

	blobber.Used = existingBlobber.Used      // copy
	blobber.LastHealthCheck = t.CreationDate // health

	// update in the list, or add to the list if the blobber was removed before
	all.Nodes.add(blobber)

	// statistics
	sc.statIncr(statUpdateBlobber)
	// if has removed (the reborn case)
	if existingBlobber.Capacity == 0 {
		sc.statIncr(statNumberOfBlobbers)
	}
	return // success
}

func filterHealthyBlobbers(now common.Timestamp) filterBlobberFunc {
	return filterBlobberFunc(func(b *StorageNode) (kick bool) {
		return b.LastHealthCheck <= (now - blobberHealthTime)
	})
}

func (sc *StorageSmartContract) blobberHealthCheck(t *transaction.Transaction,
	_ []byte, balances cstate.StateContextI) (string, error) {

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
	input []byte, balances cstate.StateContextI) (string, error) {

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

	switch {

	// remove blobber case
	case newBlobber.Capacity == 0:
		// use loaded blobber in response
		newBlobber, err = sc.removeBlobber(t, existb, allBlobbersList)

	// insert blobber case
	case err == util.ErrValueNotPresent:
		err = sc.insertBlobber(t, conf, newBlobber, allBlobbersList, balances)

	// update blobber case
	default:
		err = sc.updateBlobber(t, newBlobber, existb, allBlobbersList)

	}

	if err != nil {
		return "", common.NewError("add_blobber_failed", err.Error())
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

func (sc *StorageSmartContract) updateBlobberSettings(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (resp string, err error) {

	var conf *scConfig
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get SC configurations: "+err.Error())
	}

	var all *StorageNodes
	if all, err = sc.getBlobbersList(balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"Failed to get blobber list: "+err.Error())
	}

	var update = new(StorageNode)
	if err = update.Decode(input); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"malformed request: "+err.Error())
	}

	// when capacity is 0, then the blobber want be removed
	if update.Capacity > 0 {
		// terms and capacity validated here
		if err = update.validate(conf); err != nil {
			return "", common.NewError("update_blobber_settings_failed",
				"invalid values in request: "+err.Error())
		}
	} else {
		// validate given terms anyway
		if err = update.Terms.validate(conf); err != nil {
			return "", common.NewError("update_blobber_settings_failed",
				"invalid new terms in request: "+err.Error())
		}
	}

	var blob *StorageNode
	if blob, err = sc.getBlobber(update.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get the blobber: "+err.Error())
	}

	var sp *stakePool
	if sp, err = sc.getStakePool(update.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get related stake pool: "+err.Error())
	}

	if sp.Settings.DelegateWallet == "" {
		return "", common.NewError("update_blobber_settings_failed",
			"blobber's delegate_wallet is not set")
	}

	if t.ClientID != sp.Settings.DelegateWallet {
		return "", common.NewError("update_blobber_settings_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	// stake pool settings

	if err = update.StakePoolSettings.validate(conf); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"validating new stake pool settings: "+err.Error())
	}

	sp.Settings.MinStake = update.StakePoolSettings.MinStake
	sp.Settings.MaxStake = update.StakePoolSettings.MaxStake
	sp.Settings.ServiceCharge = update.StakePoolSettings.ServiceCharge
	sp.Settings.NumDelegates = update.StakePoolSettings.NumDelegates

	// blobber settings
	blob.Terms = update.Terms
	blob.Capacity = update.Capacity

	if blob.Capacity == 0 {
		// if already removed
		if update.Capacity == 0 {
			// keep removed
		} else {
			// reborn
			sc.statIncr(statNumberOfBlobbers)
			all.Nodes.add(blob)
		}
		sc.statIncr(statUpdateBlobber)
	} else {
		// alive blobber
		if update.Capacity == 0 {
			// remove
			all.Nodes.remove(blob.ID)
			sc.statIncr(statRemoveBlobber)
			sc.statDecr(statNumberOfBlobbers)
		} else {
			// keep alive
			all.Nodes.add(blob)
			sc.statIncr(statUpdateBlobber)
		}
	}

	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			err.Error())
	}

	// save the all
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, all)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving all blobbers: "+err.Error())
	}

	// save the blobber
	_, err = balances.InsertTrieNode(blob.GetKey(sc.ID), blob)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving blobber: "+err.Error())
	}

	// save stake pool
	if err = sp.save(sc.ID, blob.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving stake pool: "+err.Error())
	}

	return string(blob.Encode()), nil
}

func (sc *StorageSmartContract) commitBlobberRead(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (resp string, err error) {

	var commitRead = &ReadConnection{}
	if err = commitRead.Decode(input); err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"decoding input: %v", err)
	}

	if commitRead.ReadMarker == nil {
		return "", common.NewError("commit_blobber_read",
			"malformed request: missing read_marker")
	}

	var (
		lastBlobberClientReadBytes util.Serializable
		lastCommittedRM            = &ReadConnection{}
		lastKnownCtr               int64
	)

	lastBlobberClientReadBytes, err = balances.GetTrieNode(
		commitRead.GetKey(sc.ID))

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get latest blobber client read: %v", err)
	}

	err = nil // reset possible ErrValueNotPresent

	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes.Encode())
		lastKnownCtr = lastCommittedRM.ReadMarker.ReadCounter
	}

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't verify read marker: %v", err)
	}

	// move tokens to blobber's stake pool from client's read pool
	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(commitRead.ReadMarker.AllocationID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related allocation: %v", err)
	}

	if commitRead.ReadMarker.Timestamp < alloc.StartTime {
		return "", common.NewError("commit_blobber_read",
			"early reading, allocation not started yet")
	} else if commitRead.ReadMarker.Timestamp > alloc.Until() {
		return "", common.NewError("commit_blobber_read",
			"late reading, allocation expired")
	}

	var details *BlobberAllocation
	for _, d := range alloc.BlobberDetails {
		if d.BlobberID == commitRead.ReadMarker.BlobberID {
			details = d
			break
		}
	}

	if details == nil {
		return "", common.NewError("commit_blobber_read",
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
	var rp *readPool
	if rp, err = sc.getReadPool(userID, balances); err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related read pool: %v", err)
	}

	var sp *stakePool
	sp, err = sc.getStakePool(commitRead.ReadMarker.BlobberID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related stake pool: %v", err)
	}

	resp, err = rp.moveToBlobber(sc.ID, commitRead.ReadMarker.AllocationID,
		commitRead.ReadMarker.BlobberID, sp, t.CreationDate, value, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't transfer tokens from read pool to stake pool: %v", err)
	}
	details.ReadReward += value // stat
	details.Spent += value      // reduce min lock demand left

	// save pools
	err = sp.save(sc.ID, commitRead.ReadMarker.BlobberID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't save stake pool: %v", err)
	}

	if err = rp.save(sc.ID, userID, balances); err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't save read pool: %v", err)
	}

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't save allocation: %v", err)
	}

	// save read marker
	balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	sc.newRead(balances, numReads)

	return // ok, the response and nil
}

func sizePrice(size int64, price state.Balance) float64 {
	return sizeInGB(size) * float64(price)
}

// (expire - last_challenge_time) /  (allocation duration)
func allocLeftRatio(start, expire, last common.Timestamp) float64 {
	return float64(expire-last) / float64(expire-start)
}

// commitMoveTokens moves tokens on connection commit (on write marker),
// if data written (size > 0) -- from write pool to challenge pool, otherwise
// (delete write marker) from challenge back to write pool
func (sc *StorageSmartContract) commitMoveTokens(alloc *StorageAllocation,
	size int64, details *BlobberAllocation, wmTime, now common.Timestamp,
	balances cstate.StateContextI) (err error) {

	if size == 0 {
		return // zero size write marker -- no tokens movements
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
		move  state.Balance
	)

	// the details will be saved in caller with allocation object (the details
	// is part of the allocation object)

	if size > 0 {
		move = details.upload(size, wmTime,
			alloc.restDurationInTimeUnits(wmTime))
		// upload (write_pool -> challenge_pool)
		err = wp.moveToChallenge(alloc.ID, details.BlobberID, cp, now, move)
		if err != nil {
			return fmt.Errorf("can't move tokens to challenge pool: %v", err)
		}
		alloc.MovedToChallenge += move
		details.Spent += move
	} else {
		// delete (challenge_pool -> write_pool)
		move = details.delete(-size, wmTime,
			alloc.restDurationInTimeUnits(wmTime))
		err = cp.moveToWritePool(alloc.ID, details.BlobberID, until, wp, move)
		if err != nil {
			return fmt.Errorf("can't move tokens to write pool: %v", err)
		}
		alloc.MovedBack += move
		details.Returned += move
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
	t *transaction.Transaction, input []byte, balances cstate.StateContextI) (
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
		commitConnection.WriteMarker.Timestamp, t.CreationDate, balances)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"moving tokens: %v", err)
	}

	// save allocation object
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"saving allocation object: %v", err)
	}

	detailsBytes, err = json.Marshal(details.LastWriteMarker)
	sc.newWrite(balances, commitConnection.WriteMarker.Size)
	return string(detailsBytes), err
}
