package storagesc

import (
	"errors"
	"strings"

	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// shutdownBlobber
// shuts down the blobber: It is no longer available for new allocations
// but its existing commitments will still be upheld.
func (_ *StorageSmartContract) shutdownBlobber(
	tx *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var (
		blobber = &StorageNode{}
		sp      stakepool.AbstractStakePool
	)

	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewErrorf("shutdown_blobber_failed", "can't get config: %v", err)
	}

	err = provider.ShutDown(
		input,
		tx.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash/2,
		func(req provider.ProviderRequest) (provider.AbstractProvider, string, stakepool.AbstractStakePool, error) {
			var err error
			if blobber, err = getBlobber(req.ID, balances); err != nil {
				return nil, "", nil, common.NewError("shutdown_blobber_failed",
					"can't get the blobber "+tx.ClientID+": "+err.Error())
			}

			var authWallet string

			if jasonActErr := cstate.WithActivation(balances, "jason", func() error {
				authWallet = blobber.mustBase().StakePoolSettings.DelegateWallet
				return nil
			}, func() error {
				if blobber.Entity().GetVersion() == "v4" {
					v4 := blobber.Entity().(*storageNodeV4)
					if v4.ManagingWallet != nil {
						authWallet = *v4.ManagingWallet
					}
				}
				if authWallet == "" {
					authWallet = sp.GetSettings().DelegateWallet
				}
				return nil
			}); jasonActErr != nil {
				return nil, "", nil, jasonActErr
			}

			if err := partitionsChallengeReadyBlobbersRemove(balances, blobber.Id()); err != nil {
				if !strings.HasPrefix(err.Error(), partitions.ErrItemNotFoundCode) {
					return nil, "", nil, common.NewError("shutdown_blobber_failed",
						"remove blobber form challenge partition, "+err.Error())
				}
			}

			sp, err = getStakePoolAdapter(blobber.Type(), blobber.Id(), balances)
			if err != nil {
				return nil, "", nil, err
			}

			return blobber, authWallet, sp, nil
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
	if errors.Is(err, provider.AlreadyShutdownError) {
		return provider.AlreadyShutdownError.Error(), nil
	}

	if err != nil {
		return "", common.NewError("shutdown_blobber_failed", err.Error())
	}

	if blobber.mustBase().SavedData <= 0 && len(sp.GetPools()) == 0 {
		_, err = balances.DeleteTrieNode(blobber.GetKey())
		if err != nil {
			return "", common.NewErrorf("shutdown_blobber_failed", "deleting blobber: %v", err)
		}

		if err = deleteStakepool(balances, blobber.Type(), blobber.Id()); err != nil {
			return "", common.NewErrorf("shutdown_blobber_failed", "deleting stakepool: %v", err)
		}

		return "", nil
	}

	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return "", common.NewError("shutdown_blobber_failed", "saving blobber: "+err.Error())
	}
	return "", nil
}

// shutdownValidator
// shuts down the blobber: It is no longer available for validating any new challenges
// but its existing commitments will still be upheld.
func (_ *StorageSmartContract) shutdownValidator(
	tx *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var (
		validator = newValidator("")
		sp        stakepool.AbstractStakePool
	)

	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewErrorf("shutdown_validator_failed", "can't get config: %v", err)
	}

	var refreshProviderFunc func(req provider.ProviderRequest) error
	if actErr := cstate.WithActivation(balances, "hercules", func() error {
		refreshProviderFunc = func(req provider.ProviderRequest) error {
			stakePoolObject, err := getStakePool(spenum.Blobber, req.ID, balances)
			if err != nil {
				return err
			}

			stakePoolObject.TotalOffers = 0

			return stakePoolObject.Save(spenum.Blobber, req.ID, balances)
		}
		return nil
	}, func() error {
		refreshProviderFunc = nil
		return nil
	}); actErr != nil {
		return "", common.NewError("shutdown_validator_failed", actErr.Error())
	}

	err = provider.ShutDown(
		input,
		tx.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash/2,
		func(req provider.ProviderRequest) (provider.AbstractProvider, string, stakepool.AbstractStakePool, error) {
			var err error
			if err = balances.GetTrieNode(provider.GetKey(req.ID), validator); err != nil {
				return nil, "", nil, common.NewError("shutdown_validator_failed",
					"can't get the blobber "+tx.ClientID+": "+err.Error())
			}

			validatorPartitions, err := getValidatorsList(balances)
			if err != nil {
				return nil, "", nil, common.NewError("shutdown_validator_failed",
					"failed to retrieve validator list."+err.Error())
			}

			if err := validatorPartitions.Remove(balances, validator.Id()); err != nil {
				if !strings.HasPrefix(err.Error(), partitions.ErrItemNotFoundCode) {
					return nil, "", nil, common.NewErrorf("shutdown_validator_failed",
						"failed to remove validator: %v", err)
				}
			}

			actErr := cstate.WithActivation(balances, "demeter", func() error {
				return nil
			}, func() error {
				return validatorPartitions.Save(balances)
			})
			if actErr != nil {
				return nil, "", nil, actErr
			}

			sp, err = getStakePoolAdapter(validator.Type(), validator.Id(), balances)
			if err != nil {
				return nil, "", nil, err
			}
			return validator, sp.GetSettings().DelegateWallet, sp, nil
		},
		refreshProviderFunc,
		balances,
	)

	if err != nil {
		return "", common.NewError("shutdown_validator_failed", err.Error())
	}

	if len(sp.GetPools()) == 0 {
		_, err = balances.DeleteTrieNode(validator.GetKey())
		if err != nil {
			return "", common.NewErrorf("shutdown_validator_failed", "deleting validator: %v", err)
		}

		if err = deleteStakepool(balances, validator.ProviderType, validator.Id()); err != nil {
			return "", common.NewErrorf("shutdown_validator_failed", "deleting stakepool: %v", err)
		}

		return "", nil
	}

	_, err = balances.InsertTrieNode(validator.GetKey(), validator)
	if err != nil {
		return "", common.NewError("shutdown_validator_failed", "saving validator: "+err.Error())
	}
	return "", nil
}

func deleteStakepool(balances cstate.StateContextI, providerType spenum.Provider, providerID string) error {
	_, err := balances.DeleteTrieNode(stakePoolKey(providerType, providerID))
	return err
}
