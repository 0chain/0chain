package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
)

const allValidatorsPartitionSize = 50

func getValidatorsList(balances cstate.StateContextI) (partitions.RandPartition, error) {
	all, err := partitions.GetRandomSelector(ALL_VALIDATORS_KEY, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			ALL_VALIDATORS_KEY,
			allValidatorsPartitionSize,
			nil,
			partitions.ItemValidator,
		)
	}
	all.SetCallback(nil)
	return all, nil
}

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances cstate.StateContextI) (string, error) {
	newValidator := &ValidationNode{}
	err := newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	_, err = balances.GetTrieNode(newValidator.GetKey(sc.ID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("add_validator_failed",
				"Failed to get validator."+err.Error())
		}

		_, err = sc.getBlobber(newValidator.ID, balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"new validator id does not match a registered blobber: "+err.Error())
		}

		allValidatorsList, err := getValidatorsList(balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"Failed to get validator list."+err.Error())
		}
		_, err = allValidatorsList.Add(
			&partitions.ValidationNode{
				Id:  t.ClientID,
				Url: newValidator.BaseURL,
			}, balances,
		)
		if err != nil {
			return "", err
		}
		err = allValidatorsList.Save(balances)
		if err != nil {
			return "", err
		}

		balances.InsertTrieNode(newValidator.GetKey(sc.ID), newValidator)

		sc.statIncr(statAddValidator)
		sc.statIncr(statNumberOfValidators)
	} else {
		sc.statIncr(statUpdateValidator)
	}

	var conf *scConfig
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_vaidator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = sc.getOrUpdateStakePool(conf, t.ClientID, stakepool.Validator,
		newValidator.StakePoolSettings, balances)
	if err != nil {
		return "", common.NewError("add_validator_failed",
			"get or create stake pool error: "+err.Error())
	}
	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("add_validator_failed",
			"saving stake pool error: "+err.Error())
	}

	err = emitAddOrOverwriteValidatorTable(newValidator, balances, t)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emmiting Validation node failed: %v", err.Error())
	}

	buff := newValidator.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) updateValidatorSettings(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI,
) (resp string, err error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get config: "+err.Error())
	}

	var validators *StorageNodes
	if blobbers, err = sc.getBlobbersList(balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"failed to get blobber list: "+err.Error())
	}

	var updatedBlobber = new(StorageNode)
	if err = updatedBlobber.Decode(input); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"malformed request: "+err.Error())
	}

	var blobber *StorageNode
	sc.get
	if blobber, err = sc.getBlobber(updatedBlobber.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get the blobber: "+err.Error())
	}

	var sp *stakePool
	if sp, err = sc.getStakePool(updatedBlobber.ID, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"can't get related stake pool: "+err.Error())
	}

	if sp.Settings.DelegateWallet == "" {
		return "", common.NewError("update_blobber_settings_failed",
			"blobber's delegate_wallet is not set")
	}

	if t.ClientID != sp.Settings.DelegateWallet {
		return "", common.NewError("update_blobber_settings_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	blobber.Terms = updatedBlobber.Terms
	blobber.Capacity = updatedBlobber.Capacity

	if err = sc.updateBlobber(t, conf, blobber, blobbers, balances); err != nil {
		return "", common.NewError("update_blobber_settings_failed", err.Error())
	}

	// save all the blobbers
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, blobbers)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving all blobbers: "+err.Error())
	}

	// save blobber
	_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
	if err != nil {
		return "", common.NewError("update_blobber_settings_failed",
			"saving blobber: "+err.Error())
	}

	return string(blobber.Encode()), nil
}
