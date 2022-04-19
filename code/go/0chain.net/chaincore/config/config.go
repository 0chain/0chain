package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"0chain.net/core/viper"
)

var (
	// SmartContractConfig stores the wrapper Viper global instance
	// for the blockchain smart contracts configuration system.
	SmartContractConfig = viper.New()
)

//SetupDefaultConfig - setup the default config options that can be overridden via the config file
func SetupDefaultConfig() {
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("network.relay_time", 200)
	viper.SetDefault("network.timeout.small_message", 500)
	viper.SetDefault("network.timeout.large_message", 1000)
	viper.SetDefault("network.large_message_th_size", 10240)
	viper.SetDefault("server_chain.messages.verification_tickets_to", "generator")
	viper.SetDefault("server_chain.round_range", 10000000)
	viper.SetDefault("server_chain.transaction.payload.max_size", 32)
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

func SetupDefaultSmartContractConfig() {
	SmartContractConfig.SetDefault("smart_contracts.faucetsc.pour_limit", 10000)
	SmartContractConfig.SetDefault("smart_contracts.faucetsc.periodic_limit", 1000000)
	SmartContractConfig.SetDefault("smart_contracts.faucetsc.global_limit", 100000000)
	SmartContractConfig.SetDefault("smart_contracts.faucetsc.individual_reset", "2h")
	SmartContractConfig.SetDefault("smart_contracts.faucetsc.global_reset", "24h")
	SmartContractConfig.SetDefault("smart_contracts.interestpoolsc.min_lock", 100)
	SmartContractConfig.SetDefault("smart_contracts.interestpoolsc.lock_period", "2160h")
	SmartContractConfig.SetDefault("smart_contracts.interestpoolsc.interest_rate", 0.01)

	SmartContractConfig.SetDefault("smart_contracts.storagesc.challenge_enabled", true)
	SmartContractConfig.SetDefault("smart_contracts.storagesc.challenge_rate_per_mb_min", 1)
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

type ConfigReader interface {
	ReadValue(name string) (interface{}, error)
}

// This is defined to avoid 'nil pointer reference' to ConfigChain
// And also to be used for setting the configuration for Test purposes
type TestConfigReader struct {
	Fields map[string]interface{}
	mu     sync.Mutex
}

func (cr *TestConfigReader) ReadValue(name string) (interface{}, error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	v, found := cr.Fields[name]
	if !found {
		return nil, errors.New("ChainConfig - Read Config - Invalid configuration name")
	}
	return v, nil
}

func (cr *TestConfigReader) WriteValue(name string, val interface{}) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.Fields[name] = val
}

/*Config - all the config options passed from the command line*/
type Config struct {
	Host           string
	Port           int
	ChainID        string
	DeploymentMode byte
	ChainConfig    ConfigReader
}

/*Configuration of the system */
var configuration Config

func Configuration() *Config {
	if configuration.ChainConfig == nil {
		configuration.ChainConfig = &TestConfigReader{
			Fields: map[string]interface{}{},
		}
	}
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
