package storagesc

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"0chain.net/smartcontract/provider"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	commonsc "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

const (
	blobberHealthTime = 60 * 60 // 1 Hour
	CHUNK_SIZE        = 64 * KB
)

func newBlobber(id string) *StorageNode {
	return &StorageNode{
		Provider: provider.Provider{
			ID:           id,
			ProviderType: spenum.Blobber,
		},
	}
}

func getBlobber(
	blobberID string,
	balances cstate.CommonStateContextI,
) (*StorageNode, error) {
	blobber := newBlobber(blobberID)
	err := balances.GetTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return nil, err
	}
	if blobber.ProviderType != spenum.Blobber {
		return nil, fmt.Errorf("provider is %s should be %s", blobber.ProviderType, spenum.Blobber)
	}
	return blobber, nil
}

func (_ *StorageSmartContract) getBlobber(
	blobberID string,
	balances cstate.CommonStateContextI,
) (blobber *StorageNode, err error) {
	return getBlobber(blobberID, balances)
}

func (sc *StorageSmartContract) hasBlobberUrl(blobberURL string,
	balances cstate.StateContextI) (bool, error) {
	blobber := newBlobber("")
	blobber.BaseURL = blobberURL
	err := balances.GetTrieNode(blobber.GetUrlKey(sc.ID), &datastore.NOIDField{})
	switch err {
	case nil:
		return true, nil
	case util.ErrValueNotPresent:
		return false, nil
	default:
		return false, err
	}
}

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(t *transaction.Transaction,
	conf *Config, blobber *StorageNode, savedBlobber *StorageNode,
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

	if savedBlobber.BaseURL != blobber.BaseURL {
		//if updating url
		has, err := sc.hasBlobberUrl(blobber.BaseURL, balances)
		if err != nil {
			return fmt.Errorf("could not check blobber url: %v", err)
		}

		if has {
			return fmt.Errorf("blobber url update failed, %s already used", blobber.BaseURL)
		}

		// Save url
		if blobber.BaseURL != "" {
			_, err = balances.InsertTrieNode(blobber.GetUrlKey(sc.ID), &datastore.NOIDField{})
			if err != nil {
				return fmt.Errorf("saving blobber url: " + err.Error())
			}
		}
		// remove old url
		if savedBlobber.BaseURL != "" {
			_, err = balances.DeleteTrieNode(savedBlobber.GetUrlKey(sc.ID))
			if err != nil {
				return fmt.Errorf("deleting blobber old url: " + err.Error())
			}
		}
	}

	blobber.LastHealthCheck = t.CreationDate
	aDelta := blobber.Allocated - savedBlobber.Allocated
	if aDelta != 0 {
		balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, blobber.ID, event.AllocationBlobberValueChanged{
			FieldType:    event.Allocated,
			AllocationId: "",
			BlobberId:    blobber.ID,
			Delta:        aDelta,
		})
	}
	blobber.Allocated = savedBlobber.Allocated
	blobber.SavedData = savedBlobber.SavedData

	// update statistics
	sc.statIncr(statUpdateBlobber)

	if savedBlobber.Capacity == 0 {
		sc.statIncr(statNumberOfBlobbers) // reborn, if it was "removed"
	}

	if err = validateStakePoolSettings(blobber.StakePoolSettings, conf); err != nil {
		return fmt.Errorf("invalid new stake pool settings:  %v", err)
	}

	// update stake pool settings
	var sp *stakePool
	if sp, err = sc.getStakePool(spenum.Blobber, blobber.ID, balances); err != nil {
		return fmt.Errorf("can't get stake pool:  %v", err)
	}
	stakedCapacity, err := sp.stakedCapacity(blobber.Terms.WritePrice)
	if err != nil {
		return fmt.Errorf("error calculating staked capacity: %v", err)
	}

	if blobber.Capacity < stakedCapacity {
		return fmt.Errorf("write_price_change: staked capacity(%d) exceeding total_capacity(%d)",
			stakedCapacity, blobber.Capacity)
	}

	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return common.NewError("update_blobber_settings_failed", "saving blobber: "+err.Error())
	}

	sp.Settings.MinStake = blobber.StakePoolSettings.MinStake
	sp.Settings.MaxStake = blobber.StakePoolSettings.MaxStake
	sp.Settings.ServiceChargeRatio = blobber.StakePoolSettings.ServiceChargeRatio
	sp.Settings.MaxNumDelegates = blobber.StakePoolSettings.MaxNumDelegates

	// Save stake pool
	if err = sp.Save(spenum.Blobber, blobber.ID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	if err := emitAddOrOverwriteBlobber(blobber, sp, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
	}
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

	balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, blobber.ID, event.AllocationBlobberValueChanged{
		FieldType:    event.Used,
		AllocationId: "",
		BlobberId:    blobber.ID,
		Delta:        -savedBlobber.Capacity,
	})

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

	var blobber = newBlobber(t.ClientID)
	if err = blobber.Decode(input); err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"malformed request: "+err.Error())
	}

	// set transaction information
	blobber.ID = t.ClientID
	blobber.PublicKey = t.PublicKey
	blobber.ProviderType = spenum.Blobber

	// Check delegate wallet and operational wallet are not the same
	if err := commonsc.ValidateDelegateWallet(blobber.PublicKey, blobber.StakePoolSettings.DelegateWallet); err != nil {
		return "", err
	}

	// insert, update or remove blobber
	if err = sc.insertBlobber(t, conf, blobber, balances); err != nil {
		return "", common.NewError("add_or_update_blobber_failed", err.Error())
	}

	// Save the blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return "", common.NewError("add_or_update_blobber_failed",
			"saving blobber: "+err.Error())
	}

	// Save url
	if blobber.BaseURL != "" {
		_, err = balances.InsertTrieNode(blobber.GetUrlKey(sc.ID), &datastore.NOIDField{})
		if err != nil {
			return "", common.NewError("add_or_update_blobber_failed",
				"saving blobber url: "+err.Error())
		}
	}

	balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, blobber.ID, event.AllocationBlobberValueChanged{
		FieldType:    event.MaxCapacity,
		AllocationId: "",
		BlobberId:    blobber.ID,
		Delta:        blobber.Capacity,
	})

	return string(blobber.Encode()), nil
}

// update blobber settings by owner of DelegateWallet
func (sc *StorageSmartContract) updateBlobberSettings(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI,
) (resp string, err error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get config: "+err.Error())
	}

	var updatedBlobber = newBlobber("")
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
	if sp, err = sc.getStakePool(spenum.Blobber, updatedBlobber.ID, balances); err != nil {
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

	if err = sc.updateBlobber(t, conf, updatedBlobber, blobber, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed", err.Error())
	}
	balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, blobber.ID, event.AllocationBlobberValueChanged{
		FieldType:    event.Used,
		AllocationId: "",
		BlobberId:    blobber.ID,
		Delta:        updatedBlobber.Capacity - blobber.Capacity,
	})
	blobber.Capacity = updatedBlobber.Capacity
	blobber.Terms = updatedBlobber.Terms
	blobber.StakePoolSettings = updatedBlobber.StakePoolSettings

	return string(blobber.Encode()), nil
}

func filterHealthyBlobbers(now common.Timestamp) filterBlobberFunc {
	return filterBlobberFunc(func(b *StorageNode) (kick bool, err error) {
		return b.LastHealthCheck <= (now - blobberHealthTime), nil
	})
}

func (sc *StorageSmartContract) blobberHealthCheck(t *transaction.Transaction,
	_ []byte, balances cstate.StateContextI,
) (string, error) {
	var (
		blobber  *StorageNode
		downtime uint64
		err      error
	)
	if blobber, err = sc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	downtime = common.Downtime(blobber.LastHealthCheck, t.CreationDate)
	blobber.LastHealthCheck = t.CreationDate

	emitBlobberHealthCheck(blobber, downtime, balances)

	_, err = balances.InsertTrieNode(blobber.GetKey(),
		blobber)
	if err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't Save blobber: "+err.Error())
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

	if err = commitRead.ReadMarker.VerifyClientID(); err != nil {
		return "", common.NewError("commit_blobber_read", err.Error())
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
	} else if commitRead.ReadMarker.Timestamp > alloc.Until(conf.MaxChallengeCompletionTime) {
		return "", common.NewError("commit_blobber_read",
			"late reading, allocation expired")
	}

	var details *BlobberAllocation
	for _, d := range alloc.BlobberAllocs {
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
		return "", common.NewErrorf("commit_blobber_read",
			"error fetching blobber object: %v", err)
	}

	var (
		numReads = commitRead.ReadMarker.ReadCounter - lastKnownCtr
		sizeRead = sizeInGB(numReads * CHUNK_SIZE)
		value    = currency.Coin(float64(details.Terms.ReadPrice) * sizeRead)
	)

	commitRead.ReadMarker.ReadSize = sizeRead

	// move tokens from read pool to blobber
	var rp *readPool
	if rp, err = sc.getReadPool(commitRead.ReadMarker.ClientID, balances); err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related read pool: %v", err)
	}

	var sp *stakePool
	sp, err = sc.getStakePool(spenum.Blobber, commitRead.ReadMarker.BlobberID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related stake pool: %v", err)
	}

	details.Stats.NumReads++
	alloc.Stats.NumReads++

	resp, err = rp.moveToBlobber(commitRead.ReadMarker.AllocationID,
		commitRead.ReadMarker.BlobberID, sp, value, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't transfer tokens from read pool to stake pool: %v", err)
	}
	readReward, err := currency.AddCoin(details.ReadReward, value) // stat
	if err != nil {
		return "", err
	}
	details.ReadReward = readReward

	spent, err := currency.AddCoin(details.Spent, value) // reduce min lock demand left
	if err != nil {
		return "", err
	}
	details.Spent = spent

	rewardRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	if blobber.LastRewardDataReadRound >= rewardRound {
		blobber.DataReadLastRewardRound += sizeRead
	} else {
		blobber.DataReadLastRewardRound = sizeRead
	}
	blobber.LastRewardDataReadRound = balances.GetBlock().Round

	if blobber.RewardRound.StartRound >= rewardRound && blobber.RewardRound.Timestamp > 0 {
		parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		if err != nil {
			return "", common.NewErrorf("commit_blobber_read",
				"cannot fetch ongoing partition: %v", err)
		}

		var brn BlobberRewardNode
		if err := parts.Get(balances, blobber.ID, &brn); err != nil {
			return "", common.NewErrorf("commit_blobber_read",
				"cannot fetch blobber node item from partition: %v", err)
		}

		brn.DataRead = blobber.DataReadLastRewardRound

		err = parts.UpdateItem(balances, &brn)
		if err != nil {
			return "", common.NewErrorf("commit_blobber_read",
				"error updating blobber reward item: %v", err)
		}

		err = parts.Save(balances)
		if err != nil {
			return "", common.NewErrorf("commit_blobber_read",
				"error saving ongoing blobber reward partition: %v", err)
		}
	}

	// Save pools
	err = sp.Save(spenum.Blobber, commitRead.ReadMarker.BlobberID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't save stake pool: %v", err)
	}

	if err = rp.save(sc.ID, commitRead.ReadMarker.ClientID, balances); err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't Save read pool: %v", err)
	}

	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't Save blobber: %v", err)
	}

	// Save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't Save allocation: %v", err)
	}

	// Save read marker
	_, err = balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	if err != nil {
		return "", common.NewError("saving read marker", err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	err = emitAddOrOverwriteReadMarker(commitRead.ReadMarker, balances, t)
	if err != nil {
		return "", common.NewError("saving read marker in db:", err.Error())
	}

	return // ok, the response and nil
}

// commitMoveTokens moves tokens on connection commit (on write marker),
// if data written (size > 0) -- from write pool to challenge pool, otherwise
// (delete write marker) from challenge back to write pool
func (sc *StorageSmartContract) commitMoveTokens(conf *Config, alloc *StorageAllocation,
	size int64, details *BlobberAllocation, wmTime, now common.Timestamp,
	balances cstate.StateContextI) (currency.Coin, error) {
	size = (int64(math.Ceil(float64(size) / CHUNK_SIZE))) * CHUNK_SIZE
	if size == 0 {
		return 0, nil // zero size write marker -- no tokens movements
	}

	cp, err := sc.getChallengePool(alloc.ID, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get related challenge pool: %v", err)
	}

	var move currency.Coin
	if size > 0 {
		move, err = details.upload(size, wmTime,
			alloc.restDurationInTimeUnits(wmTime, conf.TimeUnit))
		if err != nil {
			return 0, fmt.Errorf("can't move tokens to challenge pool: %v", err)
		}

		err = alloc.moveToChallengePool(cp, move)
		coin, _ := move.Int64()
		balances.EmitEvent(event.TypeStats, event.TagToChallengePool, cp.ID, event.ChallengePoolLock{
			Client:       alloc.Owner,
			AllocationId: alloc.ID,
			Amount:       coin,
		})
		if err != nil {
			return 0, fmt.Errorf("can't move tokens to challenge pool: %v", err)
		}

		movedToChallenge, err := currency.AddCoin(alloc.MovedToChallenge, move)
		if err != nil {
			return 0, err
		}
		alloc.MovedToChallenge = movedToChallenge

		spent, err := currency.AddCoin(details.Spent, move)
		if err != nil {
			return 0, err
		}
		details.Spent = spent
	} else {
		move = details.delete(-size, wmTime, alloc.restDurationInTimeUnits(wmTime, conf.TimeUnit))
		err = alloc.moveFromChallengePool(cp, move)
		coin, _ := move.Int64()
		balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
			Client:       alloc.Owner,
			AllocationId: alloc.ID,
			Amount:       coin,
		})
		if err != nil {
			return 0, fmt.Errorf("can't move tokens to write pool: %v", err)
		}
		movedBack, err := currency.AddCoin(alloc.MovedBack, move)
		if err != nil {
			return 0, err
		}
		alloc.MovedBack = movedBack

		returned, err := currency.AddCoin(details.Returned, move)
		if err != nil {
			return 0, err
		}
		details.Returned = returned
	}

	if err = cp.save(sc.ID, alloc, balances); err != nil {
		return 0, fmt.Errorf("can't Save challenge pool: %v", err)
	}

	return move, nil
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

	blobAlloc, ok := alloc.BlobberAllocsMap[t.ClientID]
	if !ok {
		return "", common.NewError("commit_connection_failed",
			"Blobber is not part of the allocation")
	}

	blobAllocBytes, err := json.Marshal(blobAlloc)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"error marshalling allocation blobber details")
	}

	if !commitConnection.WriteMarker.VerifySignature(alloc.OwnerPublicKey, balances) {
		return "", common.NewError("commit_connection_failed",
			"Invalid signature for write marker")
	}

	if blobAlloc.AllocationRoot == commitConnection.AllocationRoot && blobAlloc.LastWriteMarker != nil &&
		blobAlloc.LastWriteMarker.PreviousAllocationRoot == commitConnection.PrevAllocationRoot {
		return string(blobAllocBytes), nil
	}

	blobber, err := sc.getBlobber(blobAlloc.BlobberID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"error fetching blobber: %v", err)
	}

	if blobAlloc.AllocationRoot != commitConnection.PrevAllocationRoot {
		return "", common.NewError("commit_connection_failed",
			"Previous allocation root does not match the latest allocation root")
	}

	if blobAlloc.Stats.UsedSize+commitConnection.WriteMarker.Size >
		blobAlloc.Size {

		return "", common.NewError("commit_connection_failed",
			"Size for blobber allocation exceeded maximum")
	}

	blobAlloc.AllocationRoot = commitConnection.AllocationRoot
	blobAlloc.LastWriteMarker = commitConnection.WriteMarker
	blobAlloc.Stats.UsedSize += commitConnection.WriteMarker.Size
	blobAlloc.Stats.NumWrites++

	blobber.SavedData += commitConnection.WriteMarker.Size

	alloc.Stats.UsedSize += commitConnection.WriteMarker.Size
	alloc.Stats.NumWrites++

	balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, alloc.ID, event.AllocationBlobberValueChanged{
		FieldType:    event.Used,
		AllocationId: alloc.ID,
		BlobberId:    blobber.ID,
		Delta:        commitConnection.WriteMarker.Size,
	})

	// check time boundaries
	if commitConnection.WriteMarker.Timestamp < alloc.StartTime {
		return "", common.NewError("commit_connection_failed",
			"write marker time is before allocation created")
	}

	if commitConnection.WriteMarker.Timestamp > alloc.Expiration {
		return "", common.NewError("commit_connection_failed",
			"write marker time is after allocation expires")
	}

	movedTokens, err := sc.commitMoveTokens(conf, alloc, commitConnection.WriteMarker.Size, blobAlloc,
		commitConnection.WriteMarker.Timestamp, t.CreationDate, balances)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"moving tokens: %v", err)
	}

	if err := alloc.checkFunding(conf.CancellationCharge); err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"insufficient funds: %v", err)
	}

	if err := sc.updateBlobberChallengeReady(balances, blobAlloc, uint64(blobber.SavedData)); err != nil {
		return "", common.NewErrorf("commit_connection_failed", err.Error())
	}

	startRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	if blobber.RewardRound.StartRound >= startRound && blobber.RewardRound.Timestamp > 0 {
		parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		if err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"cannot fetch ongoing partition: %v", err)
		}

		var brn BlobberRewardNode
		if err := parts.Get(balances, blobber.ID, &brn); err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"cannot fetch blobber node item from partition: %v", err)
		}

		brn.TotalData = sizeInGB(blobber.SavedData)

		err = parts.UpdateItem(balances, &brn)
		if err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"error updating blobber reward item: %v", err)
		}

		err = parts.Save(balances)
		if err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"error saving ongoing blobber reward partition: %v", err)
		}
	}

	// Save allocation object
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"saving allocation object: %v", err)
	}

	// Save blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"saving blobber object: %v", err)
	}

	emitAddWriteMarker(t, commitConnection.WriteMarker, movedTokens, balances)

	blobAllocBytes, err = json.Marshal(blobAlloc.LastWriteMarker)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed", "encode last write marker failed: %v", err)
	}

	return string(blobAllocBytes), nil
}

// updateBlobberChallengeReady add or update blobber challenge weight or
// remove itself from challenge ready partitions if there's no data stored
func (sc *StorageSmartContract) updateBlobberChallengeReady(balances cstate.StateContextI,
	blobAlloc *BlobberAllocation, blobUsedCapacity uint64) error {
	logging.Logger.Info("commit_connection, add or update blobber challenge ready partitions",
		zap.String("blobber", blobAlloc.BlobberID))
	if blobUsedCapacity == 0 {
		// remove from challenge ready partitions if this blobber has no data stored
		return partitionsChallengeReadyBlobbersRemove(balances, blobAlloc.BlobberID)
	}

	sp, err := getStakePool(spenum.Blobber, blobAlloc.BlobberID, balances)
	if err != nil {
		return fmt.Errorf("unable to fetch blobbers stake pool: %v", err)
	}
	stakedAmount, err := sp.cleanStake()
	if err != nil {
		return fmt.Errorf("unable to clean stake pool: %v", err)
	}
	weight := uint64(stakedAmount) * blobUsedCapacity
	if err := partitionsChallengeReadyBlobberAddOrUpdate(balances, blobAlloc.BlobberID, weight); err != nil {
		return fmt.Errorf("could not add blobber to challenge ready partitions: %v", err)
	}
	return nil
}

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *Config, blobber *StorageNode,
	balances cstate.StateContextI,
) (err error) {
	savedBlobber, err := sc.getBlobber(blobber.ID, balances)
	if err == nil {
		// already exist, update it
		return sc.updateBlobber(t, conf, blobber, savedBlobber, balances)
	}

	if err != util.ErrValueNotPresent {
		return err
	}

	has, err := sc.hasBlobberUrl(blobber.BaseURL, balances)
	if err != nil {
		return fmt.Errorf("could not check blobber url: %v", err)
	}

	if has {
		return fmt.Errorf("invalid blobber, url: %s already used", blobber.BaseURL)
	}

	// check params
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
	}

	blobber.LastHealthCheck = t.CreationDate // set to now

	// create stake pool
	var sp *stakePool
	sp, err = sc.getOrCreateStakePool(conf, spenum.Blobber, blobber.ID,
		blobber.StakePoolSettings, balances)
	if err != nil {
		return fmt.Errorf("creating stake pool: %v", err)
	}

	if err = sp.Save(spenum.Blobber, blobber.ID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	staked, err := sp.stake()
	if err != nil {
		return fmt.Errorf("getting stake: %v", err)
	}

	tag, data := event.NewUpdateBlobberTotalStakeEvent(t.ClientID, staked)
	balances.EmitEvent(event.TypeStats, tag, t.ClientID, data)

	// update the list
	if err := emitAddBlobber(blobber, sp, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
	}

	// update statistic
	sc.statIncr(statAddBlobber)
	sc.statIncr(statNumberOfBlobbers)

	afterInsertBlobber(blobber.ID)
	return
}

func emitUpdateBlobberWriteStatEvent(w *WriteMarker, movedTokens currency.Coin, balances cstate.StateContextI) {
	bb := event.Blobber{
		Provider:  event.Provider{ID: w.BlobberID},
		Used:      w.Size,
		SavedData: w.Size,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberStat, bb.ID, bb)
}

func emitUpdateBlobberReadStatEvent(r *ReadMarker, balances cstate.StateContextI) {
	i, _ := big.NewFloat(r.ReadSize).Int64()
	bb := event.Blobber{
		Provider: event.Provider{ID: r.BlobberID},
		ReadData: i,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberStat, bb.ID, bb)
}
