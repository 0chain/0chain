package vestingsc

import (
	"context"
	"errors"
	"net/url"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func configKey(vscKey string) datastore.Key {
	return datastore.Key(scKey + ":configurations")
}

type config struct {
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	MinFriquency    time.Duration `json:"min_friquency"`
	MaxFriquency    time.Duration `json:"max_friquency"`
	MaxDestinations int           `json:"max_destinations"`
	MaxNameLength   int           `json:"max_name_length"`
}

func (c *config) validate() (err error) {
	switch {
	case toSeconds(c.MinDuration) < 1:
		return errors.New("invalid min_duration (< 1s)")
	case toSeconds(c.MaxDuration) <= toSeconds(c.MinDuration):
		return errors.New("invalid max_duration: less or equal to min_duration")
	case toSeconds(c.MinFriquency) < 1:
		return errors.New("invalid min_friquency (< 1s)")
	case toSeconds(c.MaxFriquency) <= toSecond(c.MinFriquency):
		return errors.New("invalid max_friquency:" +
			" less or equal to min_friquency")
	case c.MaxDestinations < 1:
		return errors.New("invalid max_destinations (< 1)")
	case c.MaxNameLength < 1:
		return errors.New("invalid max_name_length (< 1)")
	}
	return
}

func (vsc *VestingSmartContract) getConfigBytes(
	balances chainstate.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(configKey(vsc.ID))
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
	conf.MinDuration = scconf.GetDuration(prefix + "min_duration")
	conf.MaxDuration = scconf.GetDuration(prefix + "max_duration")
	conf.MinFriquency = scconf.GetDuration(prefix + "min_friquency")
	conf.MaxFriquency = scconf.GetDuration(prefix + "max_friquency")
	conf.MaxDestinations = scconf.GetInt(prefix + "max_destinations")
	conf.MaxNameLength = scconf.GetInt(prefix + "max_name_length")

	err = conf.validate()
	return
}

func (vsc *VestingSmartContract) setupConfig(
	balances chainState.StateContextI) (conf *config, err error) {

	if conf, err = getConfiguredConfig(); err != nil {
		return
	}
	if _, err = balances.InsertTrieNode(configKey(vsc.ID), conf); err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) getConfig(balances chainState.StateContextI,
	setup bool) (conf *config, err error) {

	var confb []byte
	confb, err = vsc.getConfigBytes(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	conf = new(config)

	if err == util.ErrValueNotPresent {
		if !setup {
			return // value not present
		}
		return vsc.setupConfig(balances)
	}

	if err = conf.Decode(confb); err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) getConfigHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var conf *config
	conf, err = vsc.getConfig(balances, false)

	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	// return configurations from sc.yaml not saving them
	if err == util.ErrValueNotPresent {
		return getConfiguredConfig()
	}

	return conf, nil // actual value
}

// updateConfig is SC function used by SC
// owner to update storage SC configurations
func (vsc *VestingSmartContract) updateConfig(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var update config
	if err = update.Decode(input); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	if err = update.validate(); err != nil {
		return
	}

	_, err = balances.InsertTrieNode(configKey(vsc.ID), &update)
	if err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	return string(update.Encode()), nil
}
