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

			// remove killed blobber from list of blobbers to receive rewards
			if blobber.LastRewardPartition.valid() {
				activePassedBlobberRewardPart, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
				if err != nil {
					return nil, nil, common.NewError("kill_blobber_failed",
						"cannot get all blobbers list: "+err.Error())
				}
				err = activePassedBlobberRewardPart.RemoveItem(balances, blobber.LastRewardPartition.Index, blobber.ID)
				if err != nil {
					return nil, nil, common.NewError("kill_blobber_failed",
						"cannot remove blobber from active passed rewards partition: "+err.Error())
				}
			}

			if blobber.RewardPartition.valid() {
				parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
				if err != nil {
					return nil, nil, common.NewErrorf("kill_blobber_failed",
						"cannot fetch ongoing partition: %v", err)
				}
				err = parts.RemoveItem(balances, blobber.RewardPartition.Index, blobber.ID)
				if err != nil {
					return nil, nil, common.NewError("kill_blobber_failed",
						"cannot remove blobber from ongoing passed rewards partition: "+err.Error())
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
			var validator = newValidatorNode(req.ID)
			if err = balances.GetTrieNode(provider.GetKey(req.ID), validator); err != nil {
				return nil, nil, common.NewError("kill_validator_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
			}

			validatorPartitions, err := getValidatorsList(balances)
			if err != nil {
				return nil, nil, common.NewError("kill_validator_failed",
					"failed to get validator list."+err.Error())
			}
			if err := validatorPartitions.RemoveItem(balances, validator.PartitionPosition, validator.ID); err != nil {
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
