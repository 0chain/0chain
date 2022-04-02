package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool/spenum"
	"fmt"
)

const allValidatorsPartitionSize = 50

func getValidatorsList(balances c_state.StateContextI) (partitions.RandPartition, error) {
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

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	newValidator := &ValidationNode{}
	err := newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey

	tmp := &ValidationNode{}
	raw, err := balances.GetTrieNode(newValidator.GetKey(sc.ID), tmp)
	switch err {
	case nil:
		var ok bool
		if tmp, ok = raw.(*ValidationNode); !ok {
			return "", fmt.Errorf("unexpected node type")
		}
		sc.statIncr(statUpdateValidator)
	case util.ErrValueNotPresent:
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
	default:
		return "", common.NewError("add_validator_failed",
			"Failed to get validator."+err.Error())
	}

	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_vaidator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = sc.getOrUpdateStakePool(conf, t.ClientID, spenum.Validator,
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
