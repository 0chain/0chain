package storagesc

import (
	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) killBlobber(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	err := provider.Kill(
		input,
		t.ClientID, "",
		func(req provider.ProviderRequest, conf *Config) (provider.ProviderI, error) {
			var err error
			var blobber *StorageNode
			if blobber, err = ssc.getBlobber(req.ID, balances); err != nil {
				return nil, common.NewError("kill_blobber_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
			}

			// remove killed blobber from list of blobbers to receive rewards
			if blobber.LastRewardPartition.valid() {
				activePassedBlobberRewardPart, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
				if err != nil {
					return nil, common.NewError("kill_blobber_failed",
						"cannot get all blobbers list: "+err.Error())
				}
				err = activePassedBlobberRewardPart.RemoveItem(balances, blobber.LastRewardPartition.Index, blobber.ID)
				if err != nil {
					return nil, common.NewError("kill_blobber_failed",
						"cannot remove blobber from active passed rewards partition: "+err.Error())
				}
			}

			if blobber.RewardPartition.valid() {
				parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
				if err != nil {
					return nil, common.NewErrorf("kill_blobber_failed",
						"cannot fetch ongoing partition: %v", err)
				}
				err = parts.RemoveItem(balances, blobber.RewardPartition.Index, blobber.ID)
				if err != nil {
					return nil, common.NewError("kill_blobber_failed",
						"cannot remove blobber from ongoing passed rewards partition: "+err.Error())
				}
			}
			return blobber, nil
		},
		balances,
	)
	if err != nil {
		return "", common.NewError("kill_blobber_failed", err.Error())
	}
	return "", nil
}

func (ssc *StorageSmartContract) killValidator(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	return "", kill(
		input, "",
		t.ClientID,
		func(req providerRequest, _ *Config) (provider.ProviderI, error) {
			var err error
			var validator = newValidatorNode(req.ID)
			if err = balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
				return nil, common.NewError("kill_validator_failed",
					"can't get the blobber "+req.ID+": "+err.Error())
			}

			validatorPartitions, err := getValidatorsList(balances)
			if err != nil {
				return nil, common.NewError("kill_validator_failed",
					"failed to get validator list."+err.Error())
			}
			if err := validatorPartitions.RemoveItem(balances, validator.PartitionPosition, validator.ID); err != nil {
				return nil, common.NewError("kill_validator_failed",
					"failed to remove validator."+err.Error())
			}
			return validator, nil
		},
		spenum.Validator,
		balances,
	)
}
