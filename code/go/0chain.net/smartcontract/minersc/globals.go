package minersc

import (
	"0chain.net/chaincore/smartcontractinterface"
	"encoding/json"
	"fmt"
	"time"

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
	Interest                          // todo from development
	Miner                             // todo from development
	Multisig                          // todo from development

	Vesting // todo from development
	Owner   // do we want to set this.
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
	"development.state",
	"development.dkg",
	"development.view_change",
	"development.block_rewards",
	"development.smart_contract.storage",
	"development.smart_contract.faucet",
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

// GlobalSettingInfo Indicates the type of each global settings, and whether it is possible to change each setting
var GlobalSettingInfo = map[string]struct {
	settingType smartcontract.ConfigType
	mutable     bool
}{
	GlobalSettingName[State]:                             {smartcontract.Boolean, false},
	GlobalSettingName[Dkg]:                               {smartcontract.Boolean, false},
	GlobalSettingName[ViewChange]:                        {smartcontract.Boolean, false},
	GlobalSettingName[BlockRewards]:                      {smartcontract.Boolean, false},
	GlobalSettingName[Storage]:                           {smartcontract.Boolean, false},
	GlobalSettingName[Faucet]:                            {smartcontract.Boolean, false},
	GlobalSettingName[Interest]:                          {smartcontract.Boolean, false},
	GlobalSettingName[Miner]:                             {smartcontract.Boolean, false},
	GlobalSettingName[Multisig]:                          {smartcontract.Boolean, false},
	GlobalSettingName[Vesting]:                           {smartcontract.Boolean, false},
	GlobalSettingName[Owner]:                             {smartcontract.String, false},
	GlobalSettingName[BlockMinSize]:                      {smartcontract.Int32, true},
	GlobalSettingName[BlockMaxSize]:                      {smartcontract.Int32, true},
	GlobalSettingName[BlockMaxByteSize]:                  {smartcontract.Int64, true},
	GlobalSettingName[BlockReplicators]:                  {smartcontract.Int, true},
	GlobalSettingName[BlockGenerationTimout]:             {smartcontract.Int, false},
	GlobalSettingName[BlockGenerationRetryWaitTime]:      {smartcontract.Int, false},
	GlobalSettingName[BlockProposalMaxWaitTime]:          {smartcontract.Duration, true},
	GlobalSettingName[BlockProposalWaitMode]:             {smartcontract.String, true},
	GlobalSettingName[BlockConsensusThresholdByCount]:    {smartcontract.Int, true},
	GlobalSettingName[BlockConsensusThresholdByStake]:    {smartcontract.Int, true},
	GlobalSettingName[BlockShardingMinActiveSharders]:    {smartcontract.Int, true},
	GlobalSettingName[BlockShardingMinActiveReplicators]: {smartcontract.Int, true},
	GlobalSettingName[BlockValidationBatchSize]:          {smartcontract.Int, true},
	GlobalSettingName[BlockReuseTransactions]:            {smartcontract.Boolean, true},
	GlobalSettingName[BlockMinGenerators]:                {smartcontract.Int, true},
	GlobalSettingName[BlockGeneratorsPercent]:            {smartcontract.Float64, true},
	GlobalSettingName[RoundRange]:                        {smartcontract.Int64, true},
	GlobalSettingName[RoundTimeoutsSofttoMin]:            {smartcontract.Int, true},
	GlobalSettingName[RoundTimeoutsSofttoMult]:           {smartcontract.Int, true},
	GlobalSettingName[RoundTimeoutsRoundRestartMult]:     {smartcontract.Int, true},
	GlobalSettingName[RoundTimeoutsTimeoutCap]:           {smartcontract.Int, false},
	GlobalSettingName[TransactionPayloadMaxSize]:         {smartcontract.Int, true},
	GlobalSettingName[TransactionTimeout]:                {smartcontract.Int, false},
	GlobalSettingName[TransactionMinFee]:                 {smartcontract.Int64, false},
	GlobalSettingName[ClientSignatureScheme]:             {smartcontract.String, true},
	GlobalSettingName[ClientDiscover]:                    {smartcontract.Boolean, false},
	GlobalSettingName[MessagesVerificationTicketsTo]:     {smartcontract.String, true},
	GlobalSettingName[StatePruneBelowCount]:              {smartcontract.Int, true},
	GlobalSettingName[StateSyncTimeout]:                  {smartcontract.Duration, false},
	GlobalSettingName[StuckCheckInterval]:                {smartcontract.Duration, false},
	GlobalSettingName[StuckTimeThreshold]:                {smartcontract.Duration, false},
	GlobalSettingName[SmartContractTimeout]:              {smartcontract.Duration, true},
	GlobalSettingName[SmartContractSettingUpdatePeriod]:  {smartcontract.Int64, true},

	GlobalSettingName[LfbTicketRebroadcastTimeout]:              {smartcontract.Duration, false},
	GlobalSettingName[LfbTicketAhead]:                           {smartcontract.Int, false},
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromMiners]:   {smartcontract.Int, false},
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromSharders]: {smartcontract.Int, false},

	GlobalSettingName[DbsEventsEnabled]:         {smartcontract.Boolean, true},
	GlobalSettingName[DbsEventsName]:            {smartcontract.String, true},
	GlobalSettingName[DbsEventsUser]:            {smartcontract.String, true},
	GlobalSettingName[DbsEventsPassword]:        {smartcontract.String, true},
	GlobalSettingName[DbsEventsHost]:            {smartcontract.String, true},
	GlobalSettingName[DbsEventsPort]:            {smartcontract.String, true},
	GlobalSettingName[DbsEventsMaxIdleConns]:    {smartcontract.Int, true},
	GlobalSettingName[DbsEventsMaxOpenConns]:    {smartcontract.Int, true},
	GlobalSettingName[DbsEventsConnMaxLifetime]: {smartcontract.Duration, true},

	GlobalSettingName[HealthCheckDeepScanEnabled]:                 {smartcontract.Boolean, false},
	GlobalSettingName[HealthCheckDeepScanBatchSize]:               {smartcontract.Int64, false},
	GlobalSettingName[HealthCheckDeepScanWindow]:                  {smartcontract.Int64, false},
	GlobalSettingName[HealthCheckDeepScanSettleSecs]:              {smartcontract.Duration, false},
	GlobalSettingName[HealthCheckDeepScanIntervalMins]:            {smartcontract.Duration, false},
	GlobalSettingName[HealthCheckDeepScanReportStatusMins]:        {smartcontract.Duration, false},
	GlobalSettingName[HealthCheckProximityScanEnabled]:            {smartcontract.Boolean, false},
	GlobalSettingName[HealthCheckProximityScanBatchSize]:          {smartcontract.Int64, false},
	GlobalSettingName[HealthCheckProximityScanWindow]:             {smartcontract.Int64, false},
	GlobalSettingName[HealthCheckProximityScanSettleSecs]:         {smartcontract.Duration, false},
	GlobalSettingName[HealthCheckProximityScanRepeatIntervalMins]: {smartcontract.Duration, false},
	GlobalSettingName[HealthCheckProximityScanRejportStatusMins]:  {smartcontract.Duration, false},
	GlobalSettingName[HealthCheckShowCounters]:                    {smartcontract.Boolean, false},
}

var GLOBALS_KEY = datastore.Key(encryption.Hash("global_settings"))

func scConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":configurations")
}

type GlobalSettings struct {
	Fields map[string]string `json:"fields"`
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
		if !info.mutable {
			return fmt.Errorf("%s is an immutable setting", key)
		}
		_, err = smartcontract.StringToInterface(value, info.settingType)
		if err != nil {
			return fmt.Errorf("%v value %v cannot be parsed as a %s",
				key, value, smartcontract.ConfigTypeName[info.settingType])
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

func getStringMapFromViper() map[string]string {
	globals := make(map[string]string)
	for key := range GlobalSettingInfo {
		globals[key] = viper.GetString(key)
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
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	if err := smartcontractinterface.AuthorizeWithOwner("update_globals", func() bool {
		return gn.OwnerId == txn.ClientID
	}); err != nil {
		return "", err
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
