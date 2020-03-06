package storagesc

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func scConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":configurations")
}

// read pool configs

type readPoolConfig struct {
	MinLock       int64         `json:"min_lock"`
	MinLockPeriod time.Duration `json:"min_lock_period"`
	MaxLockPeriod time.Duration `json:"max_lock_period"`
}

// write pool configurations

type writePoolConfig struct {
	MinLock int64 `json:"min_lock"`
	// TODO (sfxdx): interests? other configs?
}

// scConfig represents SC configurations ('storagesc:' from sc.yaml).
type scConfig struct {
	ChallengeEnabled      bool          `json:"challenge_enabled"`
	ChallengeRatePerMBMin time.Duration `json:"challenge_rate_per_mb_min"`
	// MinAllocSize is minimum possible size (bytes)
	// of an allocation the SC accept.
	MinAllocSize int64 `json:"min_alloc_size"`
	// MinAllocDuration is minimum possible duration of an
	// allocation allowed by the SC.
	MinAllocDuration time.Duration `json:"min_alloc_duration"`
	// MaxChallengeCompletionTime is max time to complete a challenge.
	MaxChallengeCompletionTime time.Duration `json:"max_challenge_completion_time"`
	// MinOfferDuration represents lower boundary of blobber's MaxOfferDuration.
	MinOfferDuration time.Duration `json:"min_offer_duration"`
	// MinBlobberCapacity allowed to register in the SC.
	MinBlobberCapacity int64 `json:"min_blobber_capacity"`
	// ReadPool related configurations.
	ReadPool *readPoolConfig `json:"readpool"`
	// WritePool related configurations.
	WritePool *writePoolConfig `json:"writepool"`
}

func (conf *scConfig) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(conf); err != nil {
		panic(err) // must not happens
	}
	return
}

func (conf *scConfig) Decode(b []byte) error {
	return json.Unmarshal(b, conf)
}

//
// rest handler and update function
//

// getConfigBytes returns encoded configurations or an error.
func (ssc *StorageSmartContract) getConfigBytes(
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(scConfigKey(ssc.ID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// configs from sc.yaml
func getConfiguredConfig() (conf *scConfig) {

	const prefix = "smart_contracts.storagesc."

	conf = new(scConfig)
	// sc
	conf.ChallengeEnabled = config.SmartContractConfig.GetBool(
		prefix + "challenge_enabled")
	conf.ChallengeRatePerMBMin = config.SmartContractConfig.GetDuration(
		prefix + "challenge_rate_per_mb_min")
	conf.MinAllocSize = config.SmartContractConfig.GetInt64(
		prefix + "min_alloc_size")
	conf.MinAllocDuration = config.SmartContractConfig.GetDuration(
		prefix + "min_alloc_duration")
	// read pool
	conf.ReadPool = new(readPoolConfig)
	conf.ReadPool.MinLockPeriod = config.SmartContractConfig.GetDuration(
		prefix + "readpool.min_lock_period")
	conf.ReadPool.MaxLockPeriod = config.SmartContractConfig.GetDuration(
		prefix + "readpool.max_lock_period")
	conf.ReadPool.MinLock = config.SmartContractConfig.GetInt64(
		prefix + "readpool.min_lock")
	// write pool
	conf.WritePool = new(writePoolConfig)
	conf.WritePool.MinLock = config.SmartContractConfig.GetInt64(
		prefix + "writepool.min_lock")
	return
}

func (ssc *StorageSmartContract) setupConfig(
	balances chainState.StateContextI) (conf *scConfig, err error) {

	conf = getConfiguredConfig()
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return nil, err
	}
	return
}

// getConfig
func (ssc *StorageSmartContract) getConfig(
	balances chainState.StateContextI, setup bool) (
	conf *scConfig, err error) {

	var confb []byte
	confb, err = ssc.getConfigBytes(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	conf = new(scConfig)

	if err == util.ErrValueNotPresent {
		if !setup {
			return // value not present
		}
		return ssc.setupConfig(balances)
	}

	if err = conf.Decode(confb); err != nil {
		return nil, err
	}
	return
}

func (ssc *StorageSmartContract) getConfigHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var conf *scConfig
	conf, err = ssc.getConfig(balances, false)

	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	// return configurations from sc.yaml not saving them
	if err == util.ErrValueNotPresent {
		return getConfiguredConfig(), nil
	}

	return conf, nil // actual value
}

// updateConfig is SC function used by SC owner
// to update storage SC configurations
func (ssc *StorageSmartContract) updateConfig(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var update scConfig
	if err = update.Decode(input); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &update)
	if err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	return string(update.Encode()), nil
}

// getWritePoolConfig
func (ssc *StorageSmartContract) getWritePoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *writePoolConfig, err error) {

	var scconf *scConfig
	if scconf, err = ssc.getConfig(balances, setup); err != nil {
		return
	}
	return scconf.WritePool, nil
}

// getReadPoolConfig
func (ssc *StorageSmartContract) getReadPoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *readPoolConfig, err error) {

	var scconf *scConfig
	if scconf, err = ssc.getConfig(balances, setup); err != nil {
		return
	}
	return scconf.ReadPool, nil
}
