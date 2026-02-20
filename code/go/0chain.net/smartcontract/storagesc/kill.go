package storagesc

import (
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/dto"
	"encoding/json"
	"errors"
	"strings"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// killBlobber
// punitively disables a blobber. it will no longer be used for new allocations
// or receive further rewards. Stakeholders will have their stakes slashed.
func (sc *StorageSmartContract) killBlobber(
	tx *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	var (
		blobber = &StorageNode{}
		sp      stakepool.AbstractStakePool
	)

	err = provider.Kill(
		input,
		tx.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash,
		func(req provider.ProviderRequest) (provider.AbstractProvider, stakepool.AbstractStakePool, error) {
			var err error
			if blobber, err = getBlobber(req.ID, balances); err != nil {
				return nil, nil, common.NewError("kill_blobber_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
			}

			bb := blobber.mustBase()
			if err := partitionsChallengeReadyBlobbersRemove(balances, bb.Id()); err != nil {
				if !strings.HasPrefix(err.Error(), partitions.ErrItemNotFoundCode) {
					return nil, nil, common.NewError("kill_blobber_failed",
						"remove blobber form challenge partition, "+err.Error())
				}
			}

			sp, err = getStakePoolAdapter(bb.Type(), bb.Id(), balances)
			if err != nil {
				return nil, nil, err
			}

			return blobber, sp, nil
		},
		func(req provider.ProviderRequest) error {
			stakePool, err := getStakePool(spenum.Blobber, req.ID, balances)
			if err != nil {
				return err
			}

			stakePool.TotalOffers = 0

			return stakePool.Save(spenum.Blobber, req.ID, balances)
		},
		balances,
	)

	//we intentionally will skip this error and return normally, to be able to refresh the provider
	if errors.Is(err, provider.AlreadyKilledError) {
		return provider.AlreadyKilledError.Error(), nil
	}

	if err != nil {
		return "", common.NewError("kill_blobber_failed", err.Error())
	}
	bb := blobber.mustBase()

	if actErr := cstate.WithActivation(balances, "hercules", func() error {
		return nil
	}, func() error {
		_, err = balances.InsertTrieNode(GetKilledIdKey(blobber.Id(), "blobber"), &datastore.NOIDField{})
		if err != nil {
			return common.NewError("kill_blobber_failed", "saving blobber kill id: "+err.Error())
		}
		return nil
	}); actErr != nil {
		return "", actErr
	}

	// Release the blobber's URL so it can be reused by new blobbers.
	// Without this, killed blobbers permanently block their URL from being registered.
	if bb.BaseURL != "" {
		_, _ = balances.DeleteTrieNode(blobber.GetUrlKey(sc.ID))
	}

	// delete the blobber from MPT if it's empty and has no stake pools
	if bb.SavedData <= 0 && len(sp.GetPools()) == 0 {

		// remove the blobber from MPT
		_, err := balances.DeleteTrieNode(blobber.GetKey())
		if err != nil {
			return "", common.NewErrorf("kill_blobber_failed", "deleting blobber: %v", err)
		}

		if err = deleteStakepool(balances, blobber.Type(), blobber.Id()); err != nil {
			return "", common.NewErrorf("kill_blobber_failed", "deleting stakepool: %v", err)
		}

		return "", nil
	}

	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return "", common.NewError("kill_blobber_failed", "saving blobber: "+err.Error())
	}
	return "", nil
}

// killValidator
// punitively disables a validator. it will no longer be used for new validations
// or receive further rewards. Stakeholders will have their stakes slashed.
func (sc *StorageSmartContract) killValidator(
	tx *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	var (
		validator = &ValidationNode{}
		sp        stakepool.AbstractStakePool
	)
	err = provider.Kill(
		input,
		tx.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash,
		func(req provider.ProviderRequest) (provider.AbstractProvider, stakepool.AbstractStakePool, error) {
			var err error
			if err = balances.GetTrieNode(provider.GetKey(req.ID), validator); err != nil {
				return nil, nil, common.NewError("kill_validator_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
			}

			validatorPartitions, err := getValidatorsList(balances)
			if err != nil {
				return nil, nil, common.NewError("kill_validator_failed",
					"failed to retrieve validator list."+err.Error())
			}

			if err := validatorPartitions.Remove(balances, validator.Id()); err != nil {
				if !strings.HasPrefix(err.Error(), partitions.ErrItemNotFoundCode) {
					return nil, nil, common.NewError("kill_validator_failed",
						"failed to remove validator from partition, "+err.Error())
				}
			}

			actErr := cstate.WithActivation(balances, "demeter", func() error {
				return nil
			}, func() error {
				return validatorPartitions.Save(balances)
			})
			if actErr != nil {
				return nil, nil, actErr
			}

			sp, err = getStakePoolAdapter(validator.Type(), validator.Id(), balances)
			if err != nil {
				return nil, nil, err
			}
			return validator, sp, nil
		},
		nil,
		balances,
	)
	if err != nil {
		return "", common.NewError("kill_validator_failed", err.Error())
	}

	if actErr := cstate.WithActivation(balances, "hercules", func() error {
		return nil
	}, func() error {
		_, err = balances.InsertTrieNode(GetKilledIdKey(validator.Id(), "validator"), &datastore.NOIDField{})
		if err != nil {
			return common.NewError("kill_validator_failed", "saving validator kill id: "+err.Error())
		}
		return nil
	}); actErr != nil {
		return "", actErr
	}

	// Release the validator's URL so it can be reused by new validators.
	if validator.BaseURL != "" {
		_, _ = balances.DeleteTrieNode(validator.GetUrlKey(sc.ID))
	}

	// delete the validator from MPT if its stake pools is empty
	if len(sp.GetPools()) == 0 {

		// remove the validator from MPT
		_, err := balances.DeleteTrieNode(validator.GetKey())
		if err != nil {
			return "", common.NewErrorf("kill_validator_failed", "deleting validator: %v", err)
		}

		if err = deleteStakepool(balances, validator.ProviderType, validator.Id()); err != nil {
			return "", common.NewErrorf("kill_validator_failed", "deleting stakepool: %v", err)
		}

		return "", nil
	}

	_, err = balances.InsertTrieNode(validator.GetKey(), validator)
	if err != nil {
		return "", common.NewError("kill_validator_failed", "saving validator: "+err.Error())
	}
	return "", nil
}

func (sc *StorageSmartContract) insertKilledProviderID(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("insert_killed_provider_id_failed",
			"can't get the config: "+err.Error())
	}

	if err := smartcontractinterface.AuthorizeWithOwner("insert_killed_provider_id", func() bool {
		return t.ClientID == conf.OwnerId
	}); err != nil {
		return "", common.NewError("insert_killed_provider_id_failed", err.Error())
	}

	storageNodeDto := &dto.StorageNodeIdField{}
	if err := json.Unmarshal(input, storageNodeDto); err != nil {
		return "", common.NewError("insert_killed_provider_id_failed",
			"malformed request: "+err.Error())
	}

	_, err = balances.InsertTrieNode(GetKilledIdKey(storageNodeDto.Id, "blobber"), &datastore.NOIDField{})
	if err != nil {
		return "", common.NewError("insert_killed_provider_id_failed",
			"saving blobber kill id: "+err.Error())
	}

	return "", nil
}
