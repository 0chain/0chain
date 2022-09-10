package storagesc

import (
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"
)

func shutdown(
	input []byte,
	providerSpecific func(providerRequest) (provider.ProviderI, error),
	pType spenum.Provider,
	balances cstate.StateContextI,
) error {
	var errCode = "shutdown_" + pType.String() + "_failed"
	var req providerRequest
	if err := req.decode(input); err != nil {
		return common.NewError(errCode, err.Error())
	}

	p, err := providerSpecific(req)
	if err != nil {
		return common.NewError(errCode, err.Error())
	}
	if p.IsShutDown() {
		return common.NewError(errCode,
			"blobber already shut down")
	}
	p.ShutDown()

	if err := p.Save(balances); err != nil {
		return common.NewError(errCode, "cannot save: "+err.Error())
	}
	if err := emitUpdateProvider(p, nil, balances); err != nil {
		return common.NewError(errCode, fmt.Sprintf("emitting event: %v", err))
	}
	return nil
}

func (ssc *StorageSmartContract) shutDownBlobber(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	return "", shutdown(
		input,
		func(req providerRequest) (provider.ProviderI, error) {
			var err error
			var blobber *StorageNode
			if blobber, err = ssc.getBlobber(req.ID, balances); err != nil {
				return nil, errors.New("can't get the blobber " + t.ClientID + ": " + err.Error())
			}
			if t.ClientID != blobber.StakePoolSettings.DelegateWallet {
				return nil, errors.New("access denied, allowed for delegate_wallet owner only")
			}
			return blobber, nil
		},
		spenum.Blobber,
		balances,
	)
}

func (ssc *StorageSmartContract) shutDownValidator(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	return "", shutdown(
		input,
		func(req providerRequest) (provider.ProviderI, error) {
			var validator = &ValidationNode{
				ID: req.ID,
			}
			if err := balances.GetTrieNode(validator.GetKey(ssc.ID), validator); err != nil {
				return nil, errors.New("can't get the validator " + t.ClientID + ": " + err.Error())
			}
			if t.ClientID != validator.StakePoolSettings.DelegateWallet {
				return nil, errors.New("access denied, allowed for delegate_wallet owner only")
			}
			return validator, nil
		},
		spenum.Blobber,
		balances,
	)
}
