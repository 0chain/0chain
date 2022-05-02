package storagesc

import (
	"encoding/json"

	state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
)

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances state.StateContextI) (string, error) {
	newValidator := &ValidationNode{}
	err := newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey

	tmp := &ValidationNode{}
	err = balances.GetTrieNode(newValidator.GetKey(sc.ID), tmp)
	switch err {
	case nil:
		sc.statIncr(statUpdateValidator)
	case util.ErrValueNotPresent:
		_, err = sc.getBlobber(newValidator.ID, balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"new validator id does not match a registered blobber: "+err.Error())
		}

		validatorPartitions, err := getValidatorsList(balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"Failed to get validator list."+err.Error())
		}

		_, err = validatorPartitions.AddItem(
			balances,
			&ValidationPartitionNode{
				Id:  t.ClientID,
				Url: newValidator.BaseURL,
			})
		if err != nil {
			return "", err
		}

		if err := validatorPartitions.Save(balances); err != nil {
			return "", err
		}

		_, err = balances.InsertTrieNode(newValidator.GetKey(sc.ID), newValidator)
		if err != nil {
			return "", err
		}

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
	data, _ := json.Marshal(dbs.DbUpdates{
		Id: t.ClientID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, t.ClientID, string(data))

	err = emitAddOrOverwriteValidatorTable(newValidator, balances, t)
	if err != nil {
		return "", common.NewErrorf("add_validator_failed", "emmiting Validation node failed: %v", err.Error())
	}

	buff := newValidator.Encode()
	return string(buff), nil
}
