package storagesc

import (
	"encoding/json"
	"sort"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) getValidatorsList(balances cstate.StateContextI) (*ValidatorNodes, error) {
	allValidatorsList := &ValidatorNodes{}
	allValidatorsBytes, err := balances.GetTrieNode(ALL_VALIDATORS_KEY)
	if allValidatorsBytes == nil {
		return allValidatorsList, nil
	}
	err = json.Unmarshal(allValidatorsBytes.Encode(), allValidatorsList)
	if err != nil {
		return nil, common.NewError("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	sort.SliceStable(allValidatorsList.Nodes, func(i, j int) bool {
		return allValidatorsList.Nodes[i].ID < allValidatorsList.Nodes[j].ID
	})
	return allValidatorsList, nil
}

func (ssc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances cstate.StateContextI) (string, error) {
	allValidatorsList, err := ssc.getValidatorsList(balances)
	if err != nil {
		return "", common.NewError("add_validator_failed", "Failed to get validator list."+err.Error())
	}
	newValidator := &ValidationNode{}
	err = newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	blobberBytes, _ := balances.GetTrieNode(newValidator.GetKey(ssc.ID))
	if blobberBytes == nil {
		allValidatorsList.Nodes = append(allValidatorsList.Nodes, newValidator)
		// allValidatorsBytes, _ := json.Marshal(allValidatorsList)
		balances.InsertTrieNode(ALL_VALIDATORS_KEY, allValidatorsList)
		balances.InsertTrieNode(newValidator.GetKey(ssc.ID), newValidator)

		ssc.statIncr(statAddValidator)
		ssc.statIncr(statNumberOfValidators)
	} else {
		ssc.statIncr(statUpdateValidator)
	}

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_vaidator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = ssc.getOrCreateStakePool(conf, t.ClientID,
		&newValidator.StakePoolSettings, balances)
	if err != nil {
		return "", common.NewError("add_validator_failed",
			"get or create stake pool error: "+err.Error())
	}
	if err = sp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("add_validator_failed",
			"saving stake pool error: "+err.Error())
	}

	buff := newValidator.Encode()
	return string(buff), nil
}
