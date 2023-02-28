package storagesc

import (
	"strings"

	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func kill() {

}

// killBlobber
// punitively disables a blobber. it will no longer be used for new allocations
// or receive further rewards. Stakeholders will have their stakes slashed.
func (_ *StorageSmartContract) killBlobber(
	tx *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	var blobber = newBlobber("")
	err = provider.Kill(
		input,
		tx.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash,
		func(req provider.ProviderRequest) (provider.Abstract, stakepool.AbstractStakePool, error) {
			var err error
			if blobber, err = getBlobber(req.ID, balances); err != nil {
				return nil, nil, common.NewError("kill_blobber_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
			}

			if err := partitionsChallengeReadyBlobbersRemove(balances, blobber.Id()); err != nil {
				if !strings.HasPrefix(err.Error(), partitions.ErrItemNotFoundCode) {
					return nil, nil, common.NewError("kill_blobber_failed",
						"remove blobber form challenge partition, "+err.Error())
				}
			}

			sp, err := getStakePoolAdapter(blobber.Type(), blobber.Id(), balances)
			if err != nil {
				return nil, nil, err
			}

			return blobber, sp, nil
		},
		balances,
	)
	if err != nil {
		return "", common.NewError("kill_blobber_failed", err.Error())
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
func (_ *StorageSmartContract) killValidator(
	tx *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	var validator = newValidator("")
	err = provider.Kill(
		input,
		tx.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash,
		func(req provider.ProviderRequest) (provider.Abstract, stakepool.AbstractStakePool, error) {
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

			sp, err := getStakePoolAdapter(validator.Type(), validator.Id(), balances)
			if err != nil {
				return nil, nil, err
			}
			return validator, sp, nil
		},
		balances,
	)
	if err != nil {
		return "", common.NewError("kill_validator_failed", err.Error())
	}
	_, err = balances.InsertTrieNode(validator.GetKey(""), validator)
	if err != nil {
		return "", common.NewError("kill_validator_failed", "saving validator: "+err.Error())
	}
	return "", nil
}
