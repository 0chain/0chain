package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/core/maths"
	"0chain.net/core/util/entitywrapper"
	"0chain.net/smartcontract/dto"

	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/provider"

	"0chain.net/chaincore/chain/state"
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
	"github.com/minio/sha256-simd"
	"go.uber.org/zap"
)

const (
	CHUNK_SIZE             = 64 * KB
	MAX_CHAIN_LENGTH       = 32
	LARGE_MAX_CHAIN_LENGTH = 128
)

func blobberKey(id string) datastore.Key {
	return provider.GetKey(id)
}

func blobberUrlKey(url, scAddress string) datastore.Key {
	return GetUrlKey(url, scAddress)
}

func getBlobber(
	blobberID string,
	balances cstate.CommonStateContextI,
) (*StorageNode, error) {
	blobber := &StorageNode{}
	err := balances.GetTrieNode(blobberKey(blobberID), blobber)
	if err != nil {
		return nil, err
	}

	b := blobber.mustBase()

	if b.ProviderType != spenum.Blobber {
		return nil, fmt.Errorf("provider is %s should be %s", b.ProviderType, spenum.Blobber)
	}
	return blobber, nil
}

func (_ *StorageSmartContract) getBlobber(
	blobberID string,
	balances cstate.CommonStateContextI,
) (blobber *StorageNode, err error) {
	return getBlobber(blobberID, balances)
}

func (ssc *StorageSmartContract) resetBlobberStats(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (resp string, err error) {
	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("update_settings",
			"can't get config: "+err.Error())
	}

	if err := smartcontractinterface.AuthorizeWithOwner("reset_blobber_stats", func() bool {
		return conf.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var fixRequest = &dto.ResetBlobberStatsDto{}
	if err = json.Unmarshal(input, fixRequest); err != nil {
		return "", common.NewError("reset_blobber_stats_failed",
			"malformed request: "+err.Error())
	}

	sp, err := getStakePool(spenum.Blobber, fixRequest.BlobberID, balances)
	if err != nil {
		return "", common.NewError("reset_blobber_stats_failed",
			"can't get related stake pool: "+err.Error())
	}

	if sp.TotalOffers != fixRequest.PrevTotalOffers {
		return "", common.NewError("reset_blobber_stats_failed",
			"blobber's total offers doesn't match with the provided values")
	}

	sp.TotalOffers = fixRequest.NewTotalOffers
	sp.IsOfferChanged = true
	if err := sp.Save(spenum.Blobber, fixRequest.BlobberID, balances); err != nil {
		return "", common.NewError("reset_blobber_stats_failed",
			"can't save stake pool: "+err.Error())
	}

	return "reset_blobber_stats_successfully", nil
}

func (sc *StorageSmartContract) hasBlobberUrl(blobberURL string,
	balances cstate.StateContextI) (bool, error) {
	err := balances.GetTrieNode(blobberUrlKey(blobberURL, sc.ID), &datastore.NOIDField{})
	switch err {
	case nil:
		return true, nil
	case util.ErrValueNotPresent:
		return false, nil
	default:
		return false, err
	}
}

func (sc *StorageSmartContract) isBlobberInKilledIds(blobberID string,
	balances cstate.StateContextI) (bool, error) {
	err := balances.GetTrieNode(GetKilledIdKey(blobberID, "blobber"), &datastore.NOIDField{})
	switch err {
	case nil:
		return true, nil
	case util.ErrValueNotPresent:
		return false, nil
	default:
		return false, err
	}
}

func (sc *StorageSmartContract) isValidatorInKilledIds(validatorID string,
	balances cstate.StateContextI) (bool, error) {
	err := balances.GetTrieNode(GetKilledIdKey(validatorID, "validator"), &datastore.NOIDField{})
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

	var currentCapacity int64
	if err := existingBlobber.mustUpdateBase(func(snb *storageNodeBase) error {
		if updateBlobber.NotAvailable != nil {
			snb.NotAvailable = *updateBlobber.NotAvailable
		}

		// storing the current capacity because existing blobber's capacity is updated.
		currentCapacity = snb.Capacity
		if updateBlobber.Capacity != nil {
			if *updateBlobber.Capacity <= 0 {
				return sc.removeBlobber(txn, updateBlobber, balances)
			}

			snb.Capacity = *updateBlobber.Capacity
		}

		// validate other params like capacity and baseUrl
		if err = validateBlobberUpdateSettings(updateBlobber, conf); err != nil {
			return fmt.Errorf("invalid blobber params: %v", err)
		}

		return nil
	}); err != nil {
		return err
	}

	if updateBlobber.BaseURL != nil && *updateBlobber.BaseURL != existingBlobber.mustBase().BaseURL {
		has, err := sc.hasBlobberUrl(*updateBlobber.BaseURL, balances)
		if err != nil {
			return fmt.Errorf("could not check blobber url: %v", err)
		}

		if has {
			return fmt.Errorf("blobber url update failed, %s already used", *updateBlobber.BaseURL)
		}

		if existingBlobber.mustBase().BaseURL != "" {
			_, err = balances.DeleteTrieNode(existingBlobber.GetUrlKey(sc.ID))
			if err != nil {
				return fmt.Errorf("deleting blobber old url: " + err.Error())
			}
		}

		if *updateBlobber.BaseURL != "" {
			//nolint:errcheck
			existingBlobber.mustUpdateBase(func(snb *storageNodeBase) error {
				snb.BaseURL = *updateBlobber.BaseURL
				return nil
			})
			_, err = balances.InsertTrieNode(existingBlobber.GetUrlKey(sc.ID), &datastore.NOIDField{})
			if err != nil {
				return fmt.Errorf("saving blobber url: " + err.Error())
			}
		}
	}

	if err := existingBlobber.mustUpdateBase(func(snb *storageNodeBase) error {
		snb.LastHealthCheck = txn.CreationDate
		return nil
	}); err != nil {
		return err
	}

	sc.statIncr(statUpdateBlobber)
	if currentCapacity == 0 {
		sc.statIncr(statNumberOfBlobbers) // reborn, if it was "removed"
	}

	if err = validateAndSaveSp(updateBlobber, existingBlobber, existingSp, conf, balances); err != nil {
		return err
	}

	// update stake pool settings if write price has changed.
	if updateBlobber.Terms != nil && updateBlobber.Terms.WritePrice != nil {
		updatedStakedCapacity, err := existingSp.stakedCapacity(*updateBlobber.Terms.WritePrice)
		if err != nil {
			return fmt.Errorf("error calculating staked capacity: %v", err)
		}

		if existingBlobber.mustBase().Allocated > updatedStakedCapacity {
			return fmt.Errorf("write_price_change: staked capacity (%d) can't go less than allocated capacity (%d)",
				updatedStakedCapacity, existingBlobber.mustBase().Allocated)
		}
	}

	if actErr := cstate.WithActivation(balances, "electra",
		func() error {
			return existingBlobber.Update(&storageNodeV2{}, func(e entitywrapper.EntityI) error {
				b := e.(*storageNodeV2)
				b.IsRestricted = updateBlobber.IsRestricted
				return nil
			})
		}, func() error {
			if actErr := cstate.WithActivation(balances, "hercules",
				func() error {
					return existingBlobber.Update(&storageNodeV3{}, func(e entitywrapper.EntityI) error {
						b := e.(*storageNodeV3)
						b.IsRestricted = updateBlobber.IsRestricted
						return nil
					})
				}, func() error {
					return existingBlobber.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
						b := e.(*storageNodeV4)
						b.IsRestricted = updateBlobber.IsRestricted

						if b.StorageVersion == nil || *b.StorageVersion == 0 {
							b.StorageVersion = updateBlobber.StorageVersion
						}
						return nil
					})
				}); actErr != nil {
				return fmt.Errorf("error updating blobber: %v", actErr)
			}
			return nil
		}); actErr != nil {
		return fmt.Errorf("error with activation: %v", actErr)
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
	if updatedBlobber.Terms == nil {
		return nil
	}

	return existingBlobber.mustUpdateBase(func(b *storageNodeBase) error {
		if updatedBlobber.Terms.ReadPrice != nil {
			if err := validateReadPrice(*updatedBlobber.Terms.ReadPrice, conf); err != nil {
				return fmt.Errorf("invalid blobber terms: %v", err)
			}
			b.Terms.ReadPrice = *updatedBlobber.Terms.ReadPrice
		}

		if updatedBlobber.Terms.WritePrice != nil {
			if err := validateWritePrice(*updatedBlobber.Terms.WritePrice, conf); err != nil {
				return fmt.Errorf("invalid blobber terms: %v", err)
			}
			b.Terms.WritePrice = *updatedBlobber.Terms.WritePrice
		}
		return nil
	})
}

func validateAndSaveSp(
	updateBlobber *dto.StorageDtoNode,
	existingBlobber *StorageNode,
	existingSp *stakePool,
	conf *Config,
	balances cstate.StateContextI,
) error {
	if updateBlobber.StakePoolSettings == nil {
		return nil
	}

	return existingBlobber.mustUpdateBase(func(b *storageNodeBase) error {
		if updateBlobber.StakePoolSettings.DelegateWallet != nil {
			existingSp.Settings.DelegateWallet = *updateBlobber.StakePoolSettings.DelegateWallet
			b.StakePoolSettings.DelegateWallet = *updateBlobber.StakePoolSettings.DelegateWallet
		}

		if updateBlobber.StakePoolSettings.ServiceChargeRatio != nil {
			existingSp.Settings.ServiceChargeRatio = *updateBlobber.StakePoolSettings.ServiceChargeRatio
			b.StakePoolSettings.ServiceChargeRatio = *updateBlobber.StakePoolSettings.ServiceChargeRatio
		}

		if updateBlobber.StakePoolSettings.MaxNumDelegates != nil {
			existingSp.Settings.MaxNumDelegates = *updateBlobber.StakePoolSettings.MaxNumDelegates
			b.StakePoolSettings.MaxNumDelegates = *updateBlobber.StakePoolSettings.MaxNumDelegates
		}

		if err := validateStakePoolSettings(b.StakePoolSettings, conf, balances); err != nil {
			return fmt.Errorf("invalid new stake pool settings:  %v", err)
		}
		return nil
	})
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
	if savedBlobber.mustBase().Capacity > 0 {
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

	blobber := &StorageNode{}

	err = state.WithActivation(balances, "hercules", func() error {
		return state.WithActivation(balances, "electra", func() error {
			b := storageNodeV2{}
			if err := json.Unmarshal(input, &b); err != nil {
				return common.NewError("add_or_update_blobber_failed",
					"malformed request: "+err.Error())
			}
			blobber.SetEntity(&b)
			return nil
		}, func() error {
			b := storageNodeV3{}
			if err := json.Unmarshal(input, &b); err != nil {
				return common.NewError("add_or_update_blobber_failed",
					"malformed request: "+err.Error())
			}
			blobber.SetEntity(&b)
			return nil
		})
	}, func() error {
		b := storageNodeV4{}
		if err := json.Unmarshal(input, &b); err != nil {
			return common.NewError("add_or_update_blobber_failed",
				"malformed request: "+err.Error())
		}
		if b.ManagingWallet == nil || *b.ManagingWallet == "" {
			b.ManagingWallet = new(string)
			*b.ManagingWallet = b.StakePoolSettings.DelegateWallet
		}
		blobber.SetEntity(&b)
		return nil
	})
	if err != nil {
		return "", err
	}

	// set transaction information
	if err := blobber.mustUpdateBase(func(b *storageNodeBase) error {
		b.ID = t.ClientID
		b.PublicKey = t.PublicKey
		b.ProviderType = spenum.Blobber
		b.NotAvailable = false

		// Check delegate wallet and operational wallet are not the same
		return commonsc.ValidateDelegateWallet(b.PublicKey, b.StakePoolSettings.DelegateWallet)
	}); err != nil {
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
	if blobber.mustBase().BaseURL != "" {
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

	isDelegateWallet := txn.ClientID == existingSp.Settings.DelegateWallet
	if isDelegateWallet {
		// merge the savedBlobber and updatedBlobber fields using the deltas from the updatedBlobber and
		// emit the update blobber event to db.
		if err = sc.updateBlobber(txn, conf, updatedBlobber, blobber, existingSp, balances); err != nil {
			return "", common.NewError("update_blobber_settings_failed", err.Error())
		}
	}

	isManagingWallet := false

	actErr := cstate.WithActivation(balances, "hercules", func() error {
		return nil
	}, func() error {
		if blobber.Entity().GetVersion() == "v4" && updatedBlobber.StakePoolSettings != nil && updatedBlobber.StakePoolSettings.DelegateWallet != nil && *updatedBlobber.StakePoolSettings.DelegateWallet != "" {
			v4 := blobber.Entity().(*storageNodeV4)
			if v4.ManagingWallet != nil && *v4.ManagingWallet == txn.ClientID {
				isManagingWallet = true
				existingSp.Settings.DelegateWallet = *updatedBlobber.StakePoolSettings.DelegateWallet
				err = existingSp.Save(spenum.Blobber, updatedBlobber.ID, balances)
				if err != nil {
					return common.NewError("update_blobber_settings_failed",
						"can't save related stake pool: "+err.Error())
				}

				if err = blobber.mustUpdateBase(func(b *storageNodeBase) error {
					b.StakePoolSettings.DelegateWallet = *updatedBlobber.StakePoolSettings.DelegateWallet
					return nil
				}); err != nil {
					return err
				}
				_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
				if err != nil {
					return common.NewError("update_blobber_settings_failed", "saving blobber: "+err.Error())
				}
				if err := emitUpdateBlobber(blobber, existingSp, balances); err != nil {
					return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
				}
			}
		}
		return nil
	})
	if actErr != nil {
		return "", actErr
	}

	if isManagingWallet || isDelegateWallet {
		return string(blobber.Encode()), nil
	}

	return "", common.NewError("update_blobber_settings_failed",
		"access denied, allowed for delegate_wallet owner only")
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
	//nolint:errcheck
	blobber.mustUpdateBase(func(b *storageNodeBase) error {
		downtime = common.Downtime(b.LastHealthCheck, t.CreationDate, conf.HealthCheckPeriod)
		b.LastHealthCheck = t.CreationDate
		return nil
	})

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
	var sa *StorageAllocation
	sa, err = sc.getAllocation(commitRead.ReadMarker.AllocationID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't get related allocation: %v", err)
	}

	alloc := sa.mustBase()

	if commitRead.ReadMarker.Timestamp < alloc.StartTime {
		return "", common.NewError("commit_blobber_read",
			"early reading, allocation not started yet")
	} else if commitRead.ReadMarker.Timestamp > alloc.Expiration {
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

	rewardRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	if err := blobber.mustUpdateBase(func(b *storageNodeBase) error {
		if b.LastRewardDataReadRound >= rewardRound {
			b.DataReadLastRewardRound += sizeRead
		} else {
			b.DataReadLastRewardRound = sizeRead
		}
		b.LastRewardDataReadRound = balances.GetBlock().Round

		if b.RewardRound.StartRound >= rewardRound && b.RewardRound.Timestamp > 0 {
			parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
			if err != nil {
				return common.NewErrorf("commit_blobber_read",
					"cannot fetch ongoing partition: %v", err)
			}

			var brn BlobberRewardNode
			if _, err := parts.Get(balances, b.ID, &brn); err != nil {
				return common.NewErrorf("commit_blobber_read",
					"cannot fetch blobber node item from partition: %v", err)
			}

			brn.DataRead = b.DataReadLastRewardRound

			err = parts.UpdateItem(balances, &brn)
			if err != nil {
				return common.NewErrorf("commit_blobber_read",
					"error updating blobber reward item: %v", err)
			}

			err = parts.Save(balances)
			if err != nil {
				return common.NewErrorf("commit_blobber_read",
					"error saving ongoing blobber reward partition: %v", err)
			}

		}
		return nil
	}); err != nil {
		return "", err
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

	_ = sa.mustUpdateBase(func(base *storageAllocationBase) error {
		alloc.deepCopy(base)
		return nil
	})

	// Save allocation
	_, err = balances.InsertTrieNode(sa.GetKey(sc.ID), sa)
	if err != nil {
		return "", common.NewErrorf("commit_blobber_read",
			"can't Save allocation: %v", err)
	}

	// Save read marker
	_, err = balances.InsertTrieNode(commitRead.GetKey(sc.ID), commitRead)
	if err != nil {
		return "", common.NewError("saving read marker", err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, sa.buildDbUpdates(balances))

	err = emitAddOrOverwriteReadMarker(commitRead.ReadMarker, balances, t)
	if err != nil {
		return "", common.NewError("saving read marker in db:", err.Error())
	}

	return // ok, the response and nil
}

// commitMoveTokens moves tokens on connection commit (on write marker),
// if data written (size > 0) -- from write pool to challenge pool, otherwise
// (delete write marker) from challenge back to write pool
func (sc *StorageSmartContract) commitMoveTokens(conf *Config, alloc *storageAllocationBase,
	size int64, details *BlobberAllocation, wmTime, now common.Timestamp,
	balances cstate.StateContextI) (currency.Coin, error) {

	if size == 0 {
		return 0, nil // zero size write marker -- no tokens movements
	}

	cp, err := sc.getChallengePool(alloc.ID, balances)
	if err != nil {
		return 0, fmt.Errorf("can't get related challenge pool: %v", err)
	}

	var move currency.Coin
	if size > 0 {
		if size < CHUNK_SIZE {
			size = CHUNK_SIZE
		}

		rdtu, err := alloc.restDurationInTimeUnits(wmTime, conf.TimeUnit)
		if err != nil {
			return 0, fmt.Errorf("could not move tokens to challenge pool: %v", err)
		}

		move, err = details.upload(size, rdtu, alloc.WritePool)
		if err != nil {
			return 0, fmt.Errorf("can't calculate move tokens to upload: %v", err)
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
	} else {
		if size > -CHUNK_SIZE {
			size = -CHUNK_SIZE
		}

		rdtu, err := alloc.restDurationInTimeUnits(wmTime, conf.TimeUnit)
		if err != nil {
			return 0, fmt.Errorf("could not move tokens from pool: %v", err)
		}

		move, err = details.delete(-size, wmTime, rdtu)
		if err != nil {
			return 0, fmt.Errorf("can't calculate move tokens to delete: %v", err)
		}

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

	commitMarkerBase := commitConnection.WriteMarker.mustBase()

	if commitMarkerBase.BlobberID != t.ClientID {
		return "", common.NewError("commit_connection_failed",
			"Invalid Blobber ID for closing connection. Write marker not for this blobber")
	}

	sa, err := sc.getAllocation(commitMarkerBase.AllocationID,
		balances)
	if err != nil {
		return "", common.NewError("commit_connection_failed",
			"can't get allocation: "+err.Error())
	}

	if actErr := cstate.WithActivation(balances, "electra", func() error { return nil }, func() error {
		if sa.Entity().GetVersion() == "v2" {
			if v2 := sa.Entity().(*storageAllocationV2); v2 != nil && v2.IsEnterprise != nil && *v2.IsEnterprise {
				return common.NewError("commit_connection_failed",
					"commit connection not allowed for enterprise enterprise allocation")
			}
		} else if sa.Entity().GetVersion() == "v3" {
			if v3 := sa.Entity().(*storageAllocationV3); v3 != nil && v3.IsEnterprise != nil && *v3.IsEnterprise {
				return common.NewError("commit_connection_failed",
					"commit connection not allowed for enterprise enterprise allocation")
			}
		}
		return nil
	}); actErr != nil {
		return "", actErr
	}

	alloc := sa.mustBase()

	if alloc.Owner != commitMarkerBase.ClientID {
		return "", common.NewError("commit_connection_failed", fmt.Sprintf("write marker has"+
			" to be by the same client as owner of the allocation %s != %s", alloc.Owner, commitMarkerBase.ClientID))
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

	if len(commitConnection.ChainData)%32 != 0 {
		return "", common.NewError("commit_connection_failed",
			"Invalid chain data")
	}

	if actErr := cstate.WithActivation(balances, "hermes", func() error {
		if len(commitConnection.ChainData) > (32 * MAX_CHAIN_LENGTH) {
			return common.NewError("commit_connection_failed",
				"Chain data length exceeds the maximum chainlength "+strconv.Itoa(MAX_CHAIN_LENGTH))
		}

		return nil
	}, func() error {
		if len(commitConnection.ChainData) > (32 * LARGE_MAX_CHAIN_LENGTH) {
			return common.NewError("commit_connection_failed",
				"Chain data length exceeds the maximum chainlength "+strconv.Itoa(LARGE_MAX_CHAIN_LENGTH))
		}
		return nil
	}); actErr != nil {
		return "", actErr
	}

	var (
		changeSize  int64
		blobLWMBase *writeMarkerBase
	)
	if blobAlloc.LastWriteMarker != nil {
		blobLWMBase = blobAlloc.LastWriteMarker.mustBase()
	}
	// branch logic according to ChainHash being present or not
	// Chain hash is the hash of all previous roots of WM's and ChainSize will be the size of all previous WM's
	if commitConnection.WriteMarker.GetVersion() == writeMarkerV2Version {
		wm2 := commitConnection.WriteMarker.Entity().(*writeMarkerV2)

		var (
			lastWMChainHash string
			lastWMChainSize int64
		)

		if blobAlloc.LastWriteMarker != nil && blobAlloc.LastWriteMarker.GetVersion() == "v2" {
			lastwm2 := blobAlloc.LastWriteMarker.Entity().(*writeMarkerV2)
			lastWMChainHash = lastwm2.ChainHash
			lastWMChainSize = lastwm2.ChainSize
		}

		if blobAlloc.AllocationRoot == commitConnection.AllocationRoot && blobAlloc.LastWriteMarker != nil &&
			lastWMChainHash == wm2.ChainHash {
			return string(blobAllocBytes), nil
		}

		changeSize = wm2.ChainSize
		hasher := sha256.New()
		var prevHash string
		if blobAlloc.LastWriteMarker != nil {
			changeSize -= lastWMChainSize
			prevChainHash, _ := hex.DecodeString(lastWMChainHash)
			hasher.Write(prevChainHash) //nolint:errcheck
			prevHash = lastWMChainHash
		}

		// Calculate the chain hash by hashing all the previous chain data and previous chain hash present on blockchain
		for i := 0; i < len(commitConnection.ChainData); i += 32 {
			hasher.Write(commitConnection.ChainData[i : i+32]) //nolint:errcheck
			sum := hasher.Sum(nil)
			hasher.Reset()
			hasher.Write(sum) //nolint:errcheck
		}

		// Write allocationRoot to chain hash to calculate the final chain hash which should include all the previous WM allocation roots, resulting chain hash should be same as the chain hash in WM signed by the client
		allocRootBytes, err := hex.DecodeString(commitConnection.AllocationRoot)
		if err != nil {
			return "", common.NewError("commit_connection_failed",
				"Error decoding allocation root")
		}
		hasher.Write(allocRootBytes) //nolint:errcheck

		chainHash := hex.EncodeToString(hasher.Sum(nil))
		if chainHash != wm2.ChainHash {
			return "", common.NewError("commit_connection_failed",
				fmt.Sprintf("Invalid chain hash:expected %s got %s and prevChainHash %s", wm2.ChainHash, chainHash, prevHash))
		}
	} else {
		if blobAlloc.AllocationRoot == commitConnection.AllocationRoot && blobAlloc.LastWriteMarker != nil &&
			blobLWMBase.PreviousAllocationRoot == commitConnection.PrevAllocationRoot {
			return string(blobAllocBytes), nil
		}
		changeSize = commitMarkerBase.Size
		if isRollback(commitConnection, commitMarkerBase, blobLWMBase) {
			changeSize -= blobLWMBase.Size
		} else {
			if blobAlloc.AllocationRoot != commitConnection.PrevAllocationRoot {
				return "", common.NewError("commit_connection_failed",
					"Previous allocation root does not match the latest allocation root")
			}
		}
	}

	blobber, err := sc.getBlobber(blobAlloc.BlobberID, balances)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"error fetching blobber: %v", err)
	}

	if blobber.IsKilled() || blobber.IsShutDown() {
		return "", common.NewError("commit_connection_failed",
			"blobber is killed or shutdown")
	}

	if blobAlloc.Stats.UsedSize == 0 {
		blobAlloc.LatestFinalizedChallCreatedAt = commitMarkerBase.Timestamp
		blobAlloc.LatestSuccessfulChallCreatedAt = commitMarkerBase.Timestamp
	}

	blobberAllocSizeBefore := blobAlloc.Stats.UsedSize

	if blobAlloc.Stats.UsedSize+changeSize >
		blobAlloc.Size {

		return "", common.NewError("commit_connection_failed",
			"Size for blobber allocation exceeded maximum")
	}

	blobAlloc.AllocationRoot = commitConnection.AllocationRoot
	blobAlloc.LastWriteMarker = commitConnection.WriteMarker
	blobAlloc.Stats.UsedSize += changeSize
	blobAlloc.Stats.NumWrites++
	//nolint:errcheck
	blobber.mustUpdateBase(func(b *storageNodeBase) error {
		b.SavedData += changeSize
		return nil
	})

	alloc.RefreshAllocationUsedSize()

	alloc.Stats.NumWrites++

	// check time boundaries
	if commitMarkerBase.Timestamp < alloc.StartTime {
		return "", common.NewError("commit_connection_failed",
			"write marker time is before allocation created")
	}

	if commitMarkerBase.Timestamp > alloc.Expiration {
		return "", common.NewError("commit_connection_failed",
			"write marker time is after allocation expires")
	}

	movedTokens, err := sc.commitMoveTokens(conf, alloc, changeSize, blobAlloc,
		commitMarkerBase.Timestamp, t.CreationDate, balances)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"moving tokens: %v", err)
	}

	bb := blobber.mustBase()
	sd, err := maths.ConvertToUint64(bb.SavedData)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed", "savedData is negative: %v", err)
	}
	if err := sc.updateBlobberChallengeReady(balances, blobAlloc, sd); err != nil {
		return "", common.NewErrorf("commit_connection_failed", err.Error())
	}

	if blobberAllocSizeBefore == 0 && changeSize > 0 {
		if err := partitionsBlobberAllocationsAdd(balances, blobAlloc.BlobberID, blobAlloc.AllocationID); err != nil {
			logging.Logger.Error("add_blobber_allocation_to_partitions_error",
				zap.String("blobber", blobAlloc.BlobberID),
				zap.String("allocation", blobAlloc.AllocationID),
				zap.Error(err))
			return "", fmt.Errorf("could not add blobber allocation to partitions: %v", err)
		}
	} else if blobAlloc.Stats.UsedSize == 0 && changeSize < 0 {
		if err := removeAllocationFromBlobberPartitions(balances, bb.ID, alloc.ID); err != nil {
			logging.Logger.Error("remove_blobber_allocation_from_partitions_error",
				zap.String("blobber", bb.ID),
				zap.String("allocation", alloc.ID),
				zap.Error(err))
			return "", fmt.Errorf("could not remove blobber allocation from partitions: %v", err)
		}
	} else if blobAlloc.Stats.UsedSize == 0 && commitMarkerBase.Size == 0 {
		if err := removeAllocationFromBlobberPartitions(balances, bb.ID, alloc.ID); err != nil {
			logging.Logger.Error("remove_blobber_allocation_from_partitions_error",
				zap.String("blobber", bb.ID),
				zap.String("allocation", alloc.ID),
				zap.Error(err))
			return "", fmt.Errorf("could not remove blobber allocation from partitions: %v", err)
		}
	}

	startRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	if bb.RewardRound.StartRound >= startRound && bb.RewardRound.Timestamp > 0 {
		parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		if err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"cannot fetch ongoing partition: %v", err)
		}

		var brn BlobberRewardNode
		if _, err := parts.Get(balances, bb.ID, &brn); err != nil {
			return "", common.NewErrorf("commit_connection_failed",
				"cannot fetch blobber node item from partition: %v", err)
		}

		brn.TotalData = sizeInGB(bb.SavedData)

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

	_ = sa.mustUpdateBase(func(base *storageAllocationBase) error {
		alloc.deepCopy(base)
		return nil
	})
	// Save allocation object
	_, err = balances.InsertTrieNode(sa.GetKey(sc.ID), sa)
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

	emitAddWriteMarker(t, commitConnection.WriteMarker, &storageAllocationBase{
		ID: alloc.ID,
		Stats: &StorageAllocationStats{
			UsedSize:  alloc.Stats.UsedSize,
			NumWrites: alloc.Stats.NumWrites,
		},
		MovedToChallenge: alloc.MovedToChallenge,
		MovedBack:        alloc.MovedBack,
		WritePool:        alloc.WritePool,
	}, movedTokens, changeSize, balances)

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
	if err := PartitionsChallengeReadyBlobberAddOrUpdate(balances, blobAlloc.BlobberID, stakedAmount, blobUsedCapacity); err != nil {
		return fmt.Errorf("could not add blobber to challenge ready partitions: %v", err)
	}
	return nil
}

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *Config, blobber *StorageNode,
	balances cstate.StateContextI,
) (err error) {
	bb := blobber.mustBase()
	_, err = sc.getBlobber(bb.ID, balances)
	if err == nil {
		// already exists with same id
		return fmt.Errorf("blobber already exists,with id: %s ", bb.ID)
	}

	if err != util.ErrValueNotPresent {
		return err
	}

	has, err := sc.hasBlobberUrl(bb.BaseURL, balances)
	if err != nil {
		return fmt.Errorf("could not check blobber url: %v", err)
	}
	if has {
		return fmt.Errorf("invalid blobber, url: %s already used", bb.BaseURL)
	}

	if actErr := cstate.WithActivation(balances, "hercules", func() error {
		return nil
	}, func() error {
		has, err = sc.isBlobberInKilledIds(bb.ID, balances)
		if err != nil {
			return fmt.Errorf("could not check blobber killed ids: %v", err)
		}
		if has {
			return fmt.Errorf("blobber with id: %s is killed", bb.ID)
		}
		return nil
	}); actErr != nil {
		return actErr
	}

	// check params
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
	}

	//nolint:errcheck
	blobber.mustUpdateBase(func(b *storageNodeBase) error {
		b.LastHealthCheck = t.CreationDate // set to now
		return nil
	})

	bb = blobber.mustBase()
	// create stake pool
	var sp *stakePool
	sp, err = sc.createStakePool(conf, bb.StakePoolSettings, balances)
	if err != nil {
		return fmt.Errorf("creating stake pool: %v", err)
	}

	if err = sp.Save(spenum.Blobber, bb.ID, balances); err != nil {
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

	afterInsertBlobber(bb.ID)
	return
}

func emitUpdateBlobberWriteStatEvent(w *WriteMarker, changeSize int64, balances cstate.StateContextI) {
	wmb := w.mustBase()
	bb := event.Blobber{
		Provider:  event.Provider{ID: wmb.BlobberID},
		SavedData: changeSize,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberStat, bb.ID, bb)
}

func emitUpdateBlobberReadStatEvent(r *ReadMarker, balances cstate.StateContextI) {
	i, _ := big.NewFloat(r.ReadSize * GB).Int64()
	bb := event.Blobber{
		Provider: event.Provider{ID: r.BlobberID},
		ReadData: i,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberStat, bb.ID, bb)
}

func isRollback(commitConnection BlobberCloseConnection, commitWM, lastWM *writeMarkerBase) bool {
	return commitConnection.AllocationRoot == commitConnection.PrevAllocationRoot && commitWM.Size == 0 && lastWM != nil && commitWM.Timestamp == lastWM.Timestamp && commitConnection.AllocationRoot == lastWM.PreviousAllocationRoot
}

func (sc *StorageSmartContract) updateBlobberVersion(t *transaction.Transaction, input []byte, balances cstate.StateContextI) (string, error) {
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_blobber_version_failed",
			"can't get the config: "+err.Error())
	}

	if err := smartcontractinterface.AuthorizeWithOwner("update_blobber_version", func() bool {
		return t.ClientID == conf.OwnerId
	}); err != nil {
		return "", common.NewError("update_blobber_version_failed", err.Error())
	}

	storageNodeDto := &dto.StorageNodeIdField{}
	if err := json.Unmarshal(input, storageNodeDto); err != nil {
		return "", common.NewError("update_blobber_version_failed",
			"malformed request: "+err.Error())
	}

	var (
		blobber *StorageNode
	)
	if blobber, err = sc.getBlobber(storageNodeDto.Id, balances); err != nil {
		return "", common.NewError("update_blobber_version_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	err = blobber.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
		return nil
	})
	if err != nil {
		return "", common.NewError("update_blobber_version_failed", "can't update blobber version: "+err.Error())
	}

	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return "", common.NewError("update_blobber_version_failed",
			"can't Save blobber: "+err.Error())
	}

	return string(blobber.Encode()), nil
}
