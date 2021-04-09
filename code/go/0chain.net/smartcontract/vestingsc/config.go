package vestingsc

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func scConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":configurations")
}

type config struct {
	MinLock              state.Balance `json:"min_lock"`
	MinDuration          time.Duration `json:"min_duration"`
	MaxDuration          time.Duration `json:"max_duration"`
	MaxDestinations      int           `json:"max_destinations"`
	MaxDescriptionLength int           `json:"max_description_length"`
}

func (c *config) validate() (err error) {
	switch {
	case c.MinLock <= 0:
		return errors.New("invalid min_lock (<= 0)")
	case toSeconds(c.MinDuration) < 1:
		return errors.New("invalid min_duration (< 1s)")
	case toSeconds(c.MaxDuration) <= toSeconds(c.MinDuration):
		return errors.New("invalid max_duration: less or equal to min_duration")
	case c.MaxDestinations < 1:
		return errors.New("invalid max_destinations (< 1)")
	case c.MaxDescriptionLength < 1:
		return errors.New("invalid max_description_length (< 1)")
	}
	return
}

func (conf *config) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(conf); err != nil {
		panic(err) // must not happens
	}
	return
}

func (conf *config) Decode(b []byte) error {
	return json.Unmarshal(b, conf)
}

func (conf *config) update(newConf *config) {
	if newConf.MinLock > 0 {
		conf.MinLock = newConf.MinLock
	}
	if newConf.MinDuration > 0 {
		conf.MinDuration = newConf.MinDuration
	}
	if newConf.MaxDuration > 0 {
		conf.MaxDuration = newConf.MaxDuration
	}
	if newConf.MaxDestinations > 0 {
		conf.MaxDestinations = newConf.MaxDestinations
	}
	if newConf.MaxDescriptionLength > 0 {
		conf.MaxDescriptionLength = newConf.MaxDescriptionLength
	}
}

func (vsc *VestingSmartContract) updateConfig(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var conf *config
	if conf, err = vsc.getConfig(balances); err != nil {
		return "", common.NewError("update_config",
			"can't get config: "+err.Error())
	}

	var update *config
	if err = update.Decode(input); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	conf.update(update)

	_, err = balances.InsertTrieNode(scConfigKey(vsc.ID), conf)
	if err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	return string(conf.Encode()), nil
}

//
// helpers
//

func (vsc *VestingSmartContract) getConfigBytes(
	balances chainstate.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(scConfigKey(vsc.ID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// configurations from sc.yaml
func getConfiguredConfig() (conf *config, err error) {

	const prefix = "smart_contracts.vestingsc."

	conf = new(config)

	// short hand
	var scconf = configpkg.SmartContractConfig
	conf.MinLock = state.Balance(scconf.GetFloat64(prefix+"min_lock") * 1e10)
	conf.MinDuration = scconf.GetDuration(prefix + "min_duration")
	conf.MaxDuration = scconf.GetDuration(prefix + "max_duration")
	conf.MaxDestinations = scconf.GetInt(prefix + "max_destinations")
	conf.MaxDescriptionLength = scconf.GetInt(prefix + "max_description_length")

	err = conf.validate()
	return
}

func (vsc *VestingSmartContract) setupConfig(
	balances chainstate.StateContextI) (conf *config, err error) {

	if conf, err = getConfiguredConfig(); err != nil {
		return
	}
	_, err = balances.InsertTrieNode(scConfigKey(vsc.ID), conf)
	if err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) getConfig(
	balances chainstate.StateContextI) (
	conf *config, err error) {

	var confb []byte
	confb, err = vsc.getConfigBytes(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	conf = new(config)

	if err == util.ErrValueNotPresent {
		return vsc.setupConfig(balances)
	}

	if err = conf.Decode(confb); err != nil {
		return nil, err
	}
	return
}

//
// REST-handler
//

func (vsc *VestingSmartContract) getConfigHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (interface{}, error) {

	return vsc.getConfig(balances)
}
