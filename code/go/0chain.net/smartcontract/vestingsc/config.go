package vestingsc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

const idLength = 64

func configKey(vscKey string) datastore.Key {
	return datastore.Key(vscKey + ":configurations")
}

type config struct {
	AllowAny             bool            `json:"allow_any"`
	Triggers             []datastore.Key `json:"triggers"`
	MinLock              state.Balance   `json:"min_lock"`
	MinDuration          time.Duration   `json:"min_duration"`
	MaxDuration          time.Duration   `json:"max_duration"`
	MinFriquency         time.Duration   `json:"min_friquency"`
	MaxFriquency         time.Duration   `json:"max_friquency"`
	MaxDestinations      int             `json:"max_destinations"`
	MaxDescriptionLength int             `json:"max_description_length"`
}

func (c *config) Encode() (p []byte) {
	var err error
	if p, err = json.Marshal(c); err != nil {
		panic(err) // must not happen
	}
	return
}

func (c *config) Decode(p []byte) error {
	return json.Unmarshal(p, c)
}

func (c *config) validate() (err error) {
	switch {
	case len(c.Triggers) == 0 && !c.AllowAny:
		return errors.New("empty triggers list")
	case c.MinLock <= 0:
		return errors.New("invalid min_lock (<= 0)")
	case toSeconds(c.MinDuration) < 1:
		return errors.New("invalid min_duration (< 1s)")
	case toSeconds(c.MaxDuration) <= toSeconds(c.MinDuration):
		return errors.New("invalid max_duration: less or equal to min_duration")
	case toSeconds(c.MinFriquency) < 1:
		return errors.New("invalid min_friquency (< 1s)")
	case toSeconds(c.MaxFriquency) <= toSeconds(c.MinFriquency):
		return errors.New("invalid max_friquency:" +
			" less or equal to min_friquency")
	case c.MaxDestinations < 1:
		return errors.New("invalid max_destinations (< 1)")
	case c.MaxDescriptionLength < 1:
		return errors.New("invalid max_description_length (< 1)")
	}
	for _, tr := range c.Triggers {
		if len(tr) != idLength {
			return fmt.Errorf("invalid trigger ID length: %d", len(tr))
		}
	}
	return
}

func (c *config) isValidTrigger(id datastore.Key) bool {
	if c.AllowAny {
		return true
	}
	for _, tr := range c.Triggers {
		if tr == id {
			return true
		}
	}
	return false
}

//
// helpers
//

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
	conf.AllowAny = scconf.GetBool(prefix + "allow_any")
	conf.Triggers = scconf.GetStringSlice(prefix + "triggers")
	conf.MinLock = state.Balance(scconf.GetFloat64(prefix+"min_lock") * 1e10)
	conf.MinDuration = scconf.GetDuration(prefix + "min_duration")
	conf.MaxDuration = scconf.GetDuration(prefix + "max_duration")
	conf.MinFriquency = scconf.GetDuration(prefix + "min_friquency")
	conf.MaxFriquency = scconf.GetDuration(prefix + "max_friquency")
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
	if _, err = balances.InsertTrieNode(configKey(vsc.ID), conf); err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) getConfig(balances chainstate.StateContextI,
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

//
// SC functions
//

func (vsc *VestingSmartContract) updateConfig(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("update_config", "only SC owner can do that")
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

//
// REST-handler
//

func (vsc *VestingSmartContract) getConfigHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
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
