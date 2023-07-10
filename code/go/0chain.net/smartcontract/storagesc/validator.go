package storagesc

import (
	"0chain.net/smartcontract/dto"
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/provider"

	state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	commonsc "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
)

const (
	validatorHealthTime = 60 * 60 // 1 hour
)

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte, balances state.StateContextI) (string, error) {
	newValidator := newValidator("")
	err := newValidator.Decode(input) //json.Unmarshal(input, &newValidator)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	newValidator.ProviderType = spenum.Validator
	newValidator.LastHealthCheck = t.CreationDate

	// Check delegate wallet and operational wallet are not the same
	if err := commonsc.ValidateDelegateWallet(newValidator.PublicKey, newValidator.StakePoolSettings.DelegateWallet); err != nil {
		return "", err
	}

	_, err = getValidator(t.ClientID, balances)
	switch err {
	case nil:
		return "", common.NewError("add_validator_failed",
			"provider already exist at id:"+t.ClientID)
	case util.ErrValueNotPresent:
		validatorPartitions, err := getValidatorsList(balances)
		if err != nil {
			return "", common.NewError("add_validator_failed",
				"Failed to get validator list."+err.Error())
		}

		err = validatorPartitions.Add(
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
			"Failed to get validator. "+err.Error())
	}

	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_vaidator",
			"can't get SC configurations: %v", err)
	}

	// create stake pool for the validator to count its rewards
	var sp *stakePool
	sp, err = sc.getOrCreateStakePool(conf, spenum.Validator, t.ClientID,
		newValidator.StakePoolSettings, balances)
	if err != nil {
		return "", common.NewError("add_validator_failed",
			"get or create stake pool error: "+err.Error())
	}
	if err = sp.Save(spenum.Validator, t.ClientID, balances); err != nil {
		return "", common.NewError("add_validator_failed",
			"saving stake pool error: "+err.Error())
	}

	if err = newValidator.emitAddOrOverwrite(sp, balances); err != nil {
		return "", common.NewErrorf("add_validator_failed", "emmiting Validation node failed: %v", err.Error())
	}

	buff := newValidator.Encode()
	return string(buff), nil
}

func newValidator(id string) *ValidationNode {
	return &ValidationNode{
		Provider: provider.Provider{
			ID:           id,
			ProviderType: spenum.Validator,
		},
	}
}

func getValidator(
	validatorID string,
	balances state.CommonStateContextI,
) (*ValidationNode, error) {
	validator := newValidator(validatorID)
	err := balances.GetTrieNode(validator.GetKey(ADDRESS), validator)
	if err != nil {
		return nil, err
	}
	if validator.ProviderType != spenum.Validator {
		return nil, fmt.Errorf("provider is %s should be %s", validator.ProviderType, spenum.Validator)
	}
	return validator, nil
}

func (_ *StorageSmartContract) getValidator(
	validatorID string,
	balances state.StateContextI,
) (validator *ValidationNode, err error) {
	return getValidator(validatorID, balances)
}

func (sc *StorageSmartContract) updateValidatorSettings(txn *transaction.Transaction, input []byte, balances state.StateContextI) (string, error) {
	// get smart contract configuration
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get config: "+err.Error())
	}

	var updatedValidator = new(dto.ValidationDtoNode)
	if err = json.Unmarshal(input, updatedValidator); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"malformed request: "+err.Error())
	}

	var existingValidator *ValidationNode
	if existingValidator, err = sc.getValidator(updatedValidator.ID, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get the validator: "+err.Error())
	}

	var existingStakePool *stakePool
	if existingStakePool, err = sc.getStakePool(spenum.Validator, updatedValidator.ID, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"can't get related stake pool: "+err.Error())
	}

	if existingStakePool.Settings.DelegateWallet == "" {
		return "", common.NewError("update_validator_settings_failed",
			"validator's delegate_wallet is not set")
	}

	if txn.ClientID != existingStakePool.Settings.DelegateWallet {
		return "", common.NewError("update_validator_settings_failed",
			"access denied, allowed for delegate_wallet owner only")
	}

	if err = sc.updateValidator(txn, conf, updatedValidator, existingValidator, balances); err != nil {
		return "", common.NewError("update_validator_settings_failed", err.Error())
	}

	// save validator
	_, err = balances.InsertTrieNode(existingValidator.GetKey(sc.ID), existingValidator)
	if err != nil {
		return "", common.NewError("update_validator_settings_failed",
			"saving validator: "+err.Error())
	}

	return string(existingValidator.Encode()), nil
}

func (sc *StorageSmartContract) hasValidatorUrl(validatorURL string,
	balances state.StateContextI) (bool, error) {
	validator := new(ValidationNode)
	validator.BaseURL = validatorURL
	err := balances.GetTrieNode(validator.GetUrlKey(sc.ID), &datastore.NOIDField{})
	switch err {
	case nil:
		return true, nil
	case util.ErrValueNotPresent:
		return false, nil
	default:
		return false, err
	}
}

// update existing validator, or reborn a deleted one
func (sc *StorageSmartContract) updateValidator(txn *transaction.Transaction,
	conf *Config, inputValidator *dto.ValidationDtoNode, savedValidator *ValidationNode,
	balances state.StateContextI,
) (err error) {
	// check params
	if err = validateBaseUrl(inputValidator.BaseURL); err != nil {
		return fmt.Errorf("invalid validator params: %v", err)
	}

	if inputValidator.BaseURL != nil && savedValidator.BaseURL != *inputValidator.BaseURL {
		has, err := sc.hasValidatorUrl(*inputValidator.BaseURL, balances)
		if err != nil {
			return fmt.Errorf("could not get validator of url: %s : %v", *inputValidator.BaseURL, err)
		}

		if has {
			return fmt.Errorf("invalid validator url update, already used")
		}

		// remove old url
		if savedValidator.BaseURL != "" {
			_, err = balances.DeleteTrieNode(savedValidator.GetUrlKey(sc.ID))
			if err != nil {
				return fmt.Errorf("deleting validator old url: " + err.Error())
			}
		}

		// save url
		if *inputValidator.BaseURL != "" {
			savedValidator.BaseURL = *inputValidator.BaseURL
			_, err = balances.InsertTrieNode(savedValidator.GetUrlKey(sc.ID), &datastore.NOIDField{})
			if err != nil {
				return fmt.Errorf("saving validator url: " + err.Error())
			}
		}
	}

	// update stake pool settings
	var existingStakePool *stakePool
	if existingStakePool, err = sc.getStakePool(spenum.Validator, inputValidator.ID, balances); err != nil {
		return fmt.Errorf("can't get stake pool:  %v", err)
	}

	if inputValidator.StakePoolSettings != nil {
		// update statistics
		sc.statIncr(statUpdateValidator)

		if inputValidator.StakePoolSettings.ServiceChargeRatio != nil {
			existingStakePool.Settings.ServiceChargeRatio = *inputValidator.StakePoolSettings.ServiceChargeRatio
			savedValidator.StakePoolSettings.ServiceChargeRatio = *inputValidator.StakePoolSettings.ServiceChargeRatio
		}

		if inputValidator.StakePoolSettings.MaxNumDelegates != nil {
			existingStakePool.Settings.MaxNumDelegates = *inputValidator.StakePoolSettings.MaxNumDelegates
			savedValidator.StakePoolSettings.MaxNumDelegates = *inputValidator.StakePoolSettings.MaxNumDelegates
		}

		if inputValidator.StakePoolSettings.DelegateWallet != nil {
			existingStakePool.Settings.DelegateWallet = *inputValidator.StakePoolSettings.DelegateWallet
			savedValidator.StakePoolSettings.DelegateWallet = *inputValidator.StakePoolSettings.DelegateWallet
		}

		if err = validateStakePoolSettings(existingStakePool.StakePool.Settings, conf); err != nil {
			return fmt.Errorf("invalid new stake pool settings:  %v", err)
		}

		// save stake pool
		if err = existingStakePool.Save(spenum.Validator, inputValidator.ID, balances); err != nil {
			return fmt.Errorf("saving stake pool: %v", err)
		}
	}

	savedValidator.LastHealthCheck = txn.CreationDate
	if err := savedValidator.emitUpdate(existingStakePool, balances); err != nil {
		return fmt.Errorf("emmiting validator %v: %v", inputValidator, err)
	}

	return
}

func filterHealthyValidators(now common.Timestamp) filterValidatorFunc {
	return filterValidatorFunc(func(v *ValidationNode) (kick bool, err error) {
		return v.LastHealthCheck <= (now - validatorHealthTime), nil
	})
}

func (sc *StorageSmartContract) validatorHealthCheck(t *transaction.Transaction,
	_ []byte, balances state.StateContextI,
) (string, error) {

	var (
		validator *ValidationNode
		downtime  uint64
		err       error
	)

	if validator, err = sc.getValidator(t.ClientID, balances); err != nil {
		return "", common.NewError("validator_health_check_failed",
			"can't get the validator "+t.ClientID+": "+err.Error())
	}

	downtime = common.Downtime(validator.LastHealthCheck, t.CreationDate)
	validator.LastHealthCheck = t.CreationDate

	emitValidatorHealthCheck(validator, downtime, balances)

	_, err = balances.InsertTrieNode(validator.GetKey(sc.ID), validator)

	if err != nil {
		return "", common.NewError("validator_health_check_failed",
			"can't Save validator: "+err.Error())
	}

	return string(validator.Encode()), nil
}
