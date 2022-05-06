package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"

	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/partitions"

	"go.uber.org/zap"

	"0chain.net/smartcontract/dbs/event"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

const (
	blobberHealthTime = 60 * 60 // 1 Hour
)

func (sc *StorageSmartContract) getBlobber(blobberID string,
	balances cstate.StateContextI) (blobber *StorageNode, err error) {

	blobber = new(StorageNode)
	blobber.ID = blobberID
	err = balances.GetTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return nil, err
	}

	return
}

func (sc *StorageSmartContract) getBlobberChallengePartitionLocation(blobberID string,
	balances cstate.StateContextI) (blobberChallLocation *BlobberChallengePartitionLocation, err error) {

	blobberChallLocation = new(BlobberChallengePartitionLocation)
	blobberChallLocation.ID = blobberID
	err = balances.GetTrieNode(blobberChallLocation.GetKey(sc.ID), blobberChallLocation)
	if err != nil {
		return nil, err
	}

	return
}

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	conf *Config, blobber *StorageNode,
	balances cstate.StateContextI,
) (err error) {
	// check terms
	if err = blobber.Terms.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber terms: %v", err)
	}

	if blobber.Capacity <= 0 {
		return sc.removeBlobber(t, blobber, balances)
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
	blobber.SavedData = savedBlobber.SavedData

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

	if err = validateStakePoolSettings(blobber.StakePoolSettings, conf); err != nil {
		return fmt.Errorf("invalid new stake pool settings:  %v", err)
	}

	sp.Settings.MinStake = blobber.StakePoolSettings.MinStake
	sp.Settings.MaxStake = blobber.StakePoolSettings.MaxStake
	sp.Settings.ServiceCharge = blobber.StakePoolSettings.ServiceCharge
	sp.Settings.MaxNumDelegates = blobber.StakePoolSettings.MaxNumDelegates

	if err := emitAddOrOverwriteBlobber(blobber, sp, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
	}

	// save stake pool
	if err = sp.save(sc.ID, blobber.ID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	data, _ := json.Marshal(dbs.DbUpdates{
		Id: blobber.ID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, blobber.ID, string(data))

	return
}

// remove blobber (when a blobber provides capacity = 0)
func (sc *StorageSmartContract) removeBlobber(t *transaction.Transaction,
	blobber *StorageNode, balances cstate.StateContextI,
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

	var blobber = new(StorageNode)
	if err = blobber.Decode(input); err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"malformed request: "+err.Error())
	}

	// set transaction information
	blobber.ID = t.ClientID
	blobber.PublicKey = t.PublicKey

	// insert, update or remove blobber
	if err = sc.insertBlobber(t, conf, blobber, balances); err != nil {
		return "", common.NewError("add_or_update_blobber_failed", err.Error())
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
	blobber.StakePoolSettings = updatedBlobber.StakePoolSettings

	if err = sc.updateBlobber(t, conf, blobber, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed", err.Error())
	}

	// save blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving blobber: "+err.Error())
	}

	if err := emitUpdateBlobber(blobber, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"emitting update blobber: "+err.Error())
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
	var (
		blobber *StorageNode
		err     error
	)
	if blobber, err = sc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	blobber.LastHealthCheck = t.CreationDate

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

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"cannot get config: %v", err)
	}

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
		lastCommittedRM = &ReadConnection{}
		lastKnownCtr    int64
	)

	err = balances.GetTrieNode(commitRead.GetKey(sc.ID), lastCommittedRM)
	switch err {
	case nil:
		lastKnownCtr = lastCommittedRM.ReadMarker.ReadCounter
	case util.ErrValueNotPresent:
		err = nil
	default:
		return "", common.NewErrorf("commit_blobber_read",
			"can't get latest blobber client read: %v", err)
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

	blobber, err := sc.getBlobber(details.BlobberID, balances)
	if err != nil {
		return "", common.NewError("commit_blobber_read",
			"error fetching blobber object")
	}

	const CHUNK_SIZE = 64 * KB

	var (
		numReads = commitRead.ReadMarker.ReadCounter - lastKnownCtr
		sizeRead = sizeInGB(numReads * CHUNK_SIZE)
		value    = state.Balance(float64(details.Terms.ReadPrice) * sizeRead)
		userID   = commitRead.ReadMarker.PayerID
	)

	commitRead.ReadMarker.ReadSize = sizeRead

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

	rewardRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	if blobber.LastRewardDataReadRound >= rewardRound {
		blobber.DataReadLastRewardRound += sizeRead
	} else {
		blobber.DataReadLastRewardRound = sizeRead
	}
	blobber.LastRewardDataReadRound = balances.GetBlock().Round

	if blobber.RewardPartition.StartRound >= rewardRound && blobber.RewardPartition.Timestamp > 0 {
		parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		if err != nil {
			return "", common.NewErrorf("commit_blobber_read",
				"cannot fetch ongoing partition: %v", err)
		}

		var brn BlobberRewardNode
		if err := parts.GetItem(balances, blobber.RewardPartition.Index, blobber.ID, &brn); err != nil {
			return "", common.NewErrorf("commit_blobber_read",
				"cannot fetch blobber node item from partition: %v", err)
		}

		brn.DataRead = blobber.DataReadLastRewardRound

		err = parts.UpdateItem(balances, blobber.RewardPartition.Index, &brn)
		if err != nil {
			return "", common.NewError("commit_blobber_read",
				"error updating blobber reward item")
		}

		err = parts.Save(balances)
		if err != nil {
			return "", common.NewError("commit_blobber_read",
				"error saving ongoing blobber reward partition")
		}

	}

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

	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't save blobber: %v", err)
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

	err = emitAddOrOverwriteReadMarker(commitRead.ReadMarker, balances, t)
	if err != nil {
		return "", common.NewError("saving read marker in db:", err.Error())
	}

	return // ok, the response and nil
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

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"malformed input: "+err.Error())
	}

	var commitConnection BlobberCloseConnection
	err = json.Unmarshal(input, &commitConnection)
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
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"error marshalling allocation blobber details")
	}

	if !commitConnection.WriteMarker.VerifySignature(alloc.OwnerPublicKey, balances) {
		return "", common.NewError("commit_connection_failed",
			"Invalid signature for write marker")
	}

	blobber, err := sc.getBlobber(details.BlobberID, balances)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"error fetching blobber")
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

	blobber.BytesWritten += commitConnection.WriteMarker.Size

	alloc.Stats.UsedSize += commitConnection.WriteMarker.Size
	alloc.Stats.NumWrites++

	// UpdateItem saved_data on storage node
	var storageNode *StorageNode
	if _, ok := alloc.BlobberMap[commitConnection.WriteMarker.BlobberID]; ok {
		storageNode, err = sc.getBlobber(commitConnection.WriteMarker.BlobberID, balances)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"can't get blobber")
		}
	}

	storageNode.SavedData += alloc.Stats.UsedSize

	var sp *stakePool
	if sp, err = sc.getStakePool(storageNode.ID, balances); err != nil {
		return "", common.NewError("commit_connection_failed",
			"can't get stake pool")
	}

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

	// this should be replaced with getBlobber() once storageNode is normalised
	blobberChallLocation, err := sc.getBlobberChallengePartitionLocation(t.ClientID, balances)
	if err != nil {
		if err == util.ErrValueNotPresent {
			blobberChallLocation = new(BlobberChallengePartitionLocation)
			blobberChallLocation.ID = t.ClientID
		} else {
			return "", common.NewError("commit_connection_failed",
				"error fetching blobber challenge partition location")
		}
	}

	// partition blobber challenge
	//todo: handle allocations are all deleted
	pData := &BlobberChallengeNode{
		BlobberID:    t.ClientID,
		UsedCapacity: blobber.BytesWritten,
	}

	pAllocData := &BlobberChallengeAllocationNode{
		ID: details.AllocationID,
	}

	bcPartition, err := getBlobbersChallengeList(balances)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"error fetching blobber challenge partition: "+err.Error())
	}

	if blobberChallLocation.PartitionLocation == nil {
		logging.Logger.Info("commit_connection",
			zap.String("blobber doesn't exists in blobber challenge partition:", t.ClientID))

		loc, err := bcPartition.AddItem(balances, pData)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error adding to blobber challenge partition: "+err.Error())
		}
		blobberChallLocation.PartitionLocation = partitions.NewPartitionLocation(loc, t.CreationDate)

		_, err = balances.InsertTrieNode(blobberChallLocation.GetKey(sc.ID), blobberChallLocation)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error saving blobber")
		}
		logging.Logger.Info("commit_connection",
			zap.String("blobber location added to blobber object:", t.ClientID))

		err = bcPartition.Save(balances)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error saving blobber challenge partition")
		}
	} else {
		err = bcPartition.UpdateItem(balances, blobberChallLocation.PartitionLocation.Location, pData)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error updating blobber challenge partition")
		}
	}

	if details.ChallengePartitionLoc == nil {
		bcAllocPartition, err := getBlobbersChallengeAllocationList(t.ClientID, balances)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error fetching blobber challenge allocation partition")
		}

		allocLoc, err := bcAllocPartition.AddItem(balances, pAllocData)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error adding to blobber challenge allocation partition")
		}
		details.ChallengePartitionLoc = partitions.NewPartitionLocation(allocLoc, t.CreationDate)

		err = bcAllocPartition.Save(balances)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error saving blobber challenge allocation partition")
		}
	}

	startRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	if blobber.RewardPartition.StartRound >= startRound && blobber.RewardPartition.Timestamp > 0 {
		parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		if err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"cannot fetch ongoing partition: %v", err)
		}

		var brn BlobberRewardNode
		if err := parts.GetItem(balances, blobber.RewardPartition.Index, blobber.ID, &brn); err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"cannot fetch blobber node item from partition: %v", err)
		}

		brn.TotalData = sizeInGB(blobber.BytesWritten)

		err = parts.UpdateItem(balances, blobber.RewardPartition.Index, &brn)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error updating blobber reward item")
		}

		err = parts.Save(balances)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"error saving ongoing blobber reward partition")
		}

	}

	// save allocation object
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"saving allocation object: %v", err)
	}

	// emit blobber update event
	if err = emitAddOrOverwriteBlobber(storageNode, sp, balances); err != nil {
		logging.Logger.Error("error emitting blobber",
			zap.Any("blobber", storageNode.ID), zap.Error(err))
	}

	// save blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"saving blobber object: %v", err)
	}

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"emitting allocation event: %v", err)
	}

	err = emitAddOrOverwriteWriteMarker(commitConnection.WriteMarker, balances, t)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"emitting write marker event: %v", err)
	}

	detailsBytes, err = json.Marshal(details.LastWriteMarker)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed", "encode last write marker failed: %v", err)
	}

	return string(detailsBytes), nil
}

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *Config, blobber *StorageNode,
	balances cstate.StateContextI,
) (err error) {
	// check for duplicates
	_, err = sc.getBlobber(blobber.ID, balances)
	if err == nil {
		return sc.updateBlobber(t, conf, blobber, balances)
	}

	//return fmt.Errorf("only owner can update blobber")

	// check params
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
	}

	blobber.LastHealthCheck = t.CreationDate // set to now

	// create stake pool
	var sp *stakePool
	sp, err = sc.getOrUpdateStakePool(conf, blobber.ID, spenum.Blobber,
		blobber.StakePoolSettings, balances)
	if err != nil {
		return fmt.Errorf("creating stake pool: %v", err)
	}

	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	data, _ := json.Marshal(dbs.DbUpdates{
		Id: t.ClientID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, t.ClientID, string(data))

	// update the list
	if err := emitAddOrOverwriteBlobber(blobber, sp, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
	}

	// update statistic
	sc.statIncr(statAddBlobber)
	sc.statIncr(statNumberOfBlobbers)

	afterInsertBlobber(blobber.ID)
	return
}
