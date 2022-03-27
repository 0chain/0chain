package vestingsc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract"

	chainstate "0chain.net/chaincore/chain/state"
	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

type Setting int

const (
	MinLock Setting = iota
	MinDuration
	MaxDuration
	MaxDestinations
	MaxDescriptionLength
	OwnerId
	Cost
)

var (
	Settings = []string{
		"min_lock",
		"min_duration",
		"max_duration",
		"max_destinations",
		"max_description_length",
		"owner_id",
		"cost",
	}
)

func scConfigKey(scKey string) datastore.Key {
	return scKey + ":configurations"
}

// config represents SC configurations ('vestingsc:' from sc.yaml)
type config struct {
	MinLock              state.Balance  `json:"min_lock"`
	MinDuration          time.Duration  `json:"min_duration"`
	MaxDuration          time.Duration  `json:"max_duration"`
	MaxDestinations      int            `json:"max_destinations"`
	MaxDescriptionLength int            `json:"max_description_length"`
	OwnerId              string         `json:"owner_id"`
	Cost                 map[string]int `json:"cost"`
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

func (c *config) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(c); err != nil {
		panic(err) // must not happens
	}
	return
}

func (c *config) Decode(b []byte) error {
	return json.Unmarshal(b, c)
}

func (c *config) update(changes *smartcontract.StringMap) error {
	for key, value := range changes.Fields {
		switch key {
		case Settings[MinLock]:
			if sbValue, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("value %v cannot be converted to state.Balance, "+
					"failing to set config key %s", value, key)
			} else {
				c.MinLock = state.Balance(sbValue * 1e10)
			}
		case Settings[MinDuration]:
			if dValue, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				c.MinDuration = dValue
			}
		case Settings[MaxDuration]:
			if dValue, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				c.MaxDuration = dValue
			}
		case Settings[MaxDestinations]:
			if iValue, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				c.MaxDestinations = iValue
			}
		case Settings[MaxDescriptionLength]:
			if iValue, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to time.Duration, "+
					"failing to set config key %s", value, key)
			} else {
				c.MaxDescriptionLength = iValue
			}
		case Settings[OwnerId]:
			if _, err := hex.DecodeString(value); err != nil {
				return fmt.Errorf("value %v cannot be converted to int with 16 base, "+
					"failing to set config key %s", value, key)
			} else {
				c.OwnerId = value
			}
		case Settings[Cost]:

		default:
			return fmt.Errorf("config setting %s not found", key)
		}
	}
	return nil
}

func (c *config) getConfigMap() smartcontract.StringMap {
	sMap := smartcontract.StringMap{
		Fields: make(map[string]string),
	}

	sMap.Fields[Settings[MinLock]] = fmt.Sprintf("%v", float64(c.MinLock)/1e10)
	sMap.Fields[Settings[MinDuration]] = fmt.Sprintf("%v", c.MinDuration)
	sMap.Fields[Settings[MaxDuration]] = fmt.Sprintf("%v", c.MaxDuration)
	sMap.Fields[Settings[MaxDestinations]] = fmt.Sprintf("%v", c.MaxDestinations)
	sMap.Fields[Settings[MaxDescriptionLength]] = fmt.Sprintf("%v", c.MaxDescriptionLength)
	sMap.Fields[Settings[OwnerId]] = fmt.Sprintf("%v", c.OwnerId)
	sMap.Fields[Settings[Cost]] = fmt.Sprintf("%v", c.Cost)
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
	conf.Cost = scconf.GetStringMapInt(prefix + "cost")

	err = conf.validate()
	if err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) getConfig(
	balances chainstate.StateContextI,
) (conf *config, err error) {
	conf = new(config)
	err = balances.GetTrieNode(scConfigKey(vsc.ID), conf)
	switch err {
	case nil:
		return conf, nil
	case util.ErrValueNotPresent:
		return vsc.setupConfig(balances)
	default:
		return nil, err
	}
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

//
// REST-handler
//

func (vsc *VestingSmartContract) getConfigHandler(
	ctx context.Context,
	params url.Values,
	balances chainstate.StateContextI,
) (interface{}, error) {
	conf, err := vsc.getConfig(balances)
	if err != nil {
		return nil, common.NewErrInternal("can't get config", err.Error())
	}

	return conf.getConfigMap(), nil
}
