package storagesc

import (
	"encoding/json"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getValidatorsList() ([]ValidationNode, error) {
	var allValidatorsList = make([]ValidationNode, 0)
	allValidatorsBytes, err := sc.DB.GetNode(ALL_VALIDATORS_KEY)
	if err != nil {
		return nil, common.NewError("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	if allValidatorsBytes == nil {
		return allValidatorsList, nil
	}
	err = json.Unmarshal(allValidatorsBytes, &allValidatorsList)
	if err != nil {
		return nil, common.NewError("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	return allValidatorsList, nil
}

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte) (string, error) {
	allValidatorsList, err := sc.getValidatorsList()
	if err != nil {
		return "", common.NewError("add_validator_failed", "Failed to get validator list."+err.Error())
	}
	var newValidator ValidationNode
	err = newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	blobberBytes, _ := sc.DB.GetNode(newValidator.GetKey())
	if blobberBytes == nil {
		allValidatorsList = append(allValidatorsList, newValidator)
		allValidatorsBytes, _ := json.Marshal(allValidatorsList)
		sc.DB.PutNode(ALL_VALIDATORS_KEY, allValidatorsBytes)
		sc.DB.PutNode(newValidator.GetKey(), newValidator.Encode())
	}

	buff := newValidator.Encode()
	return string(buff), nil
}
