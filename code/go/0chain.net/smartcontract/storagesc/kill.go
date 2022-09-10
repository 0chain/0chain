package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type providerRequest struct {
	ID string `json:"id"`
}

func (pr *providerRequest) decode(p []byte) error {
	return json.Unmarshal(p, pr)
}

func kill(
	input []byte,
	clientID string,
	providerSpecific func(providerRequest, *Config) (provider.ProviderI, error),
	pType spenum.Provider,
	balances cstate.StateContextI,
) error {
	var errCode = "kill_" + pType.String() + "_failed"
	var err error
	var conf *Config
	if conf, err = getConfig(balances); err != nil {
		return common.NewError(errCode, "can't get config: "+err.Error())
	}
	if err := smartcontractinterface.AuthorizeWithOwner(errCode, func() bool {
		return conf.OwnerId == clientID
	}); err != nil {
		return err
	}

	var req providerRequest
	if err := req.decode(input); err != nil {
		return common.NewError(errCode, err.Error())
	}

	var sp *stakePool
	if sp, err = getProviderStakePool(pType, req.ID, balances); err != nil {
		return common.NewError(errCode, "can't get related stake pool: "+err.Error())
	}

	p, err := providerSpecific(req, conf)
	if err != nil {
		return err
	}

	if p.IsKilled() {
		return common.NewError(errCode, "already killed")
	}
	p.Kill()
	if err := p.Save(balances); err != nil {
		return common.NewError(errCode, "cannot save: "+err.Error())
	}

	sp.IsDead = true
	if err := sp.SlashFraction(
		conf.StakePool.KillSlash,
		req.ID,
		pType,
		balances,
	); err != nil {
		return common.NewError(errCode, "can't slash validator: "+err.Error())
	}

	if err = sp.save(spenum.Validator, req.ID, balances); err != nil {
		return common.NewError(errCode, fmt.Sprintf("saving stake pool: %v", err))
	}

	if err := emitUpdateProvider(p, sp, balances); err != nil {
		return common.NewError(errCode, fmt.Sprintf("emitting event: %v", err))
	}

	return nil
}

func (ssc *StorageSmartContract) killBlobber(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	return "", kill(
		input,
		t.ClientID,
		func(req providerRequest, conf *Config) (provider.ProviderI, error) {
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
		spenum.Blobber,
		balances,
	)
}

func (ssc *StorageSmartContract) killValidator(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	return "", kill(
		input,
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
