package chain

import (
	"time"

	"0chain.net/core/datastore"
)

const (
	//BlockProposalWaitStatic Static wait time for block proposals
	BlockProposalWaitStatic = 0
	//BlockProposalWaitDynamic Dyanamic wait time for block proposals
	BlockProposalWaitDynamic = iota
)

type HealthCheckScan int
const (
	DeepScan HealthCheckScan = iota
	ProximityScan
)

type HealthCheckCycleScan struct {
	Enabled bool `json:"scan_enable"`
	BatchSize int `json:"batch_size"`

	Window int64 `json:"scan_window"`

	Interval time.Duration `json:"scan_interval"`
	IntervalMins int `json:"scan_interval_mins"`

	ReportStatusMins int `json:report_status_mins`
	ReportStatus time.Duration `json:report_status`
}

//Config - chain Configuration
type Config struct {
	OwnerID                  datastore.Key `json:"owner_id"`                  // Client who created this chain
	ParentChainID            datastore.Key `json:"parent_chain_id,omitempty"` // Chain from which this chain is forked off
	GenesisBlockHash         string        `json:"genesis_block_hash"`
	Decimals                 int8          `json:"decimals"`                     // Number of decimals allowed for the token on this chain
	BlockSize                int32         `json:"block_size"`                   // Number of transactions in a block
	MinBlockSize             int32         `json:"min_block_size"`               // Number of transactions a block needs to have
	MaxByteSize              int64         `json:"max_byte_size"`                // Max number of bytes a block can have
	NumGenerators            int           `json:"num_generators"`               // Number of block generators
	NumReplicators           int           `json:"num_replicators"`              // Number of sharders that can store the block
	ThresholdByCount         int           `json:"threshold_by_count"`           // Threshold count for a block to be notarized
	ThresholdByStake         int           `json:"threshold_by_stake"`           // Stake threshold for a block to be notarized
	ValidationBatchSize      int           `json:"validation_size"`              // Batch size of txns for crypto verification
	TxnMaxPayload            int           `json:"transaction_max_payload"`      // Max payload allowed in the transaction
	PruneStateBelowCount     int           `json:"prune_state_below_count"`      // Prune state below these many rounds
	RoundRange               int64         `json:"round_range"`                  // blocks are stored in separate directory for each range of rounds
	BlocksToSharder          int           `json:"blocks_to_sharder"`            // send finalized or notarized blocks to sharder
	VerificationTicketsTo    int           `json:"verification_tickets_to"`      // send verification tickets to generator or all miners

	// Health Check switches
	HC_CycleScan               [2]HealthCheckCycleScan

	//HC_ScanWindow              [2]int64 `json:"health_check_scan_window"`  // indicates the cycle window
	//HC_ScanIntervalMinutes     [2]int
	//HC_ScanInterval            [2]time.Duration
	//
	//// HC_ProximityScanWindow     int64 `json:"health_partial_scan_window"` // indicates partial scan
	//
	//BatchSyncSize            int   `json:"batch_sync_size"`           // gives the batch size for syncing
	//
	//HC_CycleRepeatMinutes  int  `json:"health_check_cycle_repeat"` // Repeat entire health check in minutes
	//HC_CycleRepeat 		   time.Duration  `json:"-"` // Repeat entire health check in time.Duration
	//
	//HC_CycleHiatusMinutes  int  `json:"health_check_cycle_hiatus"` // gives healthcheck hiatus in minutes
	//HC_CycleHiatus         time.Duration  `json:"-"` // gives healthcheck hiatus in time.Duration

	HealthShowCounters     bool `json:"health_show_counters"`      // display detail counters

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
