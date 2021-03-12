package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

var SmartContractConfig *viper.Viper

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
	viper.SetDefault("server_chain.block.consensus.threshold_by_count", 67)
	viper.SetDefault("server_chain.block.generation.timeout", 37)
	viper.SetDefault("server_chain.state.sync.timeout", 10)
	viper.SetDefault("server_chain.stuck.check_interval", 10)
	viper.SetDefault("server_chain.stuck.time_threshold", 60)
	viper.SetDefault("server_chain.transaction.timeout", 30)
	viper.SetDefault("server_chain.block.generation.retry_wait_time", 5)
	viper.SetDefault("server_chain.block.proposal.max_wait_time", 200)
	viper.SetDefault("server_chain.block.proposal.wait_mode", "static")
	viper.SetDefault("server_chain.block.reuse_txns", true)
	viper.SetDefault("server_chain.client.signature_scheme", "ed25519")
	viper.SetDefault("server_chain.block.sharding.min_active_sharders", 100)
	viper.SetDefault("server_chain.block.sharding.min_active_replicators", 100)
	viper.SetDefault("server_chain.smart_contract.timeout", 550)
	viper.SetDefault("server_chain.round_timeouts.softto_min", 300)
	viper.SetDefault("server_chain.round_timeouts.softto_mult", 3)
	viper.SetDefault("server_chain.round_timeouts.round_restart_mult", 2)
	// Health Check related fields
	viper.SetDefault("server_chain.health_check.show_counters", true)

	// Set defaults for deep scan.
	viper.SetDefault("server_chain.health_check.deep_scan.enabled", true)
	viper.SetDefault("server_chain.health_check.deep_scan.batch_size", 100)
	viper.SetDefault("server_chain.health_check.deep_scan.window", 0)

	// Repeat deep scan every day
	viper.SetDefault("server_chain.health_check.deep_scan.settle_secs", 30)
	viper.SetDefault("server_chain.health_check.deep_scan.repeat_interval_mins", 1440)
	viper.SetDefault("server_chain.health_check.deep_scan.report_status_mins", 60)

	//Set defaults for proximity scan.
	viper.SetDefault("server_chain.health_check.proximity_scan.enabled", true)
	viper.SetDefault("server_chain.health_check.proximity_scan.batch_size", 100)
	viper.SetDefault("server_chain.health_check.proximity_scan.window", 100000)

	// Repeat proximity every hour.
	viper.SetDefault("server_chain.health_check.proximity_scan.settle_secs", 30)
	viper.SetDefault("server_chain.health_check.proximity_scan.repeat_interval_mins", 60)
	viper.SetDefault("server_chain.health_check.deep_scan.report_status_mins", 15)

	// LFB tickets.
	viper.SetDefault("server_chain.lfb_ticket.rebroadcast_timeout", time.Second*16)
	viper.SetDefault("server_chain.lfb_ticket.ahead", 2)
	viper.SetDefault("server_chain.lfb_ticket.fb_fetching_lifetime", time.Second*10)

	// Asynchronous blocks fetching.
	viper.SetDefault("async_blocks_fetching.max_simultaneous_from_miners", 100)
	viper.SetDefault("async_blocks_fetching.max_simultaneous_from_sharders", 30)
}

/*SetupConfig - setup the configuration system */
func SetupConfig() {
	viper.SetConfigName("0chain")
	viper.AddConfigPath("./config")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
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

/*SetupConfig - setup the configuration system */
func SetupSmartContractConfig() {
	SmartContractConfig = viper.New()
	SmartContractConfig.SetConfigName("sc")
	SmartContractConfig.AddConfigPath("./config")
	err := SmartContractConfig.ReadInConfig() // Find and read the config file
	if err != nil {                           // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}

//ReadConfig - read a configuration from a file given as path/to/config/dir/config.configtype
func ReadConfig(file string) *viper.Viper {
	dir, fileName := filepath.Split(file)
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".yaml"
	} else {
		fileName = fileName[:len(fileName)-len(ext)]
	}
	format := ext[1:]
	if dir == "" {
		dir = "."
	} else if dir[0] != '.' {
		dir = "." + string(filepath.Separator) + dir
	}
	nodeConfig := viper.New()
	nodeConfig.AddConfigPath(dir)
	nodeConfig.SetConfigName(fileName)
	nodeConfig.SetConfigType(format)
	err := nodeConfig.ReadInConfig()
	if err != nil {
		panic(fmt.Sprintf("error reading config file %v - %v\n", file, err))
	}
	return nodeConfig
}

const (
	DeploymentDevelopment = 0
	DeploymentTestNet     = 1
	DeploymentMainNet     = 2
)

/*Config - all the config options passed from the command line*/
type Config struct {
	Host           string
	Port           int
	ChainID        string
	DeploymentMode byte
}

/*Configuration of the system */
var Configuration Config

/*TestNet - is the server running in TestNet mode? */
func TestNet() bool {
	return Configuration.DeploymentMode == DeploymentTestNet
}

/*Development - is the server running in development mode? */
func Development() bool {
	return Configuration.DeploymentMode == DeploymentDevelopment
}

/*MainNet - is the server running in mainnet mode? */
func MainNet() bool {
	return Configuration.DeploymentMode == DeploymentMainNet
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
