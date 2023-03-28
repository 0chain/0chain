package storagesc

import (
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// shutdownBlobber
// shuts down the blobber: It is no longer available for new allocations
// but its existing commitments will still be upheld.
func (_ *StorageSmartContract) shutdownBlobber(
	tx *transaction.Transaction,
	_ []byte,
	balances cstate.StateContextI,
) (string, error) {
	var blobber = newBlobber("")
	err := provider.ShutDown(
		tx.ClientID,
		func() (provider.AbstractProvider, stakepool.AbstractStakePool, error) {
			var err error
			if blobber, err = getBlobber(tx.ClientID, balances); err != nil {
				return nil, nil, common.NewError("shutdown_blobber_failed",
					"can't get the blobber "+tx.ClientID+": "+err.Error())
			}

			if err := partitionsChallengeReadyBlobbersRemove(balances, blobber.Id()); err != nil {
				return nil, nil, err
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
		return "", common.NewError("shutdown_blobber_failed", err.Error())
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
	_ []byte,
	balances cstate.StateContextI,
) (string, error) {
	var validator = newValidator("")
	err := provider.ShutDown(
		tx.ClientID,
		func() (provider.AbstractProvider, stakepool.AbstractStakePool, error) {
			var err error
			if err = balances.GetTrieNode(provider.GetKey(tx.ClientID), validator); err != nil {
				return nil, nil, common.NewError("shutdown_validator_failed",
					"can't get the blobber "+tx.ClientID+": "+err.Error())
			}

			validatorPartitions, err := getValidatorsList(balances)
			if err != nil {
				return nil, nil, common.NewError("shutdown_validator_failed",
					"failed to retrieve validator list."+err.Error())
			}

			if err := validatorPartitions.Remove(balances, validator.Id()); err != nil {
				return nil, nil, common.NewError("shutdown_validator_failed",
					"failed to remove validator."+err.Error())
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
		return "", common.NewError("shutdown_validator_failed", err.Error())
	}
	_, err = balances.InsertTrieNode(validator.GetKey(""), validator)
	if err != nil {
		return "", common.NewError("shutdown_validator_failed", "saving validator: "+err.Error())
	}
	return "", nil
}
