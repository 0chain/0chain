package chain

import (
	"time"

	"0chain.net/smartcontract/minersc"

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

	SmartContractTimeout             time.Duration `json:"smart_contract_timeout"`               // time after which the smart contract execution will timeout
	SmartContractSettingUpdatePeriod int64         `json:"smart_contract_setting_update_period"` // rounds settings are updated

	RoundTimeoutSofttoMin  int `json:"softto_min"`         // minimum time for softtimeout to kick in milliseconds
	RoundTimeoutSofttoMult int `json:"softto_mult"`        // multiplier of mean network time for soft timeout
	RoundRestartMult       int `json:"round_restart_mult"` // multiplier of soft timeouts to restart a round
}

func (conf *Config) Update(cf *minersc.GlobalSettings) error {
	var err error
	conf.MinBlockSize, err = cf.GetInt32(minersc.BlockMinSize)
	if err != nil {
		return err
	}
	conf.BlockSize, err = cf.GetInt32(minersc.BlockMaxSize)
	if err != nil {
		return err
	}
	conf.MaxByteSize, err = cf.GetInt64(minersc.BlockMaxByteSize)
	if err != nil {
		return err
	}
	conf.NumReplicators, err = cf.GetInt(minersc.BlockReplicators)
	if err != nil {
		return err
	}
	conf.BlockProposalMaxWaitTime, err = cf.GetDuration(minersc.BlockProposalMaxWaitTime)
	if err != nil {
		return err
	}
	waitMode, err := cf.GetString(minersc.BlockProposalWaitMode)
	if err != nil {
		return err
	}
	if waitMode == "static" {
		conf.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		conf.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	conf.ThresholdByCount, err = cf.GetInt(minersc.BlockConsensusThresholdByCount)
	if err != nil {
		return err
	}
	conf.ThresholdByStake, err = cf.GetInt(minersc.BlockConsensusThresholdByStake)
	if err != nil {
		return err
	}
	conf.MinActiveSharders, err = cf.GetInt(minersc.BlockShardingMinActiveSharders)
	if err != nil {
		return err
	}
	conf.MinActiveReplicators, err = cf.GetInt(minersc.BlockShardingMinActiveReplicators)
	if err != nil {
		return err
	}
	conf.ValidationBatchSize, err = cf.GetInt(minersc.BlockValidationBatchSize)
	if err != nil {
		return err
	}
	conf.ReuseTransactions, err = cf.GetBool(minersc.BlockReuseTransactions)
	if err != nil {
		return err
	}
	conf.MinGenerators, err = cf.GetInt(minersc.BlockMinGenerators)
	if err != nil {
		return err
	}
	conf.GeneratorsPercent, err = cf.GetFloat64(minersc.BlockGeneratorsPercent)
	if err != nil {
		return err
	}
	conf.RoundRange, err = cf.GetInt64(minersc.RoundRange)
	if err != nil {
		return err
	}
	conf.RoundTimeoutSofttoMin, err = cf.GetInt(minersc.RoundTimeoutsSofttoMin)
	if err != nil {
		return err
	}
	conf.RoundTimeoutSofttoMult, err = cf.GetInt(minersc.RoundTimeoutsSofttoMult)
	if err != nil {
		return err
	}
	conf.RoundRestartMult, err = cf.GetInt(minersc.RoundTimeoutsRoundRestartMult)
	if err != nil {
		return err
	}
	conf.TxnMaxPayload, err = cf.GetInt(minersc.TransactionPayloadMaxSize)
	if err != nil {
		return err
	}
	conf.ClientSignatureScheme, err = cf.GetString(minersc.ClientSignatureScheme)
	if err != nil {
		return err
	}
	verificationTicketsTo, err := cf.GetString(minersc.MessagesVerificationTicketsTo)
	if err != nil {
		return err
	}
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		conf.VerificationTicketsTo = AllMiners
	} else {
		conf.VerificationTicketsTo = Generator
	}
	conf.PruneStateBelowCount, err = cf.GetInt(minersc.StatePruneBelowCount)
	if err != nil {
		return err
	}
	conf.SmartContractTimeout, err = cf.GetDuration(minersc.SmartContractTimeout)
	if err != nil {
		return err
	}
	if conf.SmartContractTimeout == 0 {
		conf.SmartContractTimeout = DefaultSmartContractTimeout
	}
	conf.SmartContractSettingUpdatePeriod = cf.GetInt64(minersc.SmartContractSettingUpdatePeriod)
	return nil
}

// We don't need this yet, as the health check settings are used to set up a worker thread.
func (conf *Config) UpdateHealthCheckSettings(cf *minersc.GlobalSettings) error {
	var err error
	conf.HealthShowCounters, err = cf.GetBool(minersc.HealthCheckShowCounters)
	if err != nil {
		return err
	}
	ds := &conf.HCCycleScan[DeepScan]
	ds.Enabled, err = cf.GetBool(minersc.HealthCheckDeepScanEnabled)
	if err != nil {
		return err
	}
	ds.BatchSize, err = cf.GetInt64(minersc.HealthCheckDeepScanBatchSize)
	if err != nil {
		return err
	}
	ds.Window, err = cf.GetInt64(minersc.HealthCheckDeepScanWindow)
	if err != nil {
		return err
	}
	ds.Settle, err = cf.GetDuration(minersc.HealthCheckDeepScanSettleSecs)
	if err != nil {
		return err
	}
	ds.RepeatInterval, err = cf.GetDuration(minersc.HealthCheckDeepScanIntervalMins)
	if err != nil {
		return err
	}
	ds.ReportStatus, err = cf.GetDuration(minersc.HealthCheckDeepScanReportStatusMins)
	if err != nil {
		return err
	}

	ps := &conf.HCCycleScan[ProximityScan]
	ps.Enabled, err = cf.GetBool(minersc.HealthCheckProximityScanEnabled)
	if err != nil {
		return err
	}
	ps.BatchSize, err = cf.GetInt64(minersc.HealthCheckProximityScanBatchSize)
	if err != nil {
		return err
	}
	ps.Window, err = cf.GetInt64(minersc.HealthCheckProximityScanWindow)
	if err != nil {
		return err
	}
	ps.Settle, err = cf.GetDuration(minersc.HealthCheckProximityScanSettleSecs)
	if err != nil {
		return err
	}
	ps.RepeatInterval, err = cf.GetDuration(minersc.HealthCheckProximityScanRepeatIntervalMins)
	if err != nil {
		return err
	}
	ps.ReportStatus, err = cf.GetDuration(minersc.HealthCheckProximityScanRejportStatusMins)
	if err != nil {
		return err
	}
	return nil
}
