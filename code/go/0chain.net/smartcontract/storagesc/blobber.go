package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/smartcontract/dbs/event"

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
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
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
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
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

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	conf *scConfig, blobber *StorageNode, blobbers *StorageNodes,
	balances cstate.StateContextI,
) (err error) {
	// check terms
	if err = blobber.Terms.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber terms: %v", err)
	}

	if blobber.Capacity <= 0 {
		return sc.removeBlobber(t, blobber, blobbers, balances)
	}

	// check params
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
	}

	// get saved blobber
	savedBlobber, err := sc.getBlobber(blobber.ID, balances)
	if err != nil {
		return fmt.Errorf("can't get or decode saved blobber: %v", err)
	}

	blobber.LastHealthCheck = t.CreationDate
	blobber.Used = savedBlobber.Used

	// update the list
	blobbers.Nodes.add(blobber)

	if err := emitAddOrOverwriteBlobber(blobber, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
	}

	// update statistics
	sc.statIncr(statUpdateBlobber)

	if savedBlobber.Capacity == 0 {
		sc.statIncr(statNumberOfBlobbers) // reborn, if it was "removed"
	}

	// update stake pool settings
	var sp *stakePool
	if sp, err = sc.getStakePool(blobber.ID, balances); err != nil {
		return fmt.Errorf("can't get stake pool:  %v", err)
	}

	if err = blobber.StakePoolSettings.validate(conf); err != nil {
		return fmt.Errorf("invalid new stake pool settings:  %v", err)
	}

	sp.Settings.MinStake = blobber.StakePoolSettings.MinStake
	sp.Settings.MaxStake = blobber.StakePoolSettings.MaxStake
	sp.Settings.ServiceCharge = blobber.StakePoolSettings.ServiceCharge
	sp.Settings.NumDelegates = blobber.StakePoolSettings.NumDelegates

	// save stake pool
	if err = sp.save(sc.ID, blobber.ID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	return
}

// remove blobber (when a blobber provides capacity = 0)
func (sc *StorageSmartContract) removeBlobber(t *transaction.Transaction,
	blobber *StorageNode, blobbers *StorageNodes, balances cstate.StateContextI,
) (err error) {
	// get saved blobber
	savedBlobber, err := sc.getBlobber(blobber.ID, balances)
	if err != nil {
		return fmt.Errorf("can't get or decode saved blobber: %v", err)
	}

	// set to zero explicitly, for "direct" calls
	blobber.Capacity = 0

	// remove from the all list, since the blobber can't accept new allocations
	if savedBlobber.Capacity > 0 {
		blobbers.Nodes.remove(blobber.ID)
		sc.statIncr(statRemoveBlobber)
		sc.statDecr(statNumberOfBlobbers)
	}

	balances.EmitEvent(event.TypeStats, event.TagDeleteBlobber, blobber.ID, blobber.ID)

	return // opened offers are still opened
}

// inserts, updates or removes blobber
// the blobber should provide required tokens for stake pool; otherwise it will not be
// registered; if it provides token, but it's already registered and
// related stake pool has required tokens, then no tokens will be
// transfered; if it provides more tokens then required, then all
// tokens left will be moved to unlocked part of related stake pool;
// the part can be moved back to the blobber anytime or used to
// increase blobber's capacity or write_price next time
func (sc *StorageSmartContract) addBlobber(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI,
) (string, error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"can't get config: "+err.Error())
	}

	// get registered blobbers
	blobbers, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"Failed to get blobber list: "+err.Error())
	}

	// set blobber
	var blobber = new(StorageNode)
	if err = blobber.Decode(input); err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"malformed request: "+err.Error())
	}

	// set transaction information
	blobber.ID = t.ClientID
	blobber.PublicKey = t.PublicKey

	// insert, update or remove blobber
	if err = sc.insertBlobber(t, conf, blobber, blobbers, balances); err != nil {
		return "", common.NewError("add_or_update_blobber_failed", err.Error())
	}

	// save all the blobbers
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, blobbers)
	if err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"saving all blobbers: "+err.Error())
	}

	// save the blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"saving blobber: "+err.Error())
	}

	return string(blobber.Encode()), nil
}

// update blobber settinngs by owner of DelegateWallet
func (sc *StorageSmartContract) updateBlobberSettings(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI,
) (resp string, err error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get config: "+err.Error())
	}

	var blobbers *StorageNodes
	if blobbers, err = sc.getBlobbersList(balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"failed to get blobber list: "+err.Error())
	}

	var updatedBlobber = new(StorageNode)
	if err = updatedBlobber.Decode(input); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"malformed request: "+err.Error())
	}

	var blobber *StorageNode
	if blobber, err = sc.getBlobber(updatedBlobber.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get the blobber: "+err.Error())
	}

	var sp *stakePool
	if sp, err = sc.getStakePool(updatedBlobber.ID, balances); err != nil {
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

	blobber.Terms = updatedBlobber.Terms
	blobber.Capacity = updatedBlobber.Capacity

	if err = sc.updateBlobber(t, conf, blobber, blobbers, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed", err.Error())
	}

	// save all the blobbers
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, blobbers)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving all blobbers: "+err.Error())
	}

	// save blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving blobber: "+err.Error())
	}

	return string(blobber.Encode()), nil
}

func filterHealthyBlobbers(now common.Timestamp) filterBlobberFunc {
	return filterBlobberFunc(func(b *StorageNode) (kick bool) {
		return b.LastHealthCheck <= (now - blobberHealthTime)
	})
}

func (sc *StorageSmartContract) blobberHealthCheck(t *transaction.Transaction,
	_ []byte, balances cstate.StateContextI,
) (string, error) {
	all, err := sc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"Failed to get blobber list: "+err.Error())
	}

	var blobber *StorageNode
	if blobber, err = sc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	blobber.LastHealthCheck = t.CreationDate

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

	err = emitUpdateBlobber(blobber, balances)
	if err != nil {
		return "", common.NewError("blobber_health_check_failed", err.Error())
	}
	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID),
		blobber)
	if err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}

	return string(blobber.Encode()), nil
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

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker, balances)
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
		userID   = commitRead.ReadMarker.PayerID
	)

	// if 3rd party pays
	err = commitRead.ReadMarker.verifyAuthTicket(alloc, t.CreationDate, balances)
	if err != nil {
		return "", common.NewError("commit_blobber_read", err.Error())
	}

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
	_, err = balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	if err != nil {
		return "", common.NewError("saving read marker", err.Error())
	}
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
	wps, err := alloc.getAllocationPools(sc, balances)
	if err != nil {
		return fmt.Errorf("can't move tokens to challenge pool: %v", err)
	}

	if size > 0 {
		move = details.upload(size, wmTime,
			alloc.restDurationInTimeUnits(wmTime))

		err = wps.moveToChallenge(alloc.ID, details.BlobberID, cp, now, move)
		if err != nil {
			return fmt.Errorf("can't move tokens to challenge pool: %v", err)
		}

		alloc.MovedToChallenge += move
		details.Spent += move
	} else {
		// delete (challenge_pool -> write_pool)
		move = details.delete(-size, wmTime, alloc.restDurationInTimeUnits(wmTime))
		wp, err := wps.getOwnerWP()
		if err != nil {
			return fmt.Errorf("can't move tokens to challenge pool: %v", err)
		}
		err = cp.moveToWritePool(alloc, details.BlobberID, until, wp, move)
		if err != nil {
			return fmt.Errorf("can't move tokens to write pool: %v", err)
		}
		alloc.MovedBack += move
		details.Returned += move
	}

	if err := wps.saveWritePools(sc.ID, balances); err != nil {
		return fmt.Errorf("can't move tokens to challenge pool: %v", err)
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

	if !commitConnection.WriteMarker.VerifySignature(alloc.OwnerPublicKey, balances) {
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

	err = emitAddOrOverwriteWriteMarker(commitConnection.WriteMarker, balances, t)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"emitting write marker event: %v", err)
	}

	detailsBytes, err = json.Marshal(details.LastWriteMarker)
	sc.newWrite(balances, commitConnection.WriteMarker.Size)
	return string(detailsBytes), err
}
