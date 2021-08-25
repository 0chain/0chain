package chain

import (
	"time"

	"0chain.net/smartcontract/minersc"

	"0chain.net/core/viper"

	"0chain.net/core/datastore"
)

const (
	//BlockProposalWaitStatic Static wait time for block proposals
	BlockProposalWaitStatic = 0
	//BlockProposalWaitDynamic Dyanamic wait time for block proposals
	BlockProposalWaitDynamic = iota
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

// HealthCheckCycleScan -
type HealthCheckCycleScan struct {
	Settle     time.Duration `json:"settle"`
	SettleSecs int           `json:"settle_period_secs"`

	Enabled   bool  `json:"scan_enable"`
	BatchSize int64 `json:"batch_size"`

	Window int64 `json:"scan_window"`

	RepeatInterval     time.Duration `json:"repeat_interval"`
	RepeatIntervalMins int           `json:"repeat_interval_mins"`

	ReportStatusMins int           `json:"report_status_mins"`
	ReportStatus     time.Duration `json:"report_status"`
}

//Config - chain Configuration
type Config struct {
	OwnerID              datastore.Key `json:"owner_id"`                // Client who created this chain
	BlockSize            int32         `json:"block_size"`              // Number of transactions in a block
	MinBlockSize         int32         `json:"min_block_size"`          // Number of transactions a block needs to have
	MaxByteSize          int64         `json:"max_byte_size"`           // Max number of bytes a block can have
	MinGenerators        int           `json:"min_generators"`          // Min number of block generators.
	GeneratorsPercent    float64       `json:"generators_percent"`      // Percentage of all miners
	NumReplicators       int           `json:"num_replicators"`         // Number of sharders that can store the block
	ThresholdByCount     int           `json:"threshold_by_count"`      // Threshold count for a block to be notarized
	ThresholdByStake     int           `json:"threshold_by_stake"`      // Stake threshold for a block to be notarized
	ValidationBatchSize  int           `json:"validation_size"`         // Batch size of txns for crypto verification
	TxnMaxPayload        int           `json:"transaction_max_payload"` // Max payload allowed in the transaction
	PruneStateBelowCount int           `json:"prune_state_below_count"` // Prune state below these many rounds
	RoundRange           int64         `json:"round_range"`             // blocks are stored in separate directory for each range of rounds
	// todo move BlocksToSharder out of Config
	BlocksToSharder       int `json:"blocks_to_sharder"`       // send finalized or notarized blocks to sharder
	VerificationTicketsTo int `json:"verification_tickets_to"` // send verification tickets to generator or all miners

	HealthShowCounters bool `json:"health_show_counters"` // display detail counters
	// Health Check switches
	HCCycleScan [2]HealthCheckCycleScan

	BlockProposalMaxWaitTime time.Duration `json:"block_proposal_max_wait_time"` // max time to wait to receive a block proposal
	BlockProposalWaitMode    int8          `json:"block_proposal_wait_mode"`     // wait time for the block proposal is static (0) or dynamic (1)

	ReuseTransactions bool `json:"reuse_txns"` // indicates if transactions from unrelated blocks can be reused

	ClientSignatureScheme string `json:"client_signature_scheme"` // indicates which signature scheme is being used

	MinActiveSharders    int `json:"min_active_sharders"`    // Minimum active sharders required to validate blocks
	MinActiveReplicators int `json:"min_active_replicators"` // Minimum active replicators of a block that should be active to verify the block

	SmartContractTimeout   time.Duration `json:"smart_contract_timeout"` // time after which the smart contract execution will timeout
	RoundTimeoutSofttoMin  int           `json:"softto_min"`             // minimum time for softtimeout to kick in milliseconds
	RoundTimeoutSofttoMult int           `json:"softto_mult"`            // multiplier of mean network time for soft timeout
	RoundRestartMult       int           `json:"round_restart_mult"`     // multiplier of soft timeouts to restart a round
}

func (conf *Config) Update(cf *minersc.GlobalSettings) {
	// conf.OwnerID = cf.GetString(minersc.Owner)
	conf.MinBlockSize = cf.GetInt32(minersc.BlockMinSize)
	conf.BlockSize = cf.GetInt32(minersc.BlockMaxSize)
	conf.MaxByteSize = cf.GetInt64(minersc.BlockMaxByteSize)
	conf.NumReplicators = cf.GetInt(minersc.BlockReplicators)
	conf.BlockProposalMaxWaitTime = cf.GetDuration(minersc.BlockProposalMaxWaitTime)
	waitMode := cf.GetString(minersc.BlockProposalWaitMode)
	if waitMode == "static" {
		conf.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		conf.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	conf.ThresholdByCount = cf.GetInt(minersc.BlockConsensusThresholdByCount)
	conf.ThresholdByStake = cf.GetInt(minersc.BlockConsensusThresholdByStake)
	conf.MinActiveSharders = cf.GetInt(minersc.BlockShardingMinActiveSharders)
	conf.MinActiveReplicators = cf.GetInt(minersc.BlockShardingMinActiveReplicators)
	conf.ValidationBatchSize = cf.GetInt(minersc.BlockValidationBatchSize)
	conf.ReuseTransactions = cf.GetBool(minersc.BlockReuseTransactions)
	conf.MinGenerators = cf.GetInt(minersc.BlockMinGenerators)
	conf.GeneratorsPercent = cf.GetFloat64(minersc.BlockGeneratorsPercent)
	conf.RoundRange = cf.GetInt64(minersc.RoundRange)
	conf.RoundTimeoutSofttoMin = cf.GetInt(minersc.RoundTimeoutsSofttoMin)
	conf.RoundTimeoutSofttoMult = cf.GetInt(minersc.RoundTimeoutsSofttoMult)
	conf.RoundRestartMult = cf.GetInt(minersc.RoundTimeoutsRoundRestartMult)
	conf.TxnMaxPayload = cf.GetInt(minersc.TransactionPayloadMaxSize)
	conf.ClientSignatureScheme = cf.GetString(minersc.ClientSignatureScheme)
	verificationTicketsTo := cf.GetString(minersc.MessagesVerificationTicketsTo)
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		conf.VerificationTicketsTo = AllMiners
	} else {
		conf.VerificationTicketsTo = Generator
	}
	conf.PruneStateBelowCount = cf.GetInt(minersc.StatePruneBelowCount)
	conf.SmartContractTimeout = cf.GetDuration(minersc.SmartContractTimeout)
}

func (config *Config) UpdateHealthCheck(configMap *minersc.GlobalSettings) {
	// Health Check related counters
	// Work on deep scan
	conf := &config.HCCycleScan[DeepScan]

	conf.Enabled = viper.GetBool("server_chain.health_check.deep_scan.enabled")
	conf.BatchSize = viper.GetInt64("server_chain.health_check.deep_scan.batch_size")
	conf.Window = viper.GetInt64("server_chain.health_check.deep_scan.window")

	conf.SettleSecs = viper.GetInt("server_chain.health_check.deep_scan.settle_secs")
	conf.Settle = time.Duration(conf.SettleSecs) * time.Second

	conf.RepeatIntervalMins = viper.GetInt("server_chain.health_check.deep_scan.repeat_interval_mins")
	conf.RepeatInterval = time.Duration(conf.RepeatIntervalMins) * time.Minute

	conf.ReportStatusMins = viper.GetInt("server_chain.health_check.deep_scan.report_status_mins")
	conf.ReportStatus = time.Duration(conf.ReportStatusMins) * time.Minute

	// Work on proximity scan
	conf = &config.HCCycleScan[ProximityScan]

	conf.Enabled = viper.GetBool("server_chain.health_check.proximity_scan.enabled")
	conf.BatchSize = viper.GetInt64("server_chain.health_check.proximity_scan.batch_size")
	conf.Window = viper.GetInt64("server_chain.health_check.proximity_scan.window")

	conf.SettleSecs = viper.GetInt("server_chain.health_check.proximity_scan.settle_secs")
	conf.Settle = time.Duration(conf.SettleSecs) * time.Second

	conf.RepeatIntervalMins = viper.GetInt("server_chain.health_check.proximity_scan.repeat_interval_mins")
	conf.RepeatInterval = time.Duration(conf.RepeatIntervalMins) * time.Minute

	conf.ReportStatusMins = viper.GetInt("server_chain.health_check.proximity_scan.report_status_mins")
	conf.ReportStatus = time.Duration(conf.ReportStatusMins) * time.Minute

	config.HealthShowCounters = viper.GetBool("server_chain.health_check.show_counters")
	config.SmartContractTimeout = viper.GetDuration("server_chain.smart_contract.timeout")
	if config.SmartContractTimeout == 0 {
		config.SmartContractTimeout = DefaultSmartContractTimeout
	}
}
