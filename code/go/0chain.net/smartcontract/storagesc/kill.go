package storagesc

import (
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func kill() {

}

func (_ *StorageSmartContract) killBlobber(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	err = provider.Kill(
		input,
		t.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash,
		func(req provider.ProviderRequest) (provider.Abstract, stakepool.AbstractStakePool, error) {
			var err error
			var blobber *StorageNode
			if blobber, err = getBlobber(req.ID, balances); err != nil {
				return nil, nil, common.NewError("kill_blobber_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
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
		return "", common.NewError("kill_blobber_failed", err.Error())
	}
	return "", nil
}

func (_ *StorageSmartContract) killValidator(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	err = provider.Kill(
		input,
		t.ClientID,
		conf.OwnerId,
		conf.StakePool.KillSlash,
		func(req provider.ProviderRequest) (provider.Abstract, stakepool.AbstractStakePool, error) {
			var err error
			var validator = newValidator(req.ID)
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
				return nil, nil, common.NewError("kill_validator_failed",
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
		return "", common.NewError("kill_validator_failed", err.Error())
	}
	return "", nil
}
