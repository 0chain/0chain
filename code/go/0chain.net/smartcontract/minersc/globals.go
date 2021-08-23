package minersc

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/core/util"
	"0chain.net/smartcontract"

	"0chain.net/chaincore/transaction"

	"0chain.net/core/viper"

	"0chain.net/core/encryption"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type GlobalSetting int

const (
	// development
	State        GlobalSetting = iota // todo from development
	Dkg                               // todo from development
	ViewChange                        // todo from development
	BlockRewards                      // todo from development
	Storage                           // todo from development
	Faucet                            // todo from development
	Zrc20                             // todo from development
	Interest                          // todo from development
	Miner                             // todo from development
	Multisig                          // todo from development
	Vesting                           // todo from development
	Owner
	BlockMinSize
	BlockMaxSize
	BlockMaxByteSize
	BlockReplicators
	BlockGenerationTimout        // todo from miner.update
	BlockGenerationRetryWaitTime // todo from chain
	BlockProposalMaxWaitTime
	BlockProposalWaitMode
	BlockConsensusThresholdByCount
	BlockConsensusThresholdByStake
	BlockShardingMinActiveSharders
	BlockShardingMinActiveReplicators
	BlockValidationBatchSize
	BlockReuseTransactions
	BlockMinGenerators
	BlockGeneratorsPercent
	RoundRange
	RoundTimeoutsSofttoMin
	RoundTimeoutsSofttoMult
	RoundTimeoutsRoundRestartMult
	RoundTimeoutsTimeoutCap // todo from chain
	TransactionPayloadMaxSize
	TransactionTimeout // todo from global
	TransactionMinFee  // todo from global
	ClientSignatureScheme
	ClientDiscover // todo from chain
	MessagesVerificationTicketsTo
	StatePruneBelowCount
	StateSyncTimeout   // todo from chain
	StuckCheckInterval // todo from chain
	StuckTimeThreshold // todo from chain
	SmartContractTimeout
	LfbTicketRebroadcastTimeout              // todo restart worker
	LfbTicketAhead                           // todo from chain
	AsyncFetchingMaxSimultaneousFromMiners   // todo restart worker
	AsyncFetchingMaxSimultaneousFromSharders // todo restart worker

	NumOfGlobalSettings
)

var GlobalSettingName = []string{
	"development.state",
	"development.dkg",
	"development.view_change",
	"development.block_rewards",
	"development.smart_contract.storage",
	"development.smart_contract.faucet",
	"development.smart_contract.zrc20",
	"development.smart_contract.interest",
	"development.smart_contract.miner",
	"development.smart_contract.multisig",
	"development.smart_contract.vesting",
	"server_chain.owner",
	"server_chain.block.min_block_size",
	"server_chain.block.max_block_size",
	"server_chain.block.max_byte_size",
	"server_chain.block.replicators",
	"server_chain.block.generation.timeout",
	"server_chain.block.generation.retry_wait_time",
	"server_chain.block.proposal.max_wait_time",
	"server_chain.block.proposal.wait_mode",
	"server_chain.block.consensus.threshold_by_count",
	"server_chain.block.consensus.threshold_by_stake",
	"server_chain.block.sharding.min_active_sharders",
	"server_chain.block.sharding.min_active_replicators",
	"server_chain.block.validation.batch_size",
	"server_chain.block.reuse_txns",
	"server_chain.block.min_generators",
	"server_chain.block.generators_percent",
	"server_chain.round_range",
	"server_chain.round_timeouts.softto_min",
	"server_chain.round_timeouts.softto_mult",
	"server_chain.round_timeouts.round_restart_mult",
	"server_chain.round_timeouts.timeout_cap",
	"server_chain.transaction.payload.max_size",
	"server_chain.transaction.timeout",
	"server_chain.transaction.min_fee",
	"server_chain.client.signature_scheme",
	"server_chain.client.discover",
	"server_chain.messages.verification_tickets_to",
	"server_chain.state.prune_below_count",
	"server_chain.state.sync.timeout",
	"server_chain.stuck.check_interval",
	"server_chain.stuck.time_threshold",
	"server_chain.smart_contract.timeout",
	"server_chain.lfb_ticket.rebroadcast_timeout",
	"server_chain.lfb_ticket.ahead",
	"server_chain.async_blocks_fetching.max_simultaneous_from_miners",
	"server_chain.async_blocks_fetching.max_simultaneous_from_sharders",
}

var GlobalSettingTypes = map[string]smartcontract.ConfigType{
	GlobalSettingName[State]:                                    smartcontract.Boolean,
	GlobalSettingName[Dkg]:                                      smartcontract.Boolean,
	GlobalSettingName[ViewChange]:                               smartcontract.Boolean,
	GlobalSettingName[BlockRewards]:                             smartcontract.Boolean,
	GlobalSettingName[Storage]:                                  smartcontract.Boolean,
	GlobalSettingName[Faucet]:                                   smartcontract.Boolean,
	GlobalSettingName[Zrc20]:                                    smartcontract.Boolean,
	GlobalSettingName[Interest]:                                 smartcontract.Boolean,
	GlobalSettingName[Miner]:                                    smartcontract.Boolean,
	GlobalSettingName[Multisig]:                                 smartcontract.Boolean,
	GlobalSettingName[Vesting]:                                  smartcontract.Boolean,
	GlobalSettingName[Owner]:                                    smartcontract.String,
	GlobalSettingName[BlockMinSize]:                             smartcontract.Int32,
	GlobalSettingName[BlockMaxSize]:                             smartcontract.Int32,
	GlobalSettingName[BlockMaxByteSize]:                         smartcontract.Int64,
	GlobalSettingName[BlockReplicators]:                         smartcontract.Int,
	GlobalSettingName[BlockGenerationTimout]:                    smartcontract.Int,
	GlobalSettingName[BlockGenerationRetryWaitTime]:             smartcontract.Int,
	GlobalSettingName[BlockProposalMaxWaitTime]:                 smartcontract.Duration,
	GlobalSettingName[BlockProposalWaitMode]:                    smartcontract.String,
	GlobalSettingName[BlockConsensusThresholdByCount]:           smartcontract.Int,
	GlobalSettingName[BlockConsensusThresholdByStake]:           smartcontract.Int,
	GlobalSettingName[BlockShardingMinActiveSharders]:           smartcontract.Int,
	GlobalSettingName[BlockShardingMinActiveReplicators]:        smartcontract.Int,
	GlobalSettingName[BlockValidationBatchSize]:                 smartcontract.Int,
	GlobalSettingName[BlockReuseTransactions]:                   smartcontract.Boolean,
	GlobalSettingName[BlockMinGenerators]:                       smartcontract.Int,
	GlobalSettingName[BlockGeneratorsPercent]:                   smartcontract.Float64,
	GlobalSettingName[RoundRange]:                               smartcontract.Int64,
	GlobalSettingName[RoundTimeoutsSofttoMin]:                   smartcontract.Int,
	GlobalSettingName[RoundTimeoutsSofttoMult]:                  smartcontract.Int,
	GlobalSettingName[RoundTimeoutsRoundRestartMult]:            smartcontract.Int,
	GlobalSettingName[RoundTimeoutsTimeoutCap]:                  smartcontract.Int,
	GlobalSettingName[TransactionPayloadMaxSize]:                smartcontract.Int,
	GlobalSettingName[TransactionTimeout]:                       smartcontract.Int,
	GlobalSettingName[TransactionMinFee]:                        smartcontract.Int64,
	GlobalSettingName[ClientSignatureScheme]:                    smartcontract.String,
	GlobalSettingName[ClientDiscover]:                           smartcontract.Boolean,
	GlobalSettingName[MessagesVerificationTicketsTo]:            smartcontract.String,
	GlobalSettingName[StatePruneBelowCount]:                     smartcontract.Int,
	GlobalSettingName[StateSyncTimeout]:                         smartcontract.Duration,
	GlobalSettingName[StuckCheckInterval]:                       smartcontract.Duration,
	GlobalSettingName[StuckTimeThreshold]:                       smartcontract.Duration,
	GlobalSettingName[SmartContractTimeout]:                     smartcontract.Duration,
	GlobalSettingName[LfbTicketRebroadcastTimeout]:              smartcontract.Duration,
	GlobalSettingName[LfbTicketAhead]:                           smartcontract.Int,
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromMiners]:   smartcontract.Int,
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromSharders]: smartcontract.Int,
}

var GLOBALS_KEY = datastore.Key(encryption.Hash("global_settings"))

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
		kType, found := GlobalSettingTypes[key]
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

/*
func (gl *GlobalSettings) updateField(target *interface{}, field Setting) {
	switch SettingTypes[field] {

	}
}

func (gl *GlobalSettings) updateInt(target *int, field Setting) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(int); ok {
			*target = v
			return
		}
	}
	*target = viper.GetInt(SettingName[field])
}


func(gl *GlobalSettings)  updateInt8(target *int8, field Setting, gl.Fields map[string]interface{}) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(int8); ok {
			*target = v
			return
		}
	}
	*target = int8(viper.GetInt(SettingName[field]))
}

func (gl *GlobalSettings) updateInt32(target *int32, field Setting, gl.Fields map[string]interface{}) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(int32); ok {
			*target = v
			return
		}
	}
	*target = viper.GetInt32(SettingName[field])
}

func (gl *GlobalSettings) updateInt64(target *int64, field Setting, gl.Fields map[string]interface{}) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(int64); ok {
			*target = v
			return
		}
	}
	*target = viper.GetInt64(SettingName[field])
}

func (gl *GlobalSettings) updateFloat64(target *float64, field Setting, gl.Fields map[string]interface{}) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(float64); ok {
			*target = v
			return
		}
	}
	*target = viper.GetFloat64(SettingName[field])
}

func(gl *GlobalSettings)  updateString(target *string, field Setting, gl.Fields map[string]interface{}) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(string); ok {
			*target = v
			return
		}
	}
	*target = viper.GetString(SettingName[field])
}

func(gl *GlobalSettings)  updateBool(target *bool, field Setting, gl.Fields map[string]interface{}) {
	if value, found := gl.Fields[SettingName[field]]; found {
		if v, ok := value.(bool); ok {
			*target = v
			return
		}
	}
	*target = viper.GetBool(SettingName[field])
}
*/
func getGlobalsFromViper() map[string]interface{} {
	globals := make(map[string]interface{})
	for key := range GlobalSettingTypes {
		globals[key] = viper.Get(key)
	}
	return globals
}

func getStringMapFromViper() map[string]string {
	globals := make(map[string]string)
	for key := range GlobalSettingTypes {
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
	logging.Logger.Info("piers piers updateGlobals start")
	if txn.ClientID != owner {
		return "", common.NewError("update_globals",
			"unauthorized access - only the owner can update the variables")
	}

	var changes smartcontract.StringMap
	if err = changes.Decode(inputData); err != nil {
		return "", common.NewError("update_globals", err.Error())
	}
	logging.Logger.Info("piers piers updateGlobals", zap.Any("changes", changes))

	globals, err := getGlobalSettings(balances)
	logging.Logger.Info("piers piers updateGlobals", zap.Any("globals", globals))
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

	logging.Logger.Info("piers piers updateGlobals", zap.Any("about to save globals", globals))
	if err := globals.save(balances); err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	return "", nil
}
