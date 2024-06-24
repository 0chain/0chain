package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/0chain/common/core/currency"

	"0chain.net/core/viper"
)

// MaxTokenSupply represents the maximum token supply in SAS
const MaxTokenSupply = 4e18 // max token supply 400 million ZCN

var (
	// SmartContractConfig stores the wrapper Viper global instance
	// for the blockchain smart contracts configuration system.
	SmartContractConfig = viper.New()
)

// SetupDefaultConfig - setup the default config options that can be overridden via the config file
func SetupDefaultConfig() {
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("network.relay_time", 200)
	viper.SetDefault("network.timeout.small_message", 500)
	viper.SetDefault("network.timeout.large_message", 1000)
	viper.SetDefault("network.large_message_th_size", 10240)
	viper.SetDefault("server_chain.messages.verification_tickets_to", "generator")
	viper.SetDefault("server_chain.round_range", 10000000)
	viper.SetDefault("server_chain.transaction.payload.max_size", 32)
	viper.SetDefault("server_chain.transaction.transfer_cost", 10)
	viper.SetDefault("server_chain.transaction.cost_fee_coeff", 100000)
	viper.SetDefault("server_chain.transaction.future_nonce", 10)
	viper.SetDefault("server_chain.state.prune_below_count", 100)
	viper.SetDefault("server_chain.block.consensus.threshold_by_count", 66)
	viper.SetDefault("server_chain.block.generation.timeout", 37)
	viper.SetDefault("server_chain.state.sync.timeout", 10)
	viper.SetDefault("server_chain.stuck.check_interval", 10)
	viper.SetDefault("server_chain.stuck.time_threshold", 60)
	viper.SetDefault("server_chain.transaction.timeout", 30)
	viper.SetDefault("server_chain.block.generation.retry_wait_time", 5)
	viper.SetDefault("server_chain.block.proposal.max_wait_time", "200ms")
	viper.SetDefault("server_chain.block.proposal.wait_mode", "static")
	viper.SetDefault("server_chain.block.reuse_txns", true)
	viper.SetDefault("server_chain.client.signature_scheme", "ed25519")
	viper.SetDefault("server_chain.block.sharding.min_active_sharders", 100)
	viper.SetDefault("server_chain.block.sharding.min_active_replicators", 100)
	viper.SetDefault("server_chain.smart_contract.timeout", 550*time.Millisecond)
	viper.SetDefault("server_chain.smart_contract.setting_update_period", 200)
	viper.SetDefault("server_chain.smart_contract.timeout", "500ms")
	viper.SetDefault("server_chain.round_timeouts.softto_min", 300)
	viper.SetDefault("server_chain.round_timeouts.softto_mult", 3)
	viper.SetDefault("server_chain.round_timeouts.round_restart_mult", 2)
	// Health Check related fields
	viper.SetDefault("server_chain.health_check.show_counters", true)
	viper.SetDefault("server_chain.smart_contract.setting_update_period", true)
	// Set defaults for deep scan.
	viper.SetDefault("server_chain.health_check.deep_scan.enabled", true)
	viper.SetDefault("server_chain.health_check.deep_scan.batch_size", 100)
	viper.SetDefault("server_chain.health_check.deep_scan.window", 0)

	// Repeat deep scan every day
	viper.SetDefault("server_chain.health_check.deep_scan.settle_secs", "30s")
	viper.SetDefault("server_chain.health_check.deep_scan.repeat_interval_mins", "1440m")
	viper.SetDefault("server_chain.health_check.deep_scan.report_status_mins", "60s")

	//Set defaults for proximity scan.
	viper.SetDefault("server_chain.health_check.proximity_scan.enabled", true)
	viper.SetDefault("server_chain.health_check.proximity_scan.batch_size", 100)
	viper.SetDefault("server_chain.health_check.proximity_scan.window", 100000)

	// Repeat proximity every hour.
	viper.SetDefault("server_chain.health_check.proximity_scan.settle_secs", "30s")
	viper.SetDefault("server_chain.health_check.proximity_scan.repeat_interval_mins", "60m")
	viper.SetDefault("server_chain.health_check.deep_scan.report_status_mins", "15m")

	// LFB tickets.
	viper.SetDefault("server_chain.lfb_ticket.rebroadcast_timeout", "16s")
	viper.SetDefault("server_chain.lfb_ticket.ahead", 2)
	viper.SetDefault("server_chain.lfb_ticket.fb_fetching_lifetime", "10s")

	// Asynchronous blocks fetching.
	viper.SetDefault("async_blocks_fetching.max_simultaneous_from_miners", 100)
	viper.SetDefault("async_blocks_fetching.max_simultaneous_from_sharders", 30)

	viper.SetDefault("smart_contracts.storagesc.max_blobbers_per_allocation", 40)
	viper.SetDefault("smart_contracts.storagesc.max_blobber_select_for_challenge", 5)
	viper.SetDefault("smart_contracts.storagesc.challenge_generation_gap", 1)

	viper.SetDefault(GlobalSettingName[DbsAggregateDebug], false)
	viper.SetDefault(GlobalSettingName[DbsAggregatePeriod], 10)
	viper.SetDefault(GlobalSettingName[DbsAggregatePageLimit], 50)

	viper.SetDefault("kafka.host", "localhost:9092")
	viper.SetDefault("kafka.topic", "events")
	viper.SetDefault("kafka.write_timeout", 10*time.Second) // seconds
}

// SetupConfig setups the main configuration system.
func SetupConfig(workdir string) {
	file := filepath.Join(".", "config", "0chain.yaml")

	if len(workdir) > 0 {
		file = filepath.Join(workdir, "config", "0chain.yaml")
	}

	if err := viper.ReadConfigFile(file); err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	setupDevConfig()
}

// SetupSmartContractConfig setups the smart contracts configuration system.
func SetupSmartContractConfig(workdir string) {
	file := filepath.Join(".", "config", "sc.yaml")

	if len(workdir) > 0 {
		file = filepath.Join(workdir, "config", "sc.yaml")
	}

	if err := SmartContractConfig.ReadConfigFile(file); err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}

// ReadConfig reads a configuration from given file path.
func ReadConfig(file string) *viper.Viper {
	nodeConfig := viper.New()
	if err := nodeConfig.ReadConfigFile(file); err != nil {
		panic(fmt.Sprintf("error reading config file %v - %v\n", file, err))
	}

	return nodeConfig
}

const (
	DeploymentDevelopment = 0
	DeploymentTestNet     = 1
	DeploymentMainNet     = 2
)

//go:generate mockery --case underscore --name=ChainConfig --output=./mocks
type ChainConfig interface {
	IsStateEnabled() bool
	IsDkgEnabled() bool
	IsViewChangeEnabled() bool
	IsBlockRewardsEnabled() bool
	IsStorageEnabled() bool
	IsFaucetEnabled() bool
	IsInterestEnabled() bool
	IsFeeEnabled() bool
	IsMultisigEnabled() bool
	IsVestingEnabled() bool
	IsZcnEnabled() bool
	OwnerID() string
	MinBlockSize() int32
	MaxBlockCost() int
	MaxByteSize() int64
	MinGenerators() int
	GeneratorsPercent() float64
	NumReplicators() int
	ThresholdByCount() int
	ThresholdByStake() int
	ValidationBatchSize() int
	TxnMaxPayload() int
	PruneStateBelowCount() int
	RoundRange() int64
	BlocksToSharder() int
	VerificationTicketsTo() int
	HealthShowCounters() bool
	HCCycleScan() [2]HealthCheckCycleScan
	BlockProposalMaxWaitTime() time.Duration
	BlockProposalWaitMode() int8
	ReuseTransactions() bool
	ClientSignatureScheme() string
	MinActiveSharders() int
	MinActiveReplicators() int
	SmartContractTimeout() time.Duration
	SmartContractSettingUpdatePeriod() int64
	RoundTimeoutSofttoMin() int
	RoundTimeoutSofttoMult() int
	RoundRestartMult() int
	DbsEvents() DbAccess
	DbSettings() DbSettings
	FromViper() error
	Update(configMap map[string]string, version int64) error
	TxnExempt() map[string]bool
	MinTxnFee() currency.Coin
	MaxTxnFee() currency.Coin
	TxnTransferCost() int
	TxnCostFeeCoeff() int
	TxnFutureNonce() int
	BlockFinalizationTimeout() time.Duration
}

type DbAccess struct {
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`

	MaxIdleConns        int           `json:"max_idle_conns"`
	MaxOpenConns        int           `json:"max_open_conns"`
	ConnMaxLifetime     time.Duration `json:"conn_max_lifetime"`
	Slowtablespace      string        `json:"slowtablespace"`
	KafkaEnabled        bool
	KafkaHost           string
	KafkaTopic          string
	KafkaTopicPartition int
	KafkaUsername       string
	KafkaPassword       string
	KafkaWriteTimeout   time.Duration
	KafkaTriggerRound   int64
}

type DbSettings struct {
	Debug                          bool  `json:"debug"`
	AggregatePeriod                int64 `json:"aggregate_period"`
	PartitionChangePeriod          int64 `json:"partition_change_period"`
	PartitionKeepCount             int64 `json:"partition_keep_count"`
	PermanentPartitionChangePeriod int64 `json:"permanent_partition_change_period"`
	PermanentPartitionKeepCount    int64 `json:"permanent_partition_keep_count"`
	PageLimit                      int64 `json:"page_limit"`
}

func (s *DbSettings) Update(updates map[string]string) error {
	if value, found := updates[DbsAggregateDebug.String()]; found {
		iValue, err := StringToInterface(value, Boolean)
		if err != nil {
			return err
		}
		s.Debug = iValue.(bool)
	}
	if value, found := updates[DbsAggregatePeriod.String()]; found {
		iValue, err := StringToInterface(value, Int64)
		if err != nil {
			return err
		}
		s.AggregatePeriod = iValue.(int64)
	}
	if value, found := updates[DbsPartitionChangePeriod.String()]; found {
		iValue, err := StringToInterface(value, Int64)
		if err != nil {
			return err
		}
		s.PartitionChangePeriod = iValue.(int64)
	}
	if value, found := updates[DbsPartitionKeepCount.String()]; found {
		iValue, err := StringToInterface(value, Int64)
		if err != nil {
			return err
		}
		s.PartitionKeepCount = iValue.(int64)
	}
	if value, found := updates[DbsAggregatePageLimit.String()]; found {
		iValue, err := StringToInterface(value, Int64)
		if err != nil {
			return err
		}
		s.PageLimit = iValue.(int64)
	}
	return nil
}

// HealthCheckCycleScan -
type HealthCheckCycleScan struct {
	Settle time.Duration `json:"settle"`
	//SettleSecs int           `json:"settle_period_secs"`

	Enabled   bool  `json:"scan_enable"`
	BatchSize int64 `json:"batch_size"`

	Window int64 `json:"scan_window"`

	RepeatInterval time.Duration `json:"repeat_interval"`
	//RepeatIntervalMins int           `json:"repeat_interval_mins"`

	//ReportStatusMins int `json:"report_status_mins"`
	ReportStatus time.Duration `json:"report_status"`
}

/*Config - all the config options passed from the command line*/
type Config struct {
	Host           string
	Port           int
	ChainID        string
	DeploymentMode byte
	ChainConfig
}

/*Configuration of the system */
var configuration Config

func InitConfigurationGlobal(host, chainId string, port int, conf ChainConfig) {
	configuration = Config{
		Host:           host,
		Port:           port,
		ChainID:        chainId,
		DeploymentMode: DeploymentDevelopment,
		ChainConfig:    conf,
	}
}

func Configuration() *Config {
	return &configuration
}

/*TestNet - is the server running in TestNet mode? */
func TestNet() bool {
	return Configuration().DeploymentMode == DeploymentTestNet
}

/*Development - is the server running in development mode? */
func Development() bool {
	return Configuration().DeploymentMode == DeploymentDevelopment
}

/*MainNet - is the server running in mainnet mode? */
func MainNet() bool {
	return Configuration().DeploymentMode == DeploymentMainNet
}

/*ErrSupportedChain error for indicating which chain is supported by the server */
var ErrSupportedChain error

/*MAIN_CHAIN - the main 0chain.net blockchain id */
const MAIN_CHAIN = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe" // TODO:

/*GetMainChainID - get the main chain id */
func GetMainChainID() string {
	return MAIN_CHAIN
}

/*ServerChainID - the chain this server is responsible for */
var ServerChainID = ""

/*SetServerChainID  - set the chain this server is responsible for processing */
func SetServerChainID(chain string) {
	if chain == "" {
		ServerChainID = MAIN_CHAIN
	} else {
		ServerChainID = chain
	}
	ErrSupportedChain = fmt.Errorf("chain %v is not supported by this server", chain)
}

/*GetServerChainID - get the chain this server is responsible for processing */
func GetServerChainID() string {
	if ServerChainID == "" {
		return MAIN_CHAIN
	}
	return ServerChainID
}

/*ValidChain - Is this the chain this server is supposed to process? */
func ValidChain(chain string) error {
	result := chain == ServerChainID || (chain == "" && ServerChainID == MAIN_CHAIN)
	if result {
		return nil
	}
	return ErrSupportedChain
}

/*GetThresholdCount Gets the defined threshold count */
func GetThresholdCount() int {
	return viper.GetInt("server_chain.block.consensus.threshold_by_count")
}

// LFB tickets.

func GetReBroadcastLFBTicketTimeout() time.Duration {
	return viper.GetDuration("server_chain.lfb_ticket.rebroadcast_timeout")
}

func GetLFBTicketAhead() int {
	return viper.GetInt("server_chain.lfb_ticket.ahead")
}

func GetFBFetchingLifetime() time.Duration {
	return viper.GetDuration("server_chain.lfb_ticket.fb_fetching_lifetime")
}

// Asynchronous blocks fetching.

func AsyncBlocksFetchingMaxSimultaneousFromMiners() int {
	return viper.GetInt("async_blocks_fetching.max_simultaneous_from_miners")
}

func AsyncBlocksFetchingMaxSimultaneousFromSharders() int {
	return viper.GetInt("async_blocks_fetching.max_simultaneous_from_sharders")
}
