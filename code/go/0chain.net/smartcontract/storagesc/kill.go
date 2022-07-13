package storagesc

import (
	"encoding/json"
	"fmt"

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

func (ssc *StorageSmartContract) killBlobber(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var err error
	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("kill_blobber_failed",
			"can't get config: "+err.Error())
	}
	if err := smartcontractinterface.AuthorizeWithOwner("kill_blobber_failed", func() bool {
		return conf.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var req providerRequest
	if err := req.decode(input); err != nil {
		return "", common.NewError("kill_blobber_failed", err.Error())
	}

	var blobber *StorageNode
	if blobber, err = ssc.getBlobber(req.ID, balances); err != nil {
		return "", common.NewError("kill_blobber_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}
	blobber.Kill()
	if err = emitUpdateBlobber(blobber, balances); err != nil {
		return "", common.NewError("kill_blobber_failed", err.Error())
	}

	activePassedBlobberRewardPart, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
	if err != nil {
		return "", common.NewError("kill_blobber_failed",
			"cannot get all blobbers list: "+err.Error())
	}
	err = activePassedBlobberRewardPart.RemoveItem(balances, blobber.LastRewardPartition.Index, blobber.ID)

	parts, err := getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
	if err != nil {
		return "", common.NewErrorf("commit_connection_failed",
			"cannot fetch ongoing partition: %v", err)
	}
	err = parts.RemoveItem(balances, blobber.RewardPartition.Index, blobber.ID)
	if err != nil {
		return "", common.NewError("kill_blobber_failed",
			"cannot remove blobber from ongoing passed rewards partition: "+err.Error())
	}

	var sp *stakePool
	if sp, err = ssc.getStakePool(blobber.ID, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get related stake pool: "+err.Error())
	}
	sp.IsDead = true
	if err := sp.SlashFraction(
		conf.StakePool.KillSlash,
		req.ID,
		spenum.Blobber,
		balances,
	); err != nil {
		return "", common.NewError("kill_blobber_failed",
			"can't slash blobber: "+err.Error())
	}

	if err = sp.save(ssc.ID, blobber.ID, balances); err != nil {
		return "", common.NewError("kill_blobber_failed",
			fmt.Sprintf("saving stake pool: %v", err))
	}

	if _, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber); err != nil {
		return "", common.NewError("kill_blobber_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
}

func (ssc *StorageSmartContract) killValidator(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var err error
	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("kill_validator_failed",
			"can't get config: "+err.Error())
	}
	if err := smartcontractinterface.AuthorizeWithOwner("kill_validator_failed", func() bool {
		return conf.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var req providerRequest
	if err := req.decode(input); err != nil {
		return "", common.NewError("kill_validator_failed", err.Error())
	}

	var validator = &ValidationNode{
		ID: req.ID,
	}

	if err = balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("kill_validator_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}
	validator.Kill()
	err = validator.emitUpdate(balances)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emitting Validation node failed: %v", err.Error())
	}

	validatorPartitions, err := getValidatorsList(balances)
	if err != nil {
		return "", common.NewError("kill_validator_failed",
			"failed to get validator list."+err.Error())
	}
	if err := validatorPartitions.RemoveItem(balances, validator.PartitionPosition, validator.ID); err != nil {
		return "", common.NewError("kill_validator_failed",
			"failed to remove validator."+err.Error())
	}

	var sp *stakePool
	if sp, err = ssc.getStakePool(validator.ID, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get related stake pool: "+err.Error())
	}
	sp.IsDead = true
	if err := sp.SlashFraction(
		conf.StakePool.KillSlash,
		req.ID,
		spenum.Validator,
		balances,
	); err != nil {
		return "", common.NewError("kill_validator_failed",
			"can't slash validator: "+err.Error())
	}
	if err = sp.save(ssc.ID, validator.ID, balances); err != nil {
		return "", common.NewError("kill_validator_failed",
			fmt.Sprintf("saving stake pool: %v", err))
	}

	if _, err = balances.InsertTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("kill_validator_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
}
