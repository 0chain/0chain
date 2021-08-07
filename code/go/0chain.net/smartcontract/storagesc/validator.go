package storagesc

import (
	"encoding/json"
	"sort"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"github.com/0chain/errors"
)

func (sc *StorageSmartContract) getValidatorsList(balances c_state.StateContextI) (*ValidatorNodes, error) {
	allValidatorsList := &ValidatorNodes{}
	allValidatorsBytes, err := balances.GetTrieNode(ALL_VALIDATORS_KEY)
	if allValidatorsBytes == nil {
		return allValidatorsList, nil
	}
	err = json.Unmarshal(allValidatorsBytes.Encode(), allValidatorsList)
	if err != nil {
		return nil, errors.New("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	sort.SliceStable(allValidatorsList.Nodes, func(i, j int) bool {
		return allValidatorsList.Nodes[i].ID < allValidatorsList.Nodes[j].ID
	})
	return allValidatorsList, nil
}

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	allValidatorsList, err := sc.getValidatorsList(balances)
	if err != nil {
		return "", errors.Wrap(err, errors.New("add_validator_failed", "Failed to get validator list").Error())
	}
	newValidator := &ValidationNode{}
	err = newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	blobberBytes, _ := balances.GetTrieNode(newValidator.GetKey(sc.ID))
	if blobberBytes == nil {
		allValidatorsList.Nodes = append(allValidatorsList.Nodes, newValidator)
		// allValidatorsBytes, _ := json.Marshal(allValidatorsList)
		balances.InsertTrieNode(ALL_VALIDATORS_KEY, allValidatorsList)
		balances.InsertTrieNode(newValidator.GetKey(sc.ID), newValidator)

		sc.statIncr(statAddValidator)
		sc.statIncr(statNumberOfValidators)
	} else {
		sc.statIncr(statUpdateValidator)
	}

	var conf *scConfig
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", errors.Newf("add_vaidator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = sc.getOrCreateStakePool(conf, t.ClientID,
		&newValidator.StakePoolSettings, balances)
	if err != nil {
		return "", errors.Wrap(err, errors.New("add_validator_failed",
			"get or create stake pool error").Error())

	}
	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return "", errors.Wrap(err, errors.New("add_validator_failed",
			"saving stake pool error").Error())

	}

	buff := newValidator.Encode()
	return string(buff), nil
}
