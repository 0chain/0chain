package minersc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/core/util"
	"0chain.net/smartcontract"

	"0chain.net/chaincore/transaction"

	"0chain.net/core/viper"

	"0chain.net/core/encryption"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

//msgp:ignore GlobalSetting
//go:generate msgp -io=false -tests=false -v

type GlobalSetting int

const (
	// development
	State        GlobalSetting = iota // todo from development
	Dkg                               // todo from development
	ViewChange                        // todo from development
	BlockRewards                      // todo from development
	Storage                           // todo from development
	Faucet                            // todo from development
	Miner                             // todo from development
	Multisig                          // todo from development

	Vesting // todo from development
	Zcn
	Owner // do we want to set this.
	BlockMinSize
	BlockMaxSize
	BlockMaxCost
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
	TransactionExempt
	ClientSignatureScheme
	ClientDiscover // todo from chain
	MessagesVerificationTicketsTo
	StatePruneBelowCount

	StateSyncTimeout   // todo from chain
	StuckCheckInterval // todo from chain
	StuckTimeThreshold // todo from chain
	SmartContractTimeout
	SmartContractSettingUpdatePeriod
	LfbTicketRebroadcastTimeout              // todo restart worker
	LfbTicketAhead                           // todo from chain
	AsyncFetchingMaxSimultaneousFromMiners   // todo restart worker
	AsyncFetchingMaxSimultaneousFromSharders // todo restart worker

	DbsEventsEnabled
	DbsEventsName
	DbsEventsUser
	DbsEventsPassword
	DbsEventsHost
	DbsEventsPort
	DbsEventsMaxIdleConns
	DbsEventsMaxOpenConns
	DbsEventsConnMaxLifetime

	HealthCheckDeepScanEnabled                 // todo restart worker
	HealthCheckDeepScanBatchSize               // todo restart worker
	HealthCheckDeepScanWindow                  // todo restart worker
	HealthCheckDeepScanSettleSecs              // todo restart worker
	HealthCheckDeepScanIntervalMins            // todo restart worker
	HealthCheckDeepScanReportStatusMins        // todo restart worker
	HealthCheckProximityScanEnabled            // todo restart worker
	HealthCheckProximityScanBatchSize          // todo restart worker
	HealthCheckProximityScanWindow             // todo restart worker
	HealthCheckProximityScanSettleSecs         // todo restart worker
	HealthCheckProximityScanRepeatIntervalMins // todo restart worker
	HealthCheckProximityScanRejportStatusMins  // todo restart worker
	HealthCheckShowCounters                    // todo restart worker

	NumOfGlobalSettings
)

// GlobalSettingName list of global settings keys. This key used to access each
// global setting, either in smartcontract.StringMap or in the viper settings database.
var GlobalSettingName = []string{
	"server_chain.state.enabled",
	"server_chain.dkg",
	"server_chain.view_change",
	"server_chain.block_rewards",
	"server_chain.smart_contract.storage",
	"server_chain.smart_contract.faucet",
	"server_chain.smart_contract.miner",
	"server_chain.smart_contract.multisig",
	"server_chain.smart_contract.vesting",
	"server_chain.smart_contract.zcn",
	"server_chain.owner",
	"server_chain.block.min_block_size",
	"server_chain.block.max_block_size",
	"server_chain.block.max_block_cost",
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
	"server_chain.transaction.exempt",
	"server_chain.client.signature_scheme",
	"server_chain.client.discover",
	"server_chain.messages.verification_tickets_to",
	"server_chain.state.prune_below_count",
	"server_chain.state.sync.timeout",
	"server_chain.stuck.check_interval",
	"server_chain.stuck.time_threshold",
	"server_chain.smart_contract.timeout",
	"server_chain.smart_contract.setting_update_period",
	"server_chain.lfb_ticket.rebroadcast_timeout",
	"server_chain.lfb_ticket.ahead",
	"server_chain.async_blocks_fetching.max_simultaneous_from_miners",
	"server_chain.async_blocks_fetching.max_simultaneous_from_sharders",

	"server_chain.dbs.events.enabled",
	"server_chain.dbs.events.name",
	"server_chain.dbs.events.user",
	"server_chain.dbs.events.password",
	"server_chain.dbs.events.host",
	"server_chain.dbs.events.port",
	"server_chain.dbs.events.max_idle_conns",
	"server_chain.dbs.events.max_open_conns",
	"server_chain.dbs.events.conn_max_lifetime",

	"server_chain.health_check.deep_scan.enabled",
	"server_chain.health_check.deep_scan.batch_size",
	"server_chain.health_check.deep_scan.window",
	"server_chain.health_check.deep_scan.settle_secs",
	"server_chain.health_check.deep_scan.repeat_interval_mins",
	"server_chain.health_check.deep_scan.report_status_mins",
	"server_chain.health_check.proximity_scan.enabled",
	"server_chain.health_check.proximity_scan.batch_size",
	"server_chain.health_check.proximity_scan.window",
	"server_chain.health_check.proximity_scan.settle_secs",
	"server_chain.health_check.proximity_scan.repeat_interval_mins",
	"server_chain.health_check.proximity_scan.report_status_mins",
	"server_chain.health_check.show_counters",
}

// var GlobalSettingsIgnored = map[string]bool{
// 	GlobalSettingName[DbsEventsEnabled]:         true,
// 	GlobalSettingName[DbsEventsName]:            true,
// 	GlobalSettingName[DbsEventsUser]:            true,
// 	GlobalSettingName[DbsEventsPassword]:        true,
// 	GlobalSettingName[DbsEventsHost]:            true,
// 	GlobalSettingName[DbsEventsPort]:            true,
// 	GlobalSettingName[DbsEventsMaxIdleConns]:    true,
// 	GlobalSettingName[DbsEventsMaxOpenConns]:    true,
// 	GlobalSettingName[DbsEventsConnMaxLifetime]: true,
// }

// GlobalSettingInfo Indicates the type of each global settings, and whether it is possible to change each setting
var GlobalSettingInfo = map[string]struct {
	settingDataType smartcontract.ConfigDataType
	SettingType     smartcontract.ConfigType
	// Each setting stored on the MPT will have a security setting to indicate who has permission to change that setting.
	// For global settings, we have the following list
	//   Immutable
	//   0chain owner
	//   Anyone
	securityLevel smartcontract.ConfigSecurityLevel
}{
	GlobalSettingName[State]:                             {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Dkg]:                               {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[ViewChange]:                        {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[BlockRewards]:                      {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Storage]:                           {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Faucet]:                            {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Miner]:                             {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Multisig]:                          {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Vesting]:                           {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Zcn]:                               {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[Owner]:                             {smartcontract.String, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[BlockMinSize]:                      {smartcontract.Int32, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockMaxSize]:                      {smartcontract.Int32, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockMaxCost]:                      {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockMaxByteSize]:                  {smartcontract.Int64, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockReplicators]:                  {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockGenerationTimout]:             {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[BlockGenerationRetryWaitTime]:      {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[BlockProposalMaxWaitTime]:          {smartcontract.Duration, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockProposalWaitMode]:             {smartcontract.String, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockConsensusThresholdByCount]:    {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockConsensusThresholdByStake]:    {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockShardingMinActiveSharders]:    {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockShardingMinActiveReplicators]: {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockValidationBatchSize]:          {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockReuseTransactions]:            {smartcontract.Boolean, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockMinGenerators]:                {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[BlockGeneratorsPercent]:            {smartcontract.Float64, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[RoundRange]:                        {smartcontract.Int64, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[RoundTimeoutsSofttoMin]:            {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[RoundTimeoutsSofttoMult]:           {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[RoundTimeoutsRoundRestartMult]:     {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[RoundTimeoutsTimeoutCap]:           {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[TransactionPayloadMaxSize]:         {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[TransactionTimeout]:                {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[TransactionMinFee]:                 {smartcontract.Int64, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[TransactionExempt]:                 {smartcontract.Strings, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[ClientSignatureScheme]:             {smartcontract.String, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[ClientDiscover]:                    {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[MessagesVerificationTicketsTo]:     {smartcontract.String, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[StatePruneBelowCount]:              {smartcontract.Int, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[StateSyncTimeout]:                  {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[StuckCheckInterval]:                {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[StuckTimeThreshold]:                {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[SmartContractTimeout]:              {smartcontract.Duration, smartcontract.Global, smartcontract.Owner},
	GlobalSettingName[SmartContractSettingUpdatePeriod]:  {smartcontract.Int64, smartcontract.Global, smartcontract.Owner},

	GlobalSettingName[LfbTicketRebroadcastTimeout]:              {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[LfbTicketAhead]:                           {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromMiners]:   {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromSharders]: {smartcontract.Int, smartcontract.Global, smartcontract.Immutable},

	GlobalSettingName[DbsEventsEnabled]:         {smartcontract.Boolean, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsName]:            {smartcontract.String, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsUser]:            {smartcontract.String, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsPassword]:        {smartcontract.String, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsHost]:            {smartcontract.String, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsPort]:            {smartcontract.String, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsMaxIdleConns]:    {smartcontract.Int, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsMaxOpenConns]:    {smartcontract.Int, smartcontract.Local, smartcontract.Owner},
	GlobalSettingName[DbsEventsConnMaxLifetime]: {smartcontract.Duration, smartcontract.Local, smartcontract.Owner},

	GlobalSettingName[HealthCheckDeepScanEnabled]:                 {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckDeepScanBatchSize]:               {smartcontract.Int64, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckDeepScanWindow]:                  {smartcontract.Int64, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckDeepScanSettleSecs]:              {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckDeepScanIntervalMins]:            {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckDeepScanReportStatusMins]:        {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckProximityScanEnabled]:            {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckProximityScanBatchSize]:          {smartcontract.Int64, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckProximityScanWindow]:             {smartcontract.Int64, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckProximityScanSettleSecs]:         {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckProximityScanRepeatIntervalMins]: {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckProximityScanRejportStatusMins]:  {smartcontract.Duration, smartcontract.Global, smartcontract.Immutable},
	GlobalSettingName[HealthCheckShowCounters]:                    {smartcontract.Boolean, smartcontract.Global, smartcontract.Immutable},
}

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

func NewGlobalSettingsFromViper() *GlobalSettings {
	gl := newGlobalSettings()
	gl.Fields = getStringMapFromViper()
	return gl
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

func (gl *GlobalSettings) update(inputMap smartcontract.StringMap) error {
	var err error
	for key, value := range inputMap.Fields {
		info, found := GlobalSettingInfo[key]
		if !found {
			return fmt.Errorf("'%s' is not a valid global setting", key)
		}
		if info.SettingType == smartcontract.Local {
			return fmt.Errorf("'%s' is a local setting specific to a each provider and cannot be updated via a transaction", key)
		}
		if info.SettingType == smartcontract.Local {
			return fmt.Errorf("'%s' is a local setting specific to a each provider and cannot be updated via a transaction", key)
		}
		if info.securityLevel == smartcontract.Immutable {
			return fmt.Errorf("%s cannot be modified via a transaction", key)
		}
		_, err = smartcontract.StringToInterface(value, info.settingDataType)
		if err != nil {
			return fmt.Errorf("%v value %v cannot be parsed as a %s",
				key, value, smartcontract.ConfigTypeName[info.settingDataType])
		}
		gl.Fields[key] = value
	}

	return nil
}

func getGlobalSettingName(field GlobalSetting) (string, error) {
	if field >= NumOfGlobalSettings || 0 > field {
		return "", fmt.Errorf("field out of range %d", field)
	}
	return GlobalSettingName[field], nil
}

// GetInt returns a global setting as an int, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetInt(field GlobalSetting) (int, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetInt(key), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.Int)
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
func (gl *GlobalSettings) GetInt32(field GlobalSetting) (int32, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetInt32(key), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.Int32)
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
func (gl *GlobalSettings) GetInt64(field GlobalSetting) (int64, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetInt64(key), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.Int64)
	if err != nil {
		return viper.GetInt64(key), nil
	}
	value, ok := iValue.(int64)
	if !ok {
		return value, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
	}
	return value, nil
}

// GetFloat64 returns a global setting as a float64, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetFloat64(field GlobalSetting) (float64, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetFloat64(key), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.Float64)
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
func (gl *GlobalSettings) GetDuration(field GlobalSetting) (time.Duration, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return viper.GetDuration(key), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.Duration)
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
func (gl *GlobalSettings) GetString(field GlobalSetting) (string, error) {
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
func (gl *GlobalSettings) GetBool(field GlobalSetting) (bool, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return false, err
	}

	iValue, err := smartcontract.StringToInterface(gl.Fields[key], smartcontract.Boolean)
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
func (gl *GlobalSettings) GetStrings(field GlobalSetting) ([]string, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return nil, err
	}

	iValue, err := smartcontract.StringToInterface(gl.Fields[key], smartcontract.Strings)
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
	for key, info := range GlobalSettingInfo {
		// Skip to load provider specific configuration
		if info.SettingType != smartcontract.Global {
			continue
		}
		if key == "server_chain.transaction.exempt" {
			globals[key] = strings.Join(viper.GetStringSlice(key), ",")
		} else {
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
	var changes smartcontract.StringMap
	if err = changes.Decode(inputData); err != nil {
		return "", common.NewError("update_globals", err.Error())
	}

	if len(changes.Fields) == 0 {
		return "", fmt.Errorf("update_globals: no change is defined by client. returning...")
	}

	if err := smartcontractinterface.AuthorizeWithOwner("update_globals", func() (bool, error) {
		for key := range changes.Fields {
			info, found := GlobalSettingInfo[key]
			if !found {
				return false, fmt.Errorf("update_globals: validation: '%s' is not a valid global setting", key)
			}
			if info.SettingType == smartcontract.Local {
				return false, fmt.Errorf("update_globals: validation: '%s' is a local setting specific to a each provider and cannot be updated via a transaction", key)
			}
			if gn.OwnerId != txn.ClientID {
				if info.securityLevel == smartcontract.Owner {
					return false, nil
				}
			}
		}
		return true, nil
	}); err != nil {
		return "", err
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
