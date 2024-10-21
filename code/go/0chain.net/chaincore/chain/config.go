package chain

import (
	"sync"
	"time"

	config2 "0chain.net/core/config"
	"github.com/0chain/common/core/currency"

	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/minersc"

	"0chain.net/core/datastore"
)

const (
	//BlockProposalWaitStatic Static wait time for block proposals
	BlockProposalWaitStatic = 0
	//BlockProposalWaitDynamic Dyanamic wait time for block proposals
	BlockProposalWaitDynamic = iota

	// DefaultMaxTxnFee represents the default max transaction fee
	DefaultMaxTxnFee = 1e9
)

// HealthCheckScan - Set in 0chain.yaml
type HealthCheckScan int

// DeepScan -
const (
	DeepScan HealthCheckScan = iota
	ProximityScan
)

// Defaults
const (
	DefaultRetrySendNotarizedBlockNewMiner = 5
	DefaultCountPruneRoundStorage          = 5
)

type ConfigImpl struct {
	conf  *ConfigData
	guard sync.RWMutex
}

// FOR TEST PURPOSE ONLY
func (c *ConfigImpl) ConfDataForTest() *ConfigData {
	return c.conf
}

func UpdateConfigImpl(conf *ConfigImpl, data *ConfigData) {
	if data.BlockSize != 0 {
		conf.conf.BlockSize = data.BlockSize
	}
}

func NewConfigImpl(conf *ConfigData) *ConfigImpl {
	return &ConfigImpl{conf: conf}
}

func (c *ConfigImpl) IsStateEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsStateEnabled
}
func (c *ConfigImpl) IsDkgEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsDkgEnabled
}
func (c *ConfigImpl) IsViewChangeEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsViewChangeEnabled
}
func (c *ConfigImpl) IsBlockRewardsEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsBlockRewardsEnabled
}
func (c *ConfigImpl) IsStorageEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsStorageEnabled
}
func (c *ConfigImpl) IsFaucetEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsFaucetEnabled
}
func (c *ConfigImpl) IsInterestEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsInterestEnabled
}
func (c *ConfigImpl) IsFeeEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsFeeEnabled
}
func (c *ConfigImpl) IsMultisigEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsMultisigEnabled
}
func (c *ConfigImpl) IsVestingEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsVestingEnabled
}
func (c *ConfigImpl) IsZcnEnabled() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.IsZcnEnabled
}
func (c *ConfigImpl) OwnerID() datastore.Key {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.OwnerID
}

func (c *ConfigImpl) BlockSize() int32 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.BlockSize
}

func (c *ConfigImpl) MinBlockSize() int32 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MinBlockSize
}

func (c *ConfigImpl) MaxByteSize() int64 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MaxByteSize
}

func (c *ConfigImpl) MinGenerators() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MinGenerators
}

func (c *ConfigImpl) GeneratorsPercent() float64 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.GeneratorsPercent
}

func (c *ConfigImpl) NumReplicators() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.NumReplicators
}

func (c *ConfigImpl) ThresholdByCount() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.ThresholdByCount
}

func (c *ConfigImpl) ThresholdByStake() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.ThresholdByStake
}

func (c *ConfigImpl) ValidationBatchSize() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.ValidationBatchSize
}

func (c *ConfigImpl) TxnMaxPayload() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.TxnMaxPayload
}

func (c *ConfigImpl) PruneStateBelowCount() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.PruneStateBelowCount
}

func (c *ConfigImpl) RoundRange() int64 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.RoundRange
}

func (c *ConfigImpl) BlocksToSharder() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.BlocksToSharder
}

func (c *ConfigImpl) VerificationTicketsTo() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.VerificationTicketsTo
}

func (c *ConfigImpl) HealthShowCounters() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.HealthShowCounters
}

func (c *ConfigImpl) HCCycleScan() [2]config2.HealthCheckCycleScan {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.HCCycleScan
}

func (c *ConfigImpl) BlockProposalMaxWaitTime() time.Duration {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.BlockProposalMaxWaitTime
}

func (c *ConfigImpl) BlockProposalWaitMode() int8 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.BlockProposalWaitMode
}

func (c *ConfigImpl) ReuseTransactions() bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.ReuseTransactions
}

func (c *ConfigImpl) ClientSignatureScheme() string {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.ClientSignatureScheme
}

func (c *ConfigImpl) MinActiveSharders() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MinActiveSharders
}

func (c *ConfigImpl) MinActiveReplicators() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MinActiveReplicators
}

func (c *ConfigImpl) SmartContractTimeout() time.Duration {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.SmartContractTimeout
}

func (c *ConfigImpl) SmartContractSettingUpdatePeriod() int64 {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.SmartContractSettingUpdatePeriod
}

func (c *ConfigImpl) RoundTimeoutSofttoMin() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.RoundTimeoutSofttoMin
}

func (c *ConfigImpl) RoundTimeoutSofttoMult() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.RoundTimeoutSofttoMult
}

func (c *ConfigImpl) RoundRestartMult() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.RoundRestartMult
}

func (c *ConfigImpl) DbsEvents() config2.DbAccess {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.DbsEvents
}

func (c *ConfigImpl) DbSettings() config2.DbSettings {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.DbsSettings
}

func (c *ConfigImpl) MaxBlockCost() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MaxBlockCost
}

func (c *ConfigImpl) TxnExempt() map[string]bool {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.TxnExempt
}

func (c *ConfigImpl) MinTxnFee() currency.Coin {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MinTxnFee
}

func (c *ConfigImpl) MaxTxnFee() currency.Coin {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.MaxTxnFee
}

func (c *ConfigImpl) TxnTransferCost() int {
	c.guard.RLock()
	defer c.guard.RUnlock()

	return c.conf.TxnTransferCost
}

func (c *ConfigImpl) TxnCostFeeCoeff() int {
	c.guard.RLock()
	coeff := c.conf.TxnCostFeeCoeff
	c.guard.RUnlock()

	return coeff
}

func (c *ConfigImpl) TxnFutureNonce() int {
	c.guard.RLock()
	fn := c.conf.TxnFutureNonce
	c.guard.RUnlock()
	return fn
}

func (c *ConfigImpl) BlockFinalizationTimeout() time.Duration {
	c.guard.RLock()
	t := c.conf.BlockFinalizationTimeout
	c.guard.RUnlock()
	return t
}

// ConfigData - chain Configuration
type ConfigData struct {
	version               int64         `json:"-"` //version of config to track updates
	IsStateEnabled        bool          `json:"state"`
	IsDkgEnabled          bool          `json:"dkg"`
	IsViewChangeEnabled   bool          `json:"view_change"`
	IsBlockRewardsEnabled bool          `json:"block_rewards"`
	IsStorageEnabled      bool          `json:"storage"`
	IsFaucetEnabled       bool          `json:"faucet"`
	IsInterestEnabled     bool          `json:"interest"`
	IsFeeEnabled          bool          `json:"miner"` // Indicates is fees enabled
	IsMultisigEnabled     bool          `json:"multisig"`
	IsVestingEnabled      bool          `json:"vesting"`
	IsZcnEnabled          bool          `json:"zcn"`
	OwnerID               datastore.Key `json:"owner_id"`                  // Client who created this chain
	BlockSize             int32         `json:"block_size"`                // Number of transactions in a block
	MinBlockSize          int32         `json:"min_block_size"`            // Number of transactions a block needs to have
	MaxBlockCost          int           `json:"max_block_cost"`            // multiplier of soft timeouts to restart a round
	MaxByteSize           int64         `json:"max_byte_size"`             // Max number of bytes a block can have
	MinGenerators         int           `json:"min_generators"`            // Min number of block generators.
	GeneratorsPercent     float64       `json:"generators_percent"`        // Percentage of all miners
	NumReplicators        int           `json:"num_replicators"`           // Number of sharders that can store the block
	ThresholdByCount      int           `json:"threshold_by_count"`        // Threshold count for a block to be notarized
	ThresholdByStake      int           `json:"threshold_by_stake"`        // Stake threshold for a block to be notarized
	ValidationBatchSize   int           `json:"validation_size"`           // Batch size of txns for crypto verification
	TxnMaxPayload         int           `json:"transaction_max_payload"`   // Max payload allowed in the transaction
	TxnTransferCost       int           `json:"transaction_transfer_cost"` // Transaction transfer cost
	TxnCostFeeCoeff       int           `json:"txn_cost_fee_coeff"`        // Transaction cost fee coefficient
	TxnFutureNonce        int           `json:"future_nonce"`              // Future transaction nonce allowed
	MinTxnFee             currency.Coin `json:"min_txn_fee"`               // Minimum txn fee allowed
	MaxTxnFee             currency.Coin `json:"max_txn_fee"`               // Maximum txn fee allowed
	PruneStateBelowCount  int           `json:"prune_state_below_count"`   // Prune state below these many rounds
	RoundRange            int64         `json:"round_range"`               // blocks are stored in separate directory for each range of rounds

	// todo move BlocksToSharder out of ConfigData
	BlocksToSharder       int `json:"blocks_to_sharder"`       // send finalized or notarized blocks to sharder
	VerificationTicketsTo int `json:"verification_tickets_to"` // send verification tickets to generator or all miners

	HealthShowCounters bool `json:"health_show_counters"` // display detail counters
	// Health Check switches
	HCCycleScan [2]config2.HealthCheckCycleScan

	BlockProposalMaxWaitTime time.Duration `json:"block_proposal_max_wait_time"` // max time to wait to receive a block proposal
	BlockProposalWaitMode    int8          `json:"block_proposal_wait_mode"`     // wait time for the block proposal is static (0) or dynamic (1)

	ReuseTransactions        bool          `json:"reuse_txns"`                 // indicates if transactions from unrelated blocks can be reused
	BlockFinalizationTimeout time.Duration `json:"block_finalization_timeout"` // time after which the block finalization will timeout

	ClientSignatureScheme string `json:"client_signature_scheme"` // indicates which signature scheme is being used

	MinActiveSharders    int `json:"min_active_sharders"`    // Minimum active sharders required to validate blocks
	MinActiveReplicators int `json:"min_active_replicators"` // Minimum active replicators of a block that should be active to verify the block

	SmartContractTimeout             time.Duration `json:"smart_contract_timeout"`               // time after which the smart contract execution will timeout
	SmartContractSettingUpdatePeriod int64         `json:"smart_contract_setting_update_period"` // rounds settings are updated

	RoundTimeoutSofttoMin  int `json:"softto_min"`         // minimum time for softtimeout to kick in milliseconds
	RoundTimeoutSofttoMult int `json:"softto_mult"`        // multiplier of mean network time for soft timeout
	RoundRestartMult       int `json:"round_restart_mult"` // multiplier of soft timeouts to restart a round

	DbsEvents   config2.DbAccess   `json:"dbs_event"`
	DbsSettings config2.DbSettings `json:"dbs_settings"`
	TxnExempt   map[string]bool    `json:"txn_exempt"`
}

func (c *ConfigImpl) FromViper() error {
	c.guard.Lock()
	defer c.guard.Unlock()

	if err := viper.BindEnv("server_chain.dbs.events.host", "POSTGRES_HOST"); err != nil {
		logging.Logger.Error("error during BindEnv", zap.Error(err))
	}
	if err := viper.BindEnv("server_chain.dbs.events.port", "POSTGRES_PORT"); err != nil {
		logging.Logger.Error("error during BindEnv", zap.Error(err))
	}

	if err := viper.BindEnv("server_chain.kafka.host", "KAFKA_HOSTNAME"); err != nil {
		logging.Logger.Error("error during BindEnv", zap.Error(err))
	}
	if err := viper.BindEnv("server_chain.kafka.username", "KAFKA_USERNAME"); err != nil {
		logging.Logger.Error("error during BindEnv", zap.Error(err))
	}
	if err := viper.BindEnv("server_chain.kafka.password", "KAFKA_PASSWORD"); err != nil {
		logging.Logger.Error("error during BindEnv", zap.Error(err))
	}

	conf := c.conf
	conf.IsStateEnabled = viper.GetBool("server_chain.state.enabled")
	conf.IsDkgEnabled = viper.GetBool("server_chain.dkg")
	conf.IsViewChangeEnabled = viper.GetBool("server_chain.view_change")
	conf.IsBlockRewardsEnabled = viper.GetBool("server_chain.block_rewards")
	conf.IsStorageEnabled = viper.GetBool("server_chain.smart_contract.storage")
	conf.IsFaucetEnabled = viper.GetBool("server_chain.smart_contract.faucet")
	conf.IsInterestEnabled = viper.GetBool("server_chain.smart_contract.interest")
	conf.IsFeeEnabled = viper.GetBool("server_chain.smart_contract.miner")
	conf.IsMultisigEnabled = viper.GetBool("server_chain.smart_contract.multisig")
	conf.IsVestingEnabled = viper.GetBool("server_chain.smart_contract.vesting")
	conf.IsZcnEnabled = viper.GetBool("server_chain.smart_contract.zcn")
	conf.BlockSize = viper.GetInt32("server_chain.block.max_block_size")
	conf.MinBlockSize = viper.GetInt32("server_chain.block.min_block_size")
	conf.MaxBlockCost = viper.GetInt("server_chain.block.max_block_cost")
	conf.MaxByteSize = viper.GetInt64("server_chain.block.max_byte_size")
	conf.MinGenerators = viper.GetInt("server_chain.block.min_generators")
	conf.GeneratorsPercent = viper.GetFloat64("server_chain.block.generators_percent")
	conf.NumReplicators = viper.GetInt("server_chain.block.replicators")
	conf.ThresholdByCount = viper.GetInt("server_chain.block.consensus.threshold_by_count")
	conf.ThresholdByStake = viper.GetInt("server_chain.block.consensus.threshold_by_stake")
	conf.OwnerID = viper.GetString("server_chain.owner")
	conf.ValidationBatchSize = viper.GetInt("server_chain.block.validation.batch_size")
	conf.RoundRange = viper.GetInt64("server_chain.round_range")
	conf.TxnMaxPayload = viper.GetInt("server_chain.transaction.payload.max_size")
	var err error
	conf.MinTxnFee, err = currency.ParseZCN(viper.GetFloat64("server_chain.transaction.min_fee"))
	if err != nil {
		return err
	}
	conf.MaxTxnFee, err = currency.ParseZCN(viper.GetFloat64("server_chain.transaction.max_fee"))
	if err != nil {
		return err
	}
	if conf.MaxTxnFee == 0 {
		conf.MaxTxnFee = DefaultMaxTxnFee
	}

	conf.TxnTransferCost = viper.GetInt("server_chain.transaction.transfer_cost")
	conf.TxnCostFeeCoeff = viper.GetInt("server_chain.transaction.cost_fee_coeff")
	conf.TxnFutureNonce = viper.GetInt("server_chain.transaction.future_nonce")
	txnExp := viper.GetStringSlice("server_chain.transaction.exempt")
	conf.TxnExempt = make(map[string]bool)
	for i := range txnExp {
		conf.TxnExempt[txnExp[i]] = true
	}
	conf.PruneStateBelowCount = viper.GetInt("server_chain.state.prune_below_count")

	verificationTicketsTo := viper.GetString("server_chain.messages.verification_tickets_to")
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		conf.VerificationTicketsTo = AllMiners
	} else {
		conf.VerificationTicketsTo = Generator
	}

	// Health Check related counters
	// Work on deep scan
	hc := &conf.HCCycleScan[DeepScan]

	hc.Enabled = viper.GetBool("server_chain.health_check.deep_scan.enabled")
	hc.BatchSize = viper.GetInt64("server_chain.health_check.deep_scan.batch_size")
	hc.Window = viper.GetInt64("server_chain.health_check.deep_scan.window")

	hc.Settle = viper.GetDuration("server_chain.health_check.deep_scan.settle_secs")
	hc.RepeatInterval = viper.GetDuration("server_chain.health_check.deep_scan.repeat_interval_mins")
	hc.ReportStatus = viper.GetDuration("server_chain.health_check.deep_scan.report_status_mins")

	// Work on proximity scan
	hc = &conf.HCCycleScan[ProximityScan]

	hc.Enabled = viper.GetBool("server_chain.health_check.proximity_scan.enabled")
	hc.BatchSize = viper.GetInt64("server_chain.health_check.proximity_scan.batch_size")
	hc.Window = viper.GetInt64("server_chain.health_check.proximity_scan.window")

	hc.Settle = viper.GetDuration("server_chain.health_check.proximity_scan.settle_secs")
	hc.RepeatInterval = viper.GetDuration("server_chain.health_check.proximity_scan.repeat_interval_mins")
	hc.ReportStatus = viper.GetDuration("server_chain.health_check.proximity_scan.report_status_mins")

	conf.HealthShowCounters = viper.GetBool("server_chain.health_check.show_counters")

	conf.BlockProposalMaxWaitTime = viper.GetDuration("server_chain.block.proposal.max_wait_time")
	waitMode := viper.GetString("server_chain.block.proposal.wait_mode")
	if waitMode == "static" {
		conf.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		conf.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	conf.ReuseTransactions = viper.GetBool("server_chain.block.reuse_txns")
	conf.BlockFinalizationTimeout = viper.GetDuration("server_chain.block.finalization.timeout")

	conf.MinActiveSharders = viper.GetInt("server_chain.block.sharding.min_active_sharders")
	conf.MinActiveReplicators = viper.GetInt("server_chain.block.sharding.min_active_replicators")
	conf.SmartContractTimeout = viper.GetDuration("server_chain.smart_contract.timeout")
	if conf.SmartContractTimeout == 0 {
		conf.SmartContractTimeout = DefaultSmartContractTimeout
	}
	conf.SmartContractSettingUpdatePeriod = viper.GetInt64("server_chain.smart_contract.setting_update_period")
	conf.RoundTimeoutSofttoMin = viper.GetInt("server_chain.round_timeouts.softto_min")
	conf.RoundTimeoutSofttoMult = viper.GetInt("server_chain.round_timeouts.softto_mult")
	conf.RoundRestartMult = viper.GetInt("server_chain.round_timeouts.round_restart_mult")
	conf.ClientSignatureScheme = viper.GetString("server_chain.client.signature_scheme")

	conf.DbsEvents.Enabled = viper.GetBool("server_chain.dbs.events.enabled")
	conf.DbsEvents.Name = viper.GetString("server_chain.dbs.events.name")
	conf.DbsEvents.User = viper.GetString("server_chain.dbs.events.user")
	conf.DbsEvents.Password = viper.GetString("server_chain.dbs.events.password")
	conf.DbsEvents.Host = viper.GetString("server_chain.dbs.events.host")
	conf.DbsEvents.Port = viper.GetString("server_chain.dbs.events.port")
	conf.DbsEvents.MaxIdleConns = viper.GetInt("server_chain.dbs.events.max_idle_conns")
	conf.DbsEvents.MaxOpenConns = viper.GetInt("server_chain.dbs.events.max_open_conns")
	conf.DbsEvents.ConnMaxLifetime = viper.GetDuration("server_chain.dbs.events.conn_max_lifetime")
	conf.DbsEvents.Slowtablespace = viper.GetString("server_chain.dbs.events.slowtablespace")
	if conf.DbsEvents.Slowtablespace == "" {
		conf.DbsEvents.Slowtablespace = "hddtablespace"
	}
	conf.DbsEvents.KafkaEnabled = viper.GetBool("kafka.enabled")
	conf.DbsEvents.KafkaHost = viper.GetString("server_chain.kafka.host")
	conf.DbsEvents.KafkaTopic = viper.GetString("kafka.topic")
	conf.DbsEvents.KafkaTopicPartition = viper.GetInt("kafka.partition")
	conf.DbsEvents.KafkaUsername = viper.GetString("server_chain.kafka.username")
	conf.DbsEvents.KafkaPassword = viper.GetString("server_chain.kafka.password")
	conf.DbsEvents.KafkaWriteTimeout = viper.GetDuration("kafka.write_timeout")
	conf.DbsEvents.KafkaTriggerRound = viper.GetInt64("kafka.trigger_round")
	conf.DbsSettings.Debug = viper.GetBool("server_chain.dbs.settings.debug")
	conf.DbsSettings.AggregatePeriod = viper.GetInt64("server_chain.dbs.settings.aggregate_period")
	conf.DbsSettings.PartitionChangePeriod = viper.GetInt64("server_chain.dbs.settings.partition_change_period")
	conf.DbsSettings.PartitionKeepCount = viper.GetInt64("server_chain.dbs.settings.partition_keep_count")
	conf.DbsSettings.PermanentPartitionChangePeriod = viper.GetInt64("server_chain.dbs.settings.permanent_partition_change_period")
	conf.DbsSettings.PermanentPartitionKeepCount = viper.GetInt64("server_chain.dbs.settings.permanent_partition_keep_count")
	conf.DbsSettings.PageLimit = viper.GetInt64("server_chain.dbs.settings.page_limit")
	return nil
}

// Updates the config fields from GlobalSettings fields
func (c *ConfigImpl) Update(fields map[string]string, version int64) error {
	c.guard.Lock()
	defer c.guard.Unlock()

	cf := &minersc.GlobalSettings{Fields: fields, Version: version}

	conf := c.conf
	old := conf.version
	if old >= cf.Version {
		return nil
	}

	conf.version = cf.Version
	logging.Logger.Debug("Updating config", zap.Int64("old version", old), zap.Int64("new version", conf.version))

	var err error
	conf.IsStateEnabled, err = cf.GetBool(config2.State)
	if err != nil {
		return err
	}
	conf.IsDkgEnabled, err = cf.GetBool(config2.Dkg)
	if err != nil {
		return err
	}
	conf.IsViewChangeEnabled, err = cf.GetBool(config2.ViewChange)
	if err != nil {
		return err
	}
	conf.IsBlockRewardsEnabled, err = cf.GetBool(config2.BlockRewards)
	if err != nil {
		return err
	}
	conf.IsStorageEnabled, err = cf.GetBool(config2.Storage)
	if err != nil {
		return err
	}
	conf.IsFaucetEnabled, err = cf.GetBool(config2.Faucet)
	if err != nil {
		return err
	}
	conf.IsFeeEnabled, err = cf.GetBool(config2.Miner)
	if err != nil {
		return err
	}
	conf.IsMultisigEnabled, err = cf.GetBool(config2.Multisig)
	if err != nil {
		return err
	}
	conf.IsVestingEnabled, err = cf.GetBool(config2.Vesting)
	if err != nil {
		return err
	}
	conf.IsZcnEnabled, err = cf.GetBool(config2.Zcn)
	if err != nil {
		return err
	}
	conf.MinBlockSize, err = cf.GetInt32(config2.BlockMinSize)
	if err != nil {
		return err
	}
	conf.BlockSize, err = cf.GetInt32(config2.BlockMaxSize)
	if err != nil {
		return err
	}
	conf.MaxBlockCost, err = cf.GetInt(config2.BlockMaxCost)
	if err != nil {
		return err
	}
	conf.MaxByteSize, err = cf.GetInt64(config2.BlockMaxByteSize)
	if err != nil {
		return err
	}
	conf.NumReplicators, err = cf.GetInt(config2.BlockReplicators)
	if err != nil {
		return err
	}
	conf.BlockProposalMaxWaitTime, err = cf.GetDuration(config2.BlockProposalMaxWaitTime)
	if err != nil {
		return err
	}
	waitMode, err := cf.GetString(config2.BlockProposalWaitMode)
	if err != nil {
		return err
	}
	if waitMode == "static" {
		conf.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		conf.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	conf.ThresholdByCount, err = cf.GetInt(config2.BlockConsensusThresholdByCount)
	if err != nil {
		return err
	}
	conf.ThresholdByStake, err = cf.GetInt(config2.BlockConsensusThresholdByStake)
	if err != nil {
		return err
	}
	conf.MinActiveSharders, err = cf.GetInt(config2.BlockShardingMinActiveSharders)
	if err != nil {
		return err
	}
	conf.MinActiveReplicators, err = cf.GetInt(config2.BlockShardingMinActiveReplicators)
	if err != nil {
		return err
	}
	conf.ValidationBatchSize, err = cf.GetInt(config2.BlockValidationBatchSize)
	if err != nil {
		return err
	}
	conf.ReuseTransactions, err = cf.GetBool(config2.BlockReuseTransactions)
	if err != nil {
		return err
	}
	conf.BlockFinalizationTimeout, err = cf.GetDuration(config2.BlockFinalizationTimeout)
	if err != nil {
		return err
	}
	conf.MinGenerators, err = cf.GetInt(config2.BlockMinGenerators)
	if err != nil {
		return err
	}
	conf.GeneratorsPercent, err = cf.GetFloat64(config2.BlockGeneratorsPercent)
	if err != nil {
		return err
	}
	conf.RoundRange, err = cf.GetInt64(config2.RoundRange)
	if err != nil {
		return err
	}
	conf.RoundTimeoutSofttoMin, err = cf.GetInt(config2.RoundTimeoutsSofttoMin)
	if err != nil {
		return err
	}
	conf.RoundTimeoutSofttoMult, err = cf.GetInt(config2.RoundTimeoutsSofttoMult)
	if err != nil {
		return err
	}
	conf.RoundRestartMult, err = cf.GetInt(config2.RoundTimeoutsRoundRestartMult)
	if err != nil {
		return err
	}
	conf.TxnMaxPayload, err = cf.GetInt(config2.TransactionPayloadMaxSize)
	if err != nil {
		return err
	}

	minTxnFeeF, err := cf.GetFloat64(config2.TransactionMinFee)
	if err != nil {
		return err
	}

	minTxnFee, err := currency.ParseZCN(minTxnFeeF)
	if err != nil {
		return err
	}
	conf.MinTxnFee = minTxnFee

	// get max txn fee from cf and parse it to currency.Coin
	maxTxnFeeF, err := cf.GetFloat64(config2.TransactionMaxFee)
	if err != nil {
		return err
	}

	maxTxnFee, err := currency.ParseZCN(maxTxnFeeF)
	if err != nil {
		return err
	}

	if maxTxnFee == 0 {
		maxTxnFee, _ = currency.ParseZCN(DefaultMaxTxnFee)
	}

	conf.MaxTxnFee = maxTxnFee

	conf.ClientSignatureScheme, err = cf.GetString(config2.ClientSignatureScheme)
	if err != nil {
		return err
	}
	verificationTicketsTo, err := cf.GetString(config2.MessagesVerificationTicketsTo)
	if err != nil {
		return err
	}
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		conf.VerificationTicketsTo = AllMiners
	} else {
		conf.VerificationTicketsTo = Generator
	}
	conf.PruneStateBelowCount, err = cf.GetInt(config2.StatePruneBelowCount)
	if err != nil {
		return err
	}
	conf.SmartContractTimeout, err = cf.GetDuration(config2.SmartContractTimeout)
	if err != nil {
		return err
	}
	if conf.SmartContractTimeout == 0 {
		conf.SmartContractTimeout = DefaultSmartContractTimeout
	}
	conf.SmartContractSettingUpdatePeriod, err = cf.GetInt64(config2.SmartContractSettingUpdatePeriod)
	if err != nil {
		return err
	}
	if txnsExempted, err := cf.GetStrings(config2.TransactionExempt); err != nil {
		return err
	} else {
		conf.TxnExempt = make(map[string]bool)
		for i := range txnsExempted {
			conf.TxnExempt[txnsExempted[i]] = true
		}
	}
	return nil
}

// We don't need this yet, as the health check settings are used to set up a worker thread.
func (conf *ConfigData) UpdateHealthCheckSettings(cf *minersc.GlobalSettings) error {
	var err error
	conf.HealthShowCounters, err = cf.GetBool(config2.HealthCheckShowCounters)
	if err != nil {
		return err
	}
	ds := &conf.HCCycleScan[DeepScan]
	ds.Enabled, err = cf.GetBool(config2.HealthCheckDeepScanEnabled)
	if err != nil {
		return err
	}
	ds.BatchSize, err = cf.GetInt64(config2.HealthCheckDeepScanBatchSize)
	if err != nil {
		return err
	}
	ds.Window, err = cf.GetInt64(config2.HealthCheckDeepScanWindow)
	if err != nil {
		return err
	}
	ds.Settle, err = cf.GetDuration(config2.HealthCheckDeepScanSettleSecs)
	if err != nil {
		return err
	}
	ds.RepeatInterval, err = cf.GetDuration(config2.HealthCheckDeepScanIntervalMins)
	if err != nil {
		return err
	}
	ds.ReportStatus, err = cf.GetDuration(config2.HealthCheckDeepScanReportStatusMins)
	if err != nil {
		return err
	}

	ps := &conf.HCCycleScan[ProximityScan]
	ps.Enabled, err = cf.GetBool(config2.HealthCheckProximityScanEnabled)
	if err != nil {
		return err
	}
	ps.BatchSize, err = cf.GetInt64(config2.HealthCheckProximityScanBatchSize)
	if err != nil {
		return err
	}
	ps.Window, err = cf.GetInt64(config2.HealthCheckProximityScanWindow)
	if err != nil {
		return err
	}
	ps.Settle, err = cf.GetDuration(config2.HealthCheckProximityScanSettleSecs)
	if err != nil {
		return err
	}
	ps.RepeatInterval, err = cf.GetDuration(config2.HealthCheckProximityScanRepeatIntervalMins)
	if err != nil {
		return err
	}
	ps.ReportStatus, err = cf.GetDuration(config2.HealthCheckProximityScanRejportStatusMins)
	if err != nil {
		return err
	}
	return nil
}
