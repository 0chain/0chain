package vestingsc

import (
	"0chain.net/chaincore/smartcontractinterface"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract"

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
	OwnerId
)

var (
	Settings = []string{
		"min_lock",
		"min_duration",
		"max_duration",
		"max_destinations",
		"max_description_length",
		"owner_id",
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
	OwnerId              datastore.Key `json:"owner_id"`
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
	case c.OwnerId == "":
		return errors.New("owner_id is not set or empty")
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

func (conf *config) update(changes *smartcontract.StringMap) error {
	for key, value := range changes.Fields {
		switch key {
		case Settings[MinLock]:
			if sbValue, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("value %v cannot be converted to state.Balance, "+
					"failing to set config key %s", value, key)
			} else {
				conf.MinLock = state.Balance(sbValue * 1e10)
			}
		case Settings[MinDuration]:
			if dValue, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				conf.MinDuration = dValue
			}
		case Settings[MaxDuration]:
			if dValue, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				conf.MaxDuration = dValue
			}
		case Settings[MaxDestinations]:
			if iValue, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				conf.MaxDestinations = iValue
			}
		case Settings[MaxDescriptionLength]:
			if iValue, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				conf.MaxDescriptionLength = iValue
			}
		case Settings[OwnerId]:
			if _, err := hex.DecodeString(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to int with 16 base, "+
					"failing to set config key %s", value, key)
			} else {
				conf.OwnerId = value
			}
		default:
			return fmt.Errorf("config setting %s not found", key)
		}
	}
	return nil
}

func (conf *config) getConfigMap() smartcontract.StringMap {
	sMap := smartcontract.StringMap{
		Fields: make(map[string]string),
	}
	sMap.Fields[Settings[MinLock]] = fmt.Sprintf("%v", float64(conf.MinLock)/1e10)
	sMap.Fields[Settings[MinDuration]] = fmt.Sprintf("%v", conf.MinDuration)
	sMap.Fields[Settings[MaxDuration]] = fmt.Sprintf("%v", conf.MaxDuration)
	sMap.Fields[Settings[MaxDestinations]] = fmt.Sprintf("%v", conf.MaxDestinations)
	sMap.Fields[Settings[MaxDescriptionLength]] = fmt.Sprintf("%v", conf.MaxDescriptionLength)
	sMap.Fields[Settings[OwnerId]] = fmt.Sprintf("%v", conf.OwnerId)
	return sMap
}

func (vsc *VestingSmartContract) updateConfig(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	var conf *config
	if conf, err = vsc.getConfig(balances); err != nil {
		return "", common.NewError("update_config",
			"can't get config: "+err.Error())
	}

	if err := smartcontractinterface.AuthorizeWithOwner("update_config", func() bool {
		return conf.OwnerId == txn.ClientID
	}); err != nil {
		return "", err
	}

	update := &smartcontract.StringMap{}
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
	conf.OwnerId = scconf.GetString(prefix + "owner_id")

	err = conf.validate()
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
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		conf, err = getConfiguredConfig()
		if err != nil {
			return nil, err
		}
	} else {
		conf = new(config)
		if err = conf.Decode(confb); err != nil {
			return nil, err
		}
	}
	return conf, nil
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
		if err != util.ErrValueNotPresent {
			return nil, common.NewErrInternal("can't get config", err.Error())
		}
		res, err = getConfiguredConfig()
		if err != nil {
			return nil, common.NewErrInternal("can't read config from file", err.Error())
		}
	}
	return res.getConfigMap(), nil
}
