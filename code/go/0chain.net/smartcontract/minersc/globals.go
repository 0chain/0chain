package minersc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"0chain.net/smartcontract/zbig"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/smartcontract"
	"github.com/0chain/common/core/util"

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

var GlobalSettingsIgnored = map[string]bool{
	GlobalSettingName[DbsEventsEnabled]:         true,
	GlobalSettingName[DbsEventsName]:            true,
	GlobalSettingName[DbsEventsUser]:            true,
	GlobalSettingName[DbsEventsPassword]:        true,
	GlobalSettingName[DbsEventsHost]:            true,
	GlobalSettingName[DbsEventsPort]:            true,
	GlobalSettingName[DbsEventsMaxIdleConns]:    true,
	GlobalSettingName[DbsEventsMaxOpenConns]:    true,
	GlobalSettingName[DbsEventsConnMaxLifetime]: true,
}

// GlobalSettingInfo Indicates the type of each global settings, and whether it is possible to change each setting
var GlobalSettingInfo = map[string]struct {
	settingType smartcontract.ConfigType

	// Indicates that the settings cannot be changed by a transaction
	// This includes both true immutable settings and settings that are local
	// and are changed by restarting editing 0chain.yaml and restarting the module
	// todo we need to split up immutable and stored in MPT and local so can't be changed in transaction
	mutable bool
}{
	GlobalSettingName[State]:                             {smartcontract.Boolean, false},
	GlobalSettingName[Dkg]:                               {smartcontract.Boolean, false},
	GlobalSettingName[ViewChange]:                        {smartcontract.Boolean, false},
	GlobalSettingName[BlockRewards]:                      {smartcontract.Boolean, false},
	GlobalSettingName[Storage]:                           {smartcontract.Boolean, false},
	GlobalSettingName[Faucet]:                            {smartcontract.Boolean, false},
	GlobalSettingName[Miner]:                             {smartcontract.Boolean, false},
	GlobalSettingName[Multisig]:                          {smartcontract.Boolean, false},
	GlobalSettingName[Vesting]:                           {smartcontract.Boolean, false},
	GlobalSettingName[Zcn]:                               {smartcontract.Boolean, false},
	GlobalSettingName[Owner]:                             {smartcontract.String, false},
	GlobalSettingName[BlockMinSize]:                      {smartcontract.Int32, true},
	GlobalSettingName[BlockMaxSize]:                      {smartcontract.Int32, true},
	GlobalSettingName[BlockMaxCost]:                      {smartcontract.Int, true},
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
	GlobalSettingName[BlockGeneratorsPercent]:            {smartcontract.BigRational, true},
	GlobalSettingName[RoundRange]:                        {smartcontract.Int64, true},
	GlobalSettingName[RoundTimeoutsSofttoMin]:            {smartcontract.Int, true},
	GlobalSettingName[RoundTimeoutsSofttoMult]:           {smartcontract.Int, true},
	GlobalSettingName[RoundTimeoutsRoundRestartMult]:     {smartcontract.Int, true},
	GlobalSettingName[RoundTimeoutsTimeoutCap]:           {smartcontract.Int, false},
	GlobalSettingName[TransactionPayloadMaxSize]:         {smartcontract.Int, true},
	GlobalSettingName[TransactionTimeout]:                {smartcontract.Int, false},
	GlobalSettingName[TransactionMinFee]:                 {smartcontract.Int64, false},
	GlobalSettingName[TransactionExempt]:                 {smartcontract.Strings, true},
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

	GlobalSettingName[DbsEventsEnabled]:         {smartcontract.Boolean, false},
	GlobalSettingName[DbsEventsName]:            {smartcontract.String, false},
	GlobalSettingName[DbsEventsUser]:            {smartcontract.String, false},
	GlobalSettingName[DbsEventsPassword]:        {smartcontract.String, false},
	GlobalSettingName[DbsEventsHost]:            {smartcontract.String, false},
	GlobalSettingName[DbsEventsPort]:            {smartcontract.String, false},
	GlobalSettingName[DbsEventsMaxIdleConns]:    {smartcontract.Int, false},
	GlobalSettingName[DbsEventsMaxOpenConns]:    {smartcontract.Int, false},
	GlobalSettingName[DbsEventsConnMaxLifetime]: {smartcontract.Duration, false},

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

func (gl *GlobalSettings) update(inputMap smartcontract.StringMap) error {
	var err error
	for key, value := range inputMap.Fields {
		info, found := GlobalSettingInfo[key]
		if !found {
			return fmt.Errorf("'%s' is not a valid global setting", key)
		}
		if !info.mutable {
			return fmt.Errorf("%s cannot be modified via a transaction", key)
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

// GetCoin returns a global setting as an currency.Coin, a check is made to confirm the setting's type.
func (gl *GlobalSettings) GetCoin(field GlobalSetting) (currency.Coin, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return 0, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return currency.Coin(viper.GetUint64(key)), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.CurrencyCoin)
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
func (gl *GlobalSettings) GetBigRat(field GlobalSetting) (zbig.BigRat, error) {
	key, err := getGlobalSettingName(field)
	if err != nil {
		return zbig.BigRat{}, err
	}

	sValue, found := gl.Fields[key]
	if !found {
		return zbig.BigRatFromFloat64(viper.GetFloat64(key)), nil
	}
	iValue, err := smartcontract.StringToInterface(sValue, smartcontract.BigRational)
	if err != nil {
		return zbig.BigRatFromFloat64(viper.GetFloat64(key)), nil
	}
	value, ok := iValue.(zbig.BigRat)
	if !ok {
		return zbig.BigRat{}, fmt.Errorf("cannot convert key %s value %v to type int", key, value)
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
	for key := range GlobalSettingInfo {
		if _, ok := GlobalSettingsIgnored[key]; ok {
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
