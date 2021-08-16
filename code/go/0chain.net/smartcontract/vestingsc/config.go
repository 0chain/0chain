package vestingsc

import (
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
)

type Setting int

const (
	MinLock Setting = iota
	MinDuration
	MaxDuration
	MaxDestinations
	MaxDescriptionLength
)

var (
	Settings = []string{
		"min_lock",
		"min_duration",
		"max_duration",
		"max_destinations",
		"max_description_length",
	}
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

func (conf *config) set(key string, value float64) error {
	switch key {
	case Settings[MinLock]:
		conf.MinLock = state.Balance(value)
	case Settings[MinDuration]:
		conf.MinDuration = time.Duration(value)
	case Settings[MaxDuration]:
		conf.MaxDuration = time.Duration(value)
	case Settings[MaxDestinations]:
		conf.MaxDestinations = int(value)
	case Settings[MaxDescriptionLength]:
		conf.MaxDescriptionLength = int(value)
	default:
		return fmt.Errorf("config setting %s not found", key)
	}
	return nil
}

func (conf *config) update(changes *inputMap) error {
	for key, value := range changes.Fields {
		if fValue, ok := value.(float64); !ok {
			return fmt.Errorf("value %v is not of numeric type, failing to set config key %s", value, key)
		} else {
			if err := conf.set(key, fValue); err != nil {
				return err
			}
		}
	}
	return nil
}

type inputMap struct {
	Fields map[string]interface{} `json:"fields"`
}

func (im *inputMap) Decode(input []byte) error {
	err := json.Unmarshal(input, im)
	if err != nil {
		return err
	}
	return nil
}

func (im *inputMap) Encode() []byte {
	buff, _ := json.Marshal(im)
	return buff
}

func (vsc *VestingSmartContract) updateConfig(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	if txn.ClientID != owner {
		return "", common.NewError("update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var conf *config
	if conf, err = vsc.getConfig(balances); err != nil {
		return "", common.NewError("update_config",
			"can't get config: "+err.Error())
	}

	update := &inputMap{}
	if err = update.Decode(input); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	if err := conf.update(update); err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	_, err = balances.InsertTrieNode(scConfigKey(vsc.ID), conf)
	if err != nil {
		return "", common.NewError("update_config", err.Error())
	}

	return "", nil
}

//
// helpers
//

func (vsc *VestingSmartContract) getConfigBytes(
	balances chainstate.StateContextI,
) (b []byte, err error) {
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
	if err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) setupConfig(
	balances chainstate.StateContextI,
) (conf *config, err error) {
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
	balances chainstate.StateContextI,
) (conf *config, err error) {
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

func (vsc *VestingSmartContract) getConfigHandler(
	ctx context.Context,
	params url.Values,
	balances chainstate.StateContextI,
) (interface{}, error) {
	res, err := vsc.getConfig(balances)
	if err != nil {
		return nil, common.NewErrInternal("can't get config", err.Error())
	}
	return res, nil
}
