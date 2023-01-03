package event

import (
	"sync"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/currency"
)

type TestConfig struct {
	conf  *TestConfigData
	guard sync.RWMutex
}

type TestConfigData struct {
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
	MinTxnFee             currency.Coin `json:"min_txn_fee"`               // Minimum txn fee allowed
	PruneStateBelowCount  int           `json:"prune_state_below_count"`   // Prune state below these many rounds
	RoundRange            int64         `json:"round_range"`               // blocks are stored in separate directory for each range of rounds

	// todo move BlocksToSharder out of ConfigData
	BlocksToSharder       int `json:"blocks_to_sharder"`       // send finalized or notarized blocks to sharder
	VerificationTicketsTo int `json:"verification_tickets_to"` // send verification tickets to generator or all miners

	HealthShowCounters bool `json:"health_show_counters"` // display detail counters
	// Health Check switches
	HCCycleScan [2]config.HealthCheckCycleScan

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

	DbsEvents   config.DbAccess   `json:"dbs_event"`
	DbsSettings config.DbSettings `json:"dbs_settings"`
	TxnExempt   map[string]bool   `json:"txn_exempt"`
}

func (t *TestConfig) IsStateEnabled() bool {
	return t.conf.IsStateEnabled
}

func (t *TestConfig) IsDkgEnabled() bool {
	return t.conf.IsDkgEnabled
}

func (t *TestConfig) IsViewChangeEnabled() bool {
	return t.conf.IsViewChangeEnabled
}

func (t *TestConfig) IsBlockRewardsEnabled() bool {
	return t.conf.IsBlockRewardsEnabled
}

func (t *TestConfig) IsStorageEnabled() bool {
	return t.conf.IsStorageEnabled
}

func (t *TestConfig) IsFaucetEnabled() bool {
	return t.conf.IsFaucetEnabled
}

func (t *TestConfig) IsInterestEnabled() bool {
	return t.conf.IsInterestEnabled
}

func (t *TestConfig) IsFeeEnabled() bool {
	return t.conf.IsFeeEnabled
}

func (t *TestConfig) IsMultisigEnabled() bool {
	return t.conf.IsMultisigEnabled
}

func (t *TestConfig) IsVestingEnabled() bool {
	return t.conf.IsVestingEnabled
}

func (t *TestConfig) IsZcnEnabled() bool {
	return t.conf.IsZcnEnabled
}

func (t *TestConfig) OwnerID() datastore.Key {
	return t.conf.OwnerID
}

func (t *TestConfig) MinBlockSize() int32 {
	return t.conf.MinBlockSize
}

func (t *TestConfig) MaxBlockCost() int {
	return t.conf.MaxBlockCost
}

func (t *TestConfig) MaxByteSize() int64 {
	return t.conf.MaxByteSize
}

func (t *TestConfig) MinGenerators() int {
	return t.conf.MinGenerators
}

func (t *TestConfig) GeneratorsPercent() float64 {
	return t.conf.GeneratorsPercent
}

func (t *TestConfig) NumReplicators() int {
	return t.conf.NumReplicators
}

func (t *TestConfig) ThresholdByCount() int {
	return t.conf.ThresholdByCount
}

func (t *TestConfig) ThresholdByStake() int {
	return t.conf.ThresholdByStake
}

func (t *TestConfig) ValidationBatchSize() int {
	return t.conf.ValidationBatchSize
}

func (t *TestConfig) TxnMaxPayload() int {
	return t.conf.TxnMaxPayload
}

func (t *TestConfig) PruneStateBelowCount() int {
	return t.conf.PruneStateBelowCount
}

func (t *TestConfig) RoundRange() int64 {
	return t.conf.RoundRange
}

func (t *TestConfig) BlocksToSharder() int {
	return t.conf.BlocksToSharder
}

func (t *TestConfig) VerificationTicketsTo() int {
	return t.conf.VerificationTicketsTo
}

func (t *TestConfig) HealthShowCounters() bool {
	return t.conf.HealthShowCounters
}

func (t *TestConfig) HCCycleScan() [2]config.HealthCheckCycleScan {
	return t.conf.HCCycleScan
}

func (t *TestConfig) BlockProposalMaxWaitTime() time.Duration {
	return t.conf.BlockProposalMaxWaitTime
}

func (t *TestConfig) BlockProposalWaitMode() int8 {
	return t.conf.BlockProposalWaitMode
}

func (t *TestConfig) ReuseTransactions() bool {
	return t.conf.ReuseTransactions
}

func (t *TestConfig) ClientSignatureScheme() string {
	return t.conf.ClientSignatureScheme
}

func (t *TestConfig) MinActiveSharders() int {
	return t.conf.MinActiveSharders
}

func (t *TestConfig) MinActiveReplicators() int {
	return t.conf.MinActiveReplicators
}

func (t *TestConfig) SmartContractTimeout() time.Duration {
	return t.conf.SmartContractTimeout
}

func (t *TestConfig) SmartContractSettingUpdatePeriod() int64 {
	return t.conf.SmartContractSettingUpdatePeriod
}

func (t *TestConfig) RoundTimeoutSofttoMin() int {
	return t.conf.RoundTimeoutSofttoMin
}

func (t *TestConfig) RoundTimeoutSofttoMult() int {
	return t.conf.RoundTimeoutSofttoMult
}

func (t *TestConfig) RoundRestartMult() int {
	return t.conf.RoundRestartMult
}

func (t *TestConfig) DbsEvents() config.DbAccess {
	return t.conf.DbsEvents
}

func (t *TestConfig) DbSettings() config.DbSettings {
	return t.conf.DbsSettings
}

func (t *TestConfig) FromViper() error {
	//TODO implement me
	panic("implement me")
}

func (t *TestConfig) Update(configMap map[string]string, version int64) error {
	//TODO implement me
	panic("implement me")
}

func (t *TestConfig) TxnExempt() map[string]bool {
	return t.conf.TxnExempt
}

func (t *TestConfig) MinTxnFee() currency.Coin {
	return t.conf.MinTxnFee
}

func (t *TestConfig) TxnTransferCost() int {
	return t.conf.TxnTransferCost
}
