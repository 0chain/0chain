package minersc

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/util"
	"0chain.net/smartcontract"

	"0chain.net/chaincore/transaction"

	"0chain.net/core/viper"

	"0chain.net/core/encryption"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

var SettingTypes = map[string]smartcontract.ConfigType{
	"development.state":                                                       smartcontract.Boolean,
	"development.dkg":                                                         smartcontract.Boolean,
	"development.view_change":                                                 smartcontract.Boolean,
	"development.smart_contract.storage":                                      smartcontract.Boolean,
	"development.smart_contract.faucet":                                       smartcontract.Boolean,
	"development.smart_contract.zrc20":                                        smartcontract.Boolean,
	"development.smart_contract.interest":                                     smartcontract.Boolean,
	"development.smart_contract.miner":                                        smartcontract.Boolean,
	"development.smart_contract.multisig":                                     smartcontract.Boolean,
	"development.smart_contract.vesting":                                      smartcontract.Boolean,
	"server_chain.block.min_block_size":                                       smartcontract.Int32,
	"server_chain.block.max_block_size":                                       smartcontract.Int32,
	"server_chain.block.max_byte_size":                                        smartcontract.Int64,
	"server_chain.block.generators":                                           smartcontract.Int,
	"server_chain.block.replicators":                                          smartcontract.Int,
	"server_chain.block.generation.timeout":                                   smartcontract.Int,
	"server_chain.block.generation.retry_wait_time":                           smartcontract.Int,
	"server_chain.block.proposal.max_wait_time":                               smartcontract.Duration,
	"server_chain.block.proposal.wait_mode":                                   smartcontract.String,
	"server_chain.block.consensus.threshold_by_count":                         smartcontract.Int,
	"server_chain.block.consensus.threshold_by_stake":                         smartcontract.Int,
	"server_chain.block.sharding.min_active_sharders":                         smartcontract.Int,
	"server_chain.block.sharding.min_active_replicators":                      smartcontract.Int,
	"server_chain.block.validation.batch_size":                                smartcontract.Int,
	"server_chain.block.reuse_txns":                                           smartcontract.Boolean,
	"server_chain.block.round_range":                                          smartcontract.Int64,
	"server_chain.block.round_timeouts.softto_min":                            smartcontract.Int,
	"server_chain.block.round_timeouts.softto_mult":                           smartcontract.Int,
	"server_chain.block.round_timeouts.round_restart_mult":                    smartcontract.Int,
	"server_chain.block.round_timeouts.timeout_cap":                           smartcontract.Int,
	"server_chain.block.transaction.payload.max_size":                         smartcontract.Int,
	"server_chain.block.transaction.timeout":                                  smartcontract.Int,
	"server_chain.block.transaction.min_fee":                                  smartcontract.Int64,
	"server_chain.block.client.signature_scheme":                              smartcontract.String,
	"server_chain.block.client.discover":                                      smartcontract.Boolean,
	"server_chain.block.messages.verification_tickets_to":                     smartcontract.String,
	"server_chain.block.state.prune_below_count":                              smartcontract.Int,
	"server_chain.block.state.sync.timeout":                                   smartcontract.Duration,
	"server_chain.block.stuck.check_interval":                                 smartcontract.Duration,
	"server_chain.block.stuck.time_threshold":                                 smartcontract.Duration,
	"server_chain.block.smart_contract.timeout":                               smartcontract.Duration,
	"server_chain.block.lfb_ticket.rebroadcast_timeout":                       smartcontract.Duration,
	"server_chain.block.lfb_ticket.ahead":                                     smartcontract.Int,
	"server_chain.block.lfb_ticket.fb_fetching_lifetime":                      smartcontract.Duration,
	"server_chain.block.async_blocks_fetching.max_simultaneous_from_miners":   smartcontract.Int,
	"server_chain.block.async_blocks_fetching.max_simultaneous_from_sharders": smartcontract.Int,
}

var GLOBALS_KEY = datastore.Key(ADDRESS + encryption.Hash("global_settings"))

func scConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":configurations")
}

type GlobalSettings struct {
	Fields map[string]interface{} `json:"fields"`
}

func newGlobalSettings() *GlobalSettings {
	return &GlobalSettings{
		Fields: make(map[string]interface{}),
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
	_, err := balances.InsertTrieNode(GLOBALS_KEY, gl)
	return err
}

func (gl *GlobalSettings) update(inputMap smartcontract.StringMap) error {
	var err error
	for key, value := range inputMap.Fields {
		kType, found := SettingTypes[key]
		if !found {
			return fmt.Errorf("'%s' is not a valid global setting", key)
		}
		gl.Fields[key], err = smartcontract.StringToInterface(value, kType)
		if err != nil {
			return fmt.Errorf("%v value %v cannot be perased as a %s", key, value, smartcontract.ConfigTypeName[kType])
		}
	}
	return nil
}

func getGlobalsFromViper() map[string]interface{} {
	globals := make(map[string]interface{})
	for key := range SettingTypes {
		globals[key] = viper.Get(key)
	}
	return globals
}

func getStringMapFromViper() map[string]string {
	globals := make(map[string]string)
	for key := range SettingTypes {
		globals[key] = fmt.Sprintf("%v", viper.Get(key))
	}
	return globals
}

func getGlobalSettingsBytes(balances cstate.StateContextI) ([]byte, error) {
	val, err := balances.GetTrieNode(GLOBALS_KEY)
	if err != nil {
		return nil, err
	}
	return val.Encode(), nil
}

func getGlobalSettings(balances cstate.StateContextI) (*GlobalSettings, error) {
	var err error
	var poolb []byte
	if poolb, err = getGlobalSettingsBytes(balances); err != nil {
		return nil, err
	}
	gl := newGlobalSettings()
	err = gl.Decode(poolb)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return gl, err
}

func (msc *MinerSmartContract) updateGlobals(
	txn *transaction.Transaction,
	inputData []byte,
	_ *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	if txn.ClientID != owner {
		return "", common.NewError("update_globals",
			"unauthorized access - only the owner can update the variables")
	}

	var changes smartcontract.StringMap
	if err = changes.Decode(inputData); err != nil {
		return "", common.NewError("update_globals", err.Error())
	}

	globals, err := getGlobalSettings(balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("update_globals", err.Error())
		}
		globals = &GlobalSettings{
			Fields: getGlobalsFromViper(),
		}
	}

	if err = globals.update(changes); err != nil {
		return "", common.NewErrorf("update_settings", "validation: %v", err.Error())
	}

	if err := globals.save(balances); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	return "", nil
}
