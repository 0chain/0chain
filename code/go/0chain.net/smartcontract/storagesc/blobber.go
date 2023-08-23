package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"

	"0chain.net/core/maths"
	"0chain.net/smartcontract/dto"

	"0chain.net/smartcontract/partitions"
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
	CHUNK_SIZE = 64 * KB
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

func validateBlobberUpdateSettings(updateBlobberRequest *dto.StorageDtoNode, conf *Config) error {
	if updateBlobberRequest.Capacity != nil && *updateBlobberRequest.Capacity <= conf.MinBlobberCapacity {
		return errors.New("insufficient blobber capacity in update blobber settings")
	}

	if err := validateBaseUrl(updateBlobberRequest.BaseURL); err != nil {
		return err
	}

	return nil
}

// update existing blobber, or reborn a deleted one
func (sc *StorageSmartContract) updateBlobber(
	txn *transaction.Transaction,
	conf *Config,
	updateBlobber *dto.StorageDtoNode,
	existingBlobber *StorageNode,
	existingSp *stakePool,
	balances cstate.StateContextI,
) (err error) {
	// validate the new terms and update the existing blobber's terms
	if err = validateAndSaveTerms(updateBlobber, existingBlobber, conf); err != nil {
		return err
	}

	if err = validateAndSaveGeoLoc(updateBlobber, existingBlobber); err != nil {
		return err
	}

	if updateBlobber.NotAvailable != nil {
		existingBlobber.NotAvailable = *updateBlobber.NotAvailable
	}

	// storing the current capacity because existing blobber's capacity is updated.
	currentCapacity := existingBlobber.Capacity
	if updateBlobber.Capacity != nil {
		if *updateBlobber.Capacity <= 0 {
			return sc.removeBlobber(txn, updateBlobber, balances)
		}

		existingBlobber.Capacity = *updateBlobber.Capacity
	}

	// validate other params like capacity and baseUrl
	if err = validateBlobberUpdateSettings(updateBlobber, conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
	}

	if updateBlobber.BaseURL != nil && *updateBlobber.BaseURL != existingBlobber.BaseURL {
		has, err := sc.hasBlobberUrl(*updateBlobber.BaseURL, balances)
		if err != nil {
			return fmt.Errorf("could not check blobber url: %v", err)
		}

		if has {
			return fmt.Errorf("blobber url update failed, %s already used", *updateBlobber.BaseURL)
		}

		if existingBlobber.BaseURL != "" {
			_, err = balances.DeleteTrieNode(existingBlobber.GetUrlKey(sc.ID))
			if err != nil {
				return fmt.Errorf("deleting blobber old url: " + err.Error())
			}
		}

		if *updateBlobber.BaseURL != "" {
			existingBlobber.BaseURL = *updateBlobber.BaseURL
			_, err = balances.InsertTrieNode(existingBlobber.GetUrlKey(sc.ID), &datastore.NOIDField{})
			if err != nil {
				return fmt.Errorf("saving blobber url: " + err.Error())
			}
		}
	}

	existingBlobber.LastHealthCheck = txn.CreationDate

	// update statistics
	sc.statIncr(statUpdateBlobber)

	if currentCapacity == 0 {
		sc.statIncr(statNumberOfBlobbers) // reborn, if it was "removed"
	}

	if err = validateAndSaveSp(updateBlobber, existingBlobber, existingSp, conf); err != nil {
		return err
	}

	// update stake pool settings if write price has changed.
	if updateBlobber.Terms != nil && updateBlobber.Terms.WritePrice != nil {
		updatedStakedCapacity, err := existingSp.stakedCapacity(*updateBlobber.Terms.WritePrice)
		if err != nil {
			return fmt.Errorf("error calculating staked capacity: %v", err)
		}

		if existingBlobber.Allocated > updatedStakedCapacity {
			return fmt.Errorf("write_price_change: staked capacity (%d) can't go less than allocated capacity (%d)",
				updatedStakedCapacity, existingBlobber.Allocated)
		}
	}

	_, err = balances.InsertTrieNode(existingBlobber.GetKey(), existingBlobber)
	if err != nil {
		return common.NewError("update_blobber_settings_failed", "saving blobber: "+err.Error())
	}

	if err = existingSp.Save(spenum.Blobber, updateBlobber.ID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	// existing blobber also contain the updated fields from the update blobber request
	if err := emitUpdateBlobber(existingBlobber, existingSp, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", updateBlobber, err)
	}

	return
}

func validateAndSaveTerms(
	updatedBlobber *dto.StorageDtoNode,
	existingBlobber *StorageNode,
	conf *Config,
) error {
	if updatedBlobber.Terms != nil {
		if updatedBlobber.Terms.ReadPrice != nil {
			if err := validateReadPrice(*updatedBlobber.Terms.ReadPrice, conf); err != nil {
				return fmt.Errorf("invalid blobber terms: %v", err)
			}
			existingBlobber.Terms.ReadPrice = *updatedBlobber.Terms.ReadPrice
		}

		if updatedBlobber.Terms.WritePrice != nil {
			if err := validateWritePrice(*updatedBlobber.Terms.WritePrice, conf); err != nil {
				return fmt.Errorf("invalid blobber terms: %v", err)
			}
			existingBlobber.Terms.WritePrice = *updatedBlobber.Terms.WritePrice
		}
	}

	return nil
}

func validateAndSaveGeoLoc(
	updatedBlobberRequest *dto.StorageDtoNode,
	existingBlobber *StorageNode,
) error {
	if updatedBlobberRequest.Geolocation != nil {
		if updatedBlobberRequest.Geolocation.Latitude != nil {
			existingBlobber.Geolocation.Latitude = *updatedBlobberRequest.Geolocation.Latitude
		}

		if updatedBlobberRequest.Geolocation.Longitude != nil {
			existingBlobber.Geolocation.Longitude = *updatedBlobberRequest.Geolocation.Longitude
		}
	}

	if err := existingBlobber.Geolocation.validate(); err != nil {
		return err
	}

	return nil
}

func validateAndSaveSp(
	updateBlobber *dto.StorageDtoNode,
	existingBlobber *StorageNode,
	existingSp *stakePool,
	conf *Config,
) error {
	if updateBlobber.StakePoolSettings != nil {
		if updateBlobber.StakePoolSettings.DelegateWallet != nil {
			existingSp.Settings.DelegateWallet = *updateBlobber.StakePoolSettings.DelegateWallet
			existingBlobber.StakePoolSettings.DelegateWallet = *updateBlobber.StakePoolSettings.DelegateWallet
		}

		if updateBlobber.StakePoolSettings.ServiceChargeRatio != nil {
			existingSp.Settings.ServiceChargeRatio = *updateBlobber.StakePoolSettings.ServiceChargeRatio
			existingBlobber.StakePoolSettings.ServiceChargeRatio = *updateBlobber.StakePoolSettings.ServiceChargeRatio
		}

		if updateBlobber.StakePoolSettings.MaxNumDelegates != nil {
			existingSp.Settings.MaxNumDelegates = *updateBlobber.StakePoolSettings.MaxNumDelegates
			existingBlobber.StakePoolSettings.MaxNumDelegates = *updateBlobber.StakePoolSettings.MaxNumDelegates
		}

		if err := validateStakePoolSettings(existingBlobber.StakePoolSettings, conf); err != nil {
			return fmt.Errorf("invalid new stake pool settings:  %v", err)
		}
	}

	return nil
}

// remove blobber (when a blobber provides capacity = 0)
func (sc *StorageSmartContract) removeBlobber(t *transaction.Transaction,
	blobber *dto.StorageDtoNode, balances cstate.StateContextI,
) (err error) {
	// get saved blobber
	savedBlobber, err := sc.getBlobber(blobber.ID, balances)
	if err != nil {
		return fmt.Errorf("can't get or decode saved blobber: %v", err)
	}

	// set to zero explicitly, for "direct" calls
	var zeroCapacity int64 = 0
	blobber.Capacity = &zeroCapacity

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

// only use this function to add blobber(for update call updateBlobberSettings)
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
	blobber.NotAvailable = false

	// Check delegate wallet and operational wallet are not the same
	if err := commonsc.ValidateDelegateWallet(blobber.PublicKey, blobber.StakePoolSettings.DelegateWallet); err != nil {
		return "", err
	}

	// insert blobber
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

	return string(blobber.Encode()), nil
}

// update blobber settings by owner of DelegateWallet
func (sc *StorageSmartContract) updateBlobberSettings(txn *transaction.Transaction,
	input []byte, balances cstate.StateContextI,
) (resp string, err error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get config: "+err.Error())
	}

	var updatedBlobber = new(dto.StorageDtoNode)
	if err = json.Unmarshal(input, updatedBlobber); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"malformed request: "+err.Error())
	}

	var blobber *StorageNode
	if blobber, err = sc.getBlobber(updatedBlobber.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get the blobber: "+err.Error())
	}

	var existingSp *stakePool
	if existingSp, err = sc.getStakePool(spenum.Blobber, updatedBlobber.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get related stake pool: "+err.Error())
	}

	if existingSp.Settings.DelegateWallet == "" {
		return "", common.NewError("update_blobber_settings_failed",
			"blobber's delegate_wallet is not set")
	}

	if txn.ClientID != existingSp.Settings.DelegateWallet {
		return "", common.NewError("update_blobber_settings_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	// merge the savedBlobber and updatedBlobber fields using the deltas from the updatedBlobber and
	// emit the update blobber event to db.
	if err = sc.updateBlobber(txn, conf, updatedBlobber, blobber, existingSp, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed", err.Error())
	}

	return string(blobber.Encode()), nil
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
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf("blobber_health_check_failed",
			"cannot get config: %v", err)
	}
	downtime = common.Downtime(blobber.LastHealthCheck, t.CreationDate, conf.HealthCheckPeriod)
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
		numReads = commitRead.ReadMarker.ReadCounter - lastKnownCtr // todo check if it can be negative
		sizeRead = sizeInGB(numReads * CHUNK_SIZE)
		value    = currency.Coin(float64(details.Terms.ReadPrice) * sizeRead)
	)

	commitRead.ReadMarker.ReadSize = sizeRead

	// move tokens from read pool to blobber
	rp, err := sc.getReadPool(commitRead.ReadMarker.ClientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related read pool: %v", err)
	}
	if err == util.ErrValueNotPresent || rp == nil {
		rp = new(readPool)
		if err = rp.save(sc.ID, commitRead.ReadMarker.ClientID, balances); err != nil {
			return "", common.NewError("new_read_pool_failed", err.Error())
		}
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

	// updates the readpool table
	balances.EmitEvent(event.TypeStats, event.TagUpdateReadpool, commitRead.ReadMarker.ClientID, event.ReadPool{
		UserID:  commitRead.ReadMarker.ClientID,
		Balance: rp.Balance,
	})

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

	logging.Logger.Info("commitMoveTokens", zap.Any("size", size), zap.Any("alloc", alloc))

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
		rdtu, err := alloc.restDurationInTimeUnits(wmTime, conf.TimeUnit)
		if err != nil {
			return 0, fmt.Errorf("could not move tokens to challenge pool: %v", err)
		}

		move, err = details.upload(size, wmTime, rdtu)
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
		logging.Logger.Info("Jayash negative tokens", zap.Any("size", size), zap.Any("move", move), zap.Any("alloc", alloc))
		rdtu, err := alloc.restDurationInTimeUnits(wmTime, conf.TimeUnit)
		if err != nil {
			return 0, fmt.Errorf("could not move tokens from pool: %v", err)
		}

		move = details.delete(-size, wmTime, rdtu)
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

		logging.Logger.Info("Jayash negative tokens", zap.Any("size", size), zap.Any("move", move), zap.Any("alloc", alloc.MovedBack), zap.Any("alloc", alloc))
	}

	logging.Logger.Info("commitMoveTokens", zap.Any("size", size), zap.Any("move", move), zap.Any("alloc", alloc.MovedBack))

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

	logging.Logger.Info("commitBlobberConnection", zap.Any("commitConnection", commitConnection.WriteMarker.Size))

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

	blobberAllocSizeBefore := blobAlloc.Stats.UsedSize
	if isRollback(commitConnection, blobAlloc.LastWriteMarker) {
		changeSize := blobAlloc.LastWriteMarker.Size
		blobAlloc.AllocationRoot = commitConnection.AllocationRoot
		blobAlloc.LastWriteMarker = commitConnection.WriteMarker
		blobAlloc.Stats.UsedSize = blobAlloc.Stats.UsedSize - changeSize
		// TODO: check if this is correct
		blobAlloc.Stats.NumWrites++
		blobber.SavedData -= changeSize
		alloc.Stats.UsedSize -= int64(float64(changeSize) * float64(alloc.DataShards) / float64(alloc.DataShards+alloc.ParityShards))

		alloc.Stats.NumWrites++
	} else {

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
		alloc.Stats.UsedSize += int64(float64(commitConnection.WriteMarker.Size) * float64(alloc.DataShards) / float64(alloc.DataShards+alloc.ParityShards))

		alloc.Stats.NumWrites++

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

	sd, err := maths.ConvertToUint64(blobber.SavedData)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed", "savedData is negative: %v", err)
	}
	if err := sc.updateBlobberChallengeReady(balances, blobAlloc, sd); err != nil {
		return "", common.NewErrorf("commit_connection_failed", err.Error())
	}

	if blobberAllocSizeBefore == 0 && commitConnection.WriteMarker.Size > 0 {
		if err := partitionsBlobberAllocationsAdd(balances, blobAlloc.BlobberID, blobAlloc.AllocationID); err != nil {
			logging.Logger.Error("add_blobber_allocation_to_partitions_error",
				zap.String("blobber", blobAlloc.BlobberID),
				zap.String("allocation", blobAlloc.AllocationID),
				zap.Error(err))
			return "", fmt.Errorf("could not add blobber allocation to partitions: %v", err)
		}
	} else if blobAlloc.Stats.UsedSize == 0 && commitConnection.WriteMarker.Size < 0 {
		if err := removeAllocationFromBlobberPartitions(balances, blobber.ID, alloc.ID); err != nil {
			logging.Logger.Error("remove_blobber_allocation_from_partitions_error",
				zap.String("blobber", blobber.ID),
				zap.String("allocation", alloc.ID),
				zap.Error(err))
			return "", fmt.Errorf("could not remove blobber allocation from partitions: %v", err)
		}
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

	emitAddWriteMarker(t, commitConnection.WriteMarker, &StorageAllocation{
		ID: alloc.ID,
		Stats: &StorageAllocationStats{
			UsedSize:  alloc.Stats.UsedSize,
			NumWrites: alloc.Stats.NumWrites,
		},
		MovedToChallenge: alloc.MovedToChallenge,
		MovedBack:        alloc.MovedBack,
		WritePool:        alloc.WritePool,
	}, movedTokens, balances)

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
		err := partitionsChallengeReadyBlobbersRemove(balances, blobAlloc.BlobberID)
		if err != nil && !partitions.ErrItemNotFound(err) {
			return err
		}
		return nil
	}

	sp, err := getStakePool(spenum.Blobber, blobAlloc.BlobberID, balances)
	if err != nil {
		return fmt.Errorf("unable to fetch blobbers stake pool: %v", err)
	}
	stakedAmount, err := sp.stake()
	if err != nil {
		return fmt.Errorf("unable to total stake pool: %v", err)
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
	_, err = sc.getBlobber(blobber.ID, balances)
	if err == nil {
		// already exists with same id
		return fmt.Errorf("blobber already exists,with id: %s ", blobber.ID)
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
	sp, err = sc.createStakePool(conf, blobber.StakePoolSettings)
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

func isRollback(commitConnection BlobberCloseConnection, lastWM *WriteMarker) bool {
	return commitConnection.AllocationRoot == commitConnection.PrevAllocationRoot && commitConnection.WriteMarker.Size == 0 && lastWM != nil && commitConnection.WriteMarker.Timestamp == lastWM.Timestamp && commitConnection.AllocationRoot == lastWM.PreviousAllocationRoot
}
