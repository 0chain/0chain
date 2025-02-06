package minersc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	config2 "0chain.net/core/config"
	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/smartcontractinterface"

	"github.com/0chain/common/core/util"

	"0chain.net/chaincore/transaction"

	"0chain.net/core/viper"

	"0chain.net/core/encryption"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

//msgp:ignore enums.GlobalSetting
//go:generate msgp -io=false -tests=false -v

var GLOBALS_KEY = datastore.Key(encryption.Hash("global_settings"))

// swagger:model MinerGlobalSettings
type GlobalSettings struct {
	Version int64             `json:"version"`
	Fields  map[string]string `json:"fields"`
}

func newGlobalSettings() *GlobalSettings {
	return &GlobalSettings{
		Fields: make(map[string]string),
	}
}

// Encode implements util.Serializable interface.
func (gl *GlobalSettings) Encode() []byte {
	var b, err = json.Marshal(gl)
	if err != nil {
		panic(err)
	}
	return b
}

// Decode implements util.Serializable interface.
func (gl *GlobalSettings) Decode(p []byte) error {
	return json.Unmarshal(p, gl)
}

func (gl *GlobalSettings) save(balances cstate.StateContextI) error {
	gl.Version += 1
	_, err := balances.InsertTrieNode(GLOBALS_KEY, gl)
	return err
}

func (gl *GlobalSettings) update(inputMap config2.StringMap) error {
	var err error
	for key, value := range inputMap.Fields {
		info, found := config2.GlobalSettingInfo[key]
		if !found {
			return fmt.Errorf("'%s' is not a valid global setting", key)
		}
		if !info.Mutable {
			return fmt.Errorf("%s cannot be modified via a transaction", key)
		}
		_, err = config2.StringToInterface(value, info.SettingType)
		if err != nil {
			return fmt.Errorf("%v value %v cannot be parsed as a %s",
				key, value, config2.ConfigTypeName[info.SettingType])
		}
		gl.Fields[key] = value
	}

	return nil
}

func getGlobalSettingName(field config2.GlobalSetting) (string, error) {
	if field >= config2.NumOfGlobalSettings || 0 > field {
		return "", fmt.Errorf("field out of range %d", field)
	}
	return config2.GlobalSettingName[field], nil
}

// GetInt returns a global setting as an int, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetInt(field config2.GlobalSetting) (int, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetInt(key), nil
	}
	iValue, err := config2.StringToInterface(sValue, config2.Int)
	if err != nil {
		return viper.GetInt(key), nil
	}
	value, ok := iValue.(int)
	if !ok {
		return 0, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetInt32 returns a global setting as an int32, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetInt32(field config2.GlobalSetting) (int32, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetInt32(key), nil
	}
	iValue, err := config2.StringToInterface(sValue, config2.Int32)
	if err != nil {
		return viper.GetInt32(key), nil
	}
	value, ok := iValue.(int32)
	if !ok {
		return 0, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetInt64 returns a global setting as an int64, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetInt64(field config2.GlobalSetting) (int64, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetInt64(key), nil
	}
	iValue, err := config2.StringToInterface(sValue, config2.Int64)
	if err != nil {
		return viper.GetInt64(key), nil
	}
	value, ok := iValue.(int64)
	if !ok {
		return value, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetCoin returns a global setting as an currency.Coin, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetCoin(field config2.GlobalSetting) (currency.Coin, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return currency.Coin(viper.GetUint64(key)), nil
	}
	iValue, err := config2.StringToInterface(sValue, config2.CurrencyCoin)
	if err != nil {
		return currency.Coin(viper.GetUint64(key)), nil
	}
	value, ok := iValue.(currency.Coin)
	if !ok {
		return value, fmt.Errorf("cannot convert key %s value %v to type currency.Coin", key, value)
	}
	return value, nil
}

// GetFloat64 returns a global setting as a float64, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetFloat64(field config2.GlobalSetting) (float64, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetFloat64(key), nil
	}
	iValue, err := config2.StringToInterface(sValue, config2.Float64)
	if err != nil {
		return viper.GetFloat64(key), nil
	}
	value, ok := iValue.(float64)
	if !ok {
		return 0.0, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetDuration returns a global setting as a duration, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetDuration(field config2.GlobalSetting) (time.Duration, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetDuration(key), nil
	}
	iValue, err := config2.StringToInterface(sValue, config2.Duration)
	if err != nil {
		return viper.GetDuration(key), nil
	}
	value, ok := iValue.(time.Duration)
	if !ok {
		return 0, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetString returns a global setting as a string, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetString(field config2.GlobalSetting) (string, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return "", err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetString(key), nil
	}
	return sValue, nil
}

// GetBool returns a global setting as a boolean, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetBool(field config2.GlobalSetting) (bool, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return false, err
	}

	iValue, err := config2.StringToInterface(gl.Fields[key], config2.Boolean)
	if err != nil {
		return viper.GetBool(key), nil
	}
	value, ok := iValue.(bool)
	if !ok {
		return false, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetStrings returns a global setting as a []string], a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetStrings(field config2.GlobalSetting) ([]string, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return nil, err
	}

	iValue, err := config2.StringToInterface(gl.Fields[key], config2.Strings)
	if err != nil {
		return viper.GetStringSlice(key), nil
	}
	value, ok := iValue.([]string)
	if !ok {
		return nil, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

func getStringMapFromViper() map[string]string {
	globals := make(map[string]string)
	for key := range config2.GlobalSettingInfo {
		if _, ok := config2.GlobalSettingsIgnored[key]; ok {
			continue
		}
		switch key {
		case config2.GlobalSettingName[config2.TransactionExempt]:
			globals[key] = strings.Join(viper.GetStringSlice(key), ",")
		default:
			globals[key] = viper.GetString(key)
		}
	}
	return globals
}

func getGlobalSettings(balances cstate.CommonStateContextI) (*GlobalSettings, error) {
	gl := newGlobalSettings()

	err := balances.GetTrieNode(GLOBALS_KEY, gl)
	if err != nil {
		return nil, err
	}

	return gl, nil
}

func (msc *MinerSmartContract) updateGlobals(
	txn *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	if err := smartcontractinterface.AuthorizeWithOwner("update_globals", func() bool {
		return gn.MustBase().OwnerId == txn.ClientID
	}); err != nil {
		return "", err
	}
	var changes config2.StringMap
	if err = changes.Decode(inputData); err != nil {
		return "", common.NewError("update_globals", err.Error())
	}

	globals, err := getGlobalSettings(balances)

	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("update_globals", err.Error())
		}
		globals = &GlobalSettings{
			Fields: getStringMapFromViper(),
		}
	}

	if err = globals.update(changes); err != nil {
		return "", common.NewErrorf("update_globals", "validation: %v", err.Error())
	}

	if err := globals.save(balances); err != nil {
		return "", common.NewError("update_globals", err.Error())
	}

	return "", nil
}

func initGlobalSettings(balances cstate.CommonStateContextI) error {
	gs := newGlobalSettings()
	err := balances.GetTrieNode(GLOBALS_KEY, gs)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		gs.Fields = getStringMapFromViper()
		_, err = balances.InsertTrieNode(GLOBALS_KEY, gs)
		return err
	}
	return nil
}
