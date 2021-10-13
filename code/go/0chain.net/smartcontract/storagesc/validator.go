package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"
)

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

func (sc *StorageSmartContract) addValidator(
	t *transaction.Transaction,
	input []byte,
	balances c_state.StateContextI,
) (string, error) {
	var newValidator InputValidationNode
	if err := newValidator.Decode(input); err != nil {
		return "", err
	}
	_, err := balances.GetTrieNode(getValidatorKey(sc.ID, t.ClientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("add_validator_failed",
				"Failed to get validator."+err.Error())
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
		sc.statIncr(statAddValidator)
		sc.statIncr(statNumberOfValidators)
	} else {
		sc.statIncr(statUpdateValidator)
	}

	var conf *scConfig
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_validator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = sc.getOrCreateStakePool(conf, t.ClientID,
		&newValidator.StakePoolSettings, balances)
	if err != nil {
		return "", common.NewError("add_validator_failed",
			"get or create stake pool error: "+err.Error())
	}
	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("add_validator_failed",
			"saving stake pool error: "+err.Error())
	}

	return string(newValidator.Encode()), nil
}
