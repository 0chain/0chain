package storagesc

import (
	"encoding/json"

	"0chain.net/chaincore/smartcontractinterface"

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

func (ssc *StorageSmartContract) shutDownBlobber(
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
	blobber.ShutDown()
	if err = emitUpdateBlobber(blobber, balances); err != nil {
		return "", common.NewError("blobber_health_check_failed", err.Error())
	}
	if _, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
}

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
		return "", common.NewError("update_settings",
			"can't get config: "+err.Error())
	}
	if err := smartcontractinterface.AuthorizeWithOwner("update_settings", func() bool {
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
		return "", common.NewError("blobber_health_check_failed", err.Error())
	}
	if _, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
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

func (ssc *StorageSmartContract) shutDownValidator(
	t *transaction.Transaction,
	_ []byte,
	balances cstate.StateContextI,
) (string, error) {
	var validator = &ValidationNode{
		ID: t.ClientID,
	}
	var err error
	if err = balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}
	validator.ShutDown()
	err = validator.emitUpdate(balances)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emitting Validation node failed: %v", err.Error())
	}
	if _, err = balances.InsertTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("blobber_health_check_failed",
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
		return "", common.NewError("update_settings",
			"can't get config: "+err.Error())
	}
	if err := smartcontractinterface.AuthorizeWithOwner("update_settings", func() bool {
		return conf.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var req providerRequest
	if err := req.decode(input); err != nil {
		return "", common.NewError("kill_blobber_failed", err.Error())
	}

	var validator = &ValidationNode{
		ID: req.ID,
	}

	if err = balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't get the blobber "+t.ClientID+": "+err.Error())
	}
	validator.Kill()
	err = validator.emitUpdate(balances)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emitting Validation node failed: %v", err.Error())
	}
	if _, err = balances.InsertTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
		return "", common.NewError("blobber_health_check_failed",
			"can't save blobber: "+err.Error())
	}
	return "", nil
}
