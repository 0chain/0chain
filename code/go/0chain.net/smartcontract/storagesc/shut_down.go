package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) shutDownBlobber(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var req providerRequest
	if err := req.decode(input); err != nil {
		return "", common.NewError("shut_down_blobber_failed", err.Error())
	}

	var err error
	var sp *stakePool
	if sp, err = ssc.getStakePool(req.ID, balances); err != nil {
		return "", common.NewError("shut_down_blobber_failed",
			"can't get related stake pool: "+err.Error())
	}
	if t.ClientID != sp.Settings.DelegateWallet {
		return "", common.NewError("shut_down_blobber_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	var blobber *StorageNode
	if blobber, err = ssc.getBlobber(req.ID, balances); err != nil {
		return "", common.NewError("shut_down_blobber_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}
	blobber.ShutDown()
	if err = emitUpdateBlobber(blobber, balances); err != nil {
		return "", common.NewError("shut_down_blobber_failed", err.Error())
	}
	if _, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber); err != nil {
		return "", common.NewError("shut_down_blobber_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
}

func (ssc *StorageSmartContract) shutDownValidator(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var req providerRequest
	if err := req.decode(input); err != nil {
		return "", common.NewError("shut_down_validator_failed", err.Error())
	}

	var err error
	var sp *stakePool
	if sp, err = ssc.getStakePool(req.ID, balances); err != nil {
		return "", common.NewError("shut_down_validator_failed",
			"can't get related stake pool: "+err.Error())
	}
	if t.ClientID != sp.Settings.DelegateWallet {
		return "", common.NewError("shut_down_validator_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	var validator = &ValidationNode{
		ID: t.ClientID,
	}
	if err = balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("shut_down_validator_failed",
			"can't get the validator "+t.ClientID+": "+err.Error())
	}
	validator.ShutDown()
	err = validator.emitUpdate(balances)
	if err != nil {
		return "", common.NewErrorf("shut_down_validator_failed", "emitting validation node failed: %v", err.Error())
	}

	validatorPartitions, err := getValidatorsList(balances)
	if err != nil {
		return "", common.NewError("shut_down_validator_failed",
			"failed to get validator list."+err.Error())
	}
	if err := validatorPartitions.RemoveItem(balances, validator.PartitionPosition, validator.ID); err != nil {
		return "", common.NewError("shut_down_validator_failed",
			"failed to remove validator."+err.Error())
	}

	if _, err = balances.InsertTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("shut_down_validator_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
}
