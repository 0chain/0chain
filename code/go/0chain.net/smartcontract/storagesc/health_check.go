package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) blobberHealthCheck(
	t *transaction.Transaction,
	_ []byte,
	balances cstate.StateContextI,
) (string, error) {
	var blobber *StorageNode
	var err error
	if blobber, err = ssc.getBlobber(t.ClientID, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	blobber.HealthCheck(t.CreationDate)

	if err = emitUpdateBlobber(blobber, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed", err.Error())
	}
	if _, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}

	return string(blobber.Encode()), nil
}

func (ssc *StorageSmartContract) validatorHealthCheck(
	t *transaction.Transaction,
	_ []byte,
	balances cstate.StateContextI,
) (string, error) {
	var validator = &ValidationNode{
		ID: t.ClientID,
	}
	if err := balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}

	validator.HealthCheck(t.CreationDate)

	err := validator.emitUpdate(balances)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emitting Validation node failed: %v", err.Error())
	}
	if _, err = balances.InsertTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}

	return "", nil
}
