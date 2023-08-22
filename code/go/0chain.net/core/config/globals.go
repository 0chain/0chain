package config

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
	Vesting                           // todo from development
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
	BlockFinalizationTimeout
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
	TransactionMaxFee
	TransactionExempt
	TransactionCostFeeCoeff
	TransactionFutureNonce

	ClientSignatureScheme
	ClientDiscover // todo from chain

	MessagesVerificationTicketsTo

	StatePruneBelowCount
	StateSyncTimeout // todo from chain

	StuckCheckInterval // todo from chain
	StuckTimeThreshold // todo from chain

	SmartContractTimeout
	SmartContractSettingUpdatePeriod

	LfbTicketRebroadcastTimeout // todo restart worker
	LfbTicketAhead              // todo from chain

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

	DbsAggregateDebug
	DbsAggregatePeriod
	DbsPartitionChangePeriod
	DbsPartitionKeepCount
	DbsAggregatePageLimit

	HealthCheckDeepScanEnabled          // todo restart worker
	HealthCheckDeepScanBatchSize        // todo restart worker
	HealthCheckDeepScanWindow           // todo restart worker
	HealthCheckDeepScanSettleSecs       // todo restart worker
	HealthCheckDeepScanIntervalMins     // todo restart worker
	HealthCheckDeepScanReportStatusMins // todo restart worker

	HealthCheckProximityScanEnabled            // todo restart worker
	HealthCheckProximityScanBatchSize          // todo restart worker
	HealthCheckProximityScanWindow             // todo restart worker
	HealthCheckProximityScanSettleSecs         // todo restart worker
	HealthCheckProximityScanRepeatIntervalMins // todo restart worker
	HealthCheckProximityScanRejportStatusMins  // todo restart worker
	HealthCheckShowCounters                    // todo restart worke

	NumOfGlobalSettings
)

var (
	GlobalSettingName []string
	GlobalSettingInfo map[string]struct {
		SettingType ConfigType

		// Indicates that the settings cannot be changed by a transaction
		// This includes both true immutable settings and settings that are local
		// and are changed by restarting editing 0chain.yaml and restarting the module
		// todo we need to split up immutable and stored in MPT and local so can't be changed in transaction
		Mutable bool
	}
	GlobalSettingsIgnored map[string]bool
)

func (gs GlobalSetting) String() string {
	if gs < 0 || int(gs) >= len(GlobalSettingName) {
		return "unknown"
	}

	return GlobalSettingName[gs]
}

func (gs GlobalSetting) Int() int {
	return int(gs)
}

func init() {
	initGlobalSettingNames()
	initGlobalSettings()
	initGlobalSettingsIgnored()
}

func initGlobalSettingNames() {
	GlobalSettingName = make([]string, NumOfGlobalSettings+1)
	GlobalSettingName[State] = "server_chain.state.enabled"
	GlobalSettingName[Dkg] = "server_chain.dkg"
	GlobalSettingName[ViewChange] = "server_chain.view_change"
	GlobalSettingName[BlockRewards] = "server_chain.block_rewards"
	GlobalSettingName[Storage] = "server_chain.smart_contract.storage"
	GlobalSettingName[Faucet] = "server_chain.smart_contract.faucet"
	GlobalSettingName[Miner] = "server_chain.smart_contract.miner"
	GlobalSettingName[Multisig] = "server_chain.smart_contract.multisig"
	GlobalSettingName[Vesting] = "server_chain.smart_contract.vesting"
	GlobalSettingName[Zcn] = "server_chain.smart_contract.zcn"

	GlobalSettingName[Owner] = "server_chain.owner"

	GlobalSettingName[BlockMinSize] = "server_chain.block.min_block_size"
	GlobalSettingName[BlockMaxSize] = "server_chain.block.max_block_size"
	GlobalSettingName[BlockMaxCost] = "server_chain.block.max_block_cost"
	GlobalSettingName[BlockMaxByteSize] = "server_chain.block.max_byte_size"
	GlobalSettingName[BlockReplicators] = "server_chain.block.replicators"
	GlobalSettingName[BlockGenerationTimout] = "server_chain.block.generation.timeout"

	GlobalSettingName[BlockGenerationRetryWaitTime] = "server_chain.block.generation.retry_wait_time"
	GlobalSettingName[BlockProposalMaxWaitTime] = "server_chain.block.proposal.max_wait_time"
	GlobalSettingName[BlockProposalWaitMode] = "server_chain.block.proposal.wait_mode"
	GlobalSettingName[BlockConsensusThresholdByCount] = "server_chain.block.consensus.threshold_by_count"
	GlobalSettingName[BlockConsensusThresholdByStake] = "server_chain.block.consensus.threshold_by_stake"
	GlobalSettingName[BlockShardingMinActiveSharders] = "server_chain.block.sharding.min_active_sharders"
	GlobalSettingName[BlockShardingMinActiveReplicators] = "server_chain.block.sharding.min_active_replicators"
	GlobalSettingName[BlockValidationBatchSize] = "server_chain.block.validation.batch_size"
	GlobalSettingName[BlockReuseTransactions] = "server_chain.block.reuse_txns"
	GlobalSettingName[BlockFinalizationTimeout] = "server_chain.block.finalization.timeout"
	GlobalSettingName[BlockMinGenerators] = "server_chain.block.min_generators"
	GlobalSettingName[BlockGeneratorsPercent] = "server_chain.block.generators_percent"

	GlobalSettingName[RoundRange] = "server_chain.round_range"

	GlobalSettingName[RoundTimeoutsSofttoMin] = "server_chain.round_timeouts.softto_min"
	GlobalSettingName[RoundTimeoutsSofttoMult] = "server_chain.round_timeouts.softto_mult"
	GlobalSettingName[RoundTimeoutsRoundRestartMult] = "server_chain.round_timeouts.round_restart_mult"
	GlobalSettingName[RoundTimeoutsTimeoutCap] = "server_chain.round_timeouts.timeout_cap"

	GlobalSettingName[TransactionPayloadMaxSize] = "server_chain.transaction.payload.max_size"
	GlobalSettingName[TransactionTimeout] = "server_chain.transaction.timeout"
	GlobalSettingName[TransactionMinFee] = "server_chain.transaction.min_fee"
	GlobalSettingName[TransactionMaxFee] = "server_chain.transaction.max_fee"
	GlobalSettingName[TransactionExempt] = "server_chain.transaction.exempt"
	GlobalSettingName[TransactionCostFeeCoeff] = "server_chain.transaction.cost_fee_coeff"
	GlobalSettingName[TransactionFutureNonce] = "server_chain.transaction.future_nonce"

	GlobalSettingName[ClientSignatureScheme] = "server_chain.client.signature_scheme"
	GlobalSettingName[ClientDiscover] = "server_chain.client.discover"

	GlobalSettingName[MessagesVerificationTicketsTo] = "server_chain.messages.verification_tickets_to"

	GlobalSettingName[StatePruneBelowCount] = "server_chain.state.prune_below_count"
	GlobalSettingName[StateSyncTimeout] = "server_chain.state.sync.timeout"

	GlobalSettingName[StuckCheckInterval] = "server_chain.stuck.check_interval"
	GlobalSettingName[StuckTimeThreshold] = "server_chain.stuck.time_threshold"

	GlobalSettingName[SmartContractTimeout] = "server_chain.smart_contract.timeout"
	GlobalSettingName[SmartContractSettingUpdatePeriod] = "server_chain.smart_contract.setting_update_period"

	GlobalSettingName[LfbTicketRebroadcastTimeout] = "server_chain.lfb_ticket.rebroadcast_timeout"
	GlobalSettingName[LfbTicketAhead] = "server_chain.lfb_ticket.ahead"

	GlobalSettingName[AsyncFetchingMaxSimultaneousFromMiners] = "server_chain.async_blocks_fetching.max_simultaneous_from_miners"
	GlobalSettingName[AsyncFetchingMaxSimultaneousFromSharders] = "server_chain.async_blocks_fetching.max_simultaneous_from_sharders"

	GlobalSettingName[DbsEventsEnabled] = "server_chain.dbs.events.enabled"
	GlobalSettingName[DbsEventsName] = "server_chain.dbs.events.name"
	GlobalSettingName[DbsEventsUser] = "server_chain.dbs.events.user"
	GlobalSettingName[DbsEventsPassword] = "server_chain.dbs.events.password"
	GlobalSettingName[DbsEventsHost] = "server_chain.dbs.events.host"
	GlobalSettingName[DbsEventsPort] = "server_chain.dbs.events.port"
	GlobalSettingName[DbsEventsMaxIdleConns] = "server_chain.dbs.events.max_idle_conns"
	GlobalSettingName[DbsEventsMaxOpenConns] = "server_chain.dbs.events.max_open_conns"
	GlobalSettingName[DbsEventsConnMaxLifetime] = "server_chain.dbs.events.conn_max_lifetime"

	GlobalSettingName[DbsAggregateDebug] = "server_chain.dbs.settings.debug"
	GlobalSettingName[DbsAggregatePeriod] = "server_chain.dbs.settings.aggregate_period"
	GlobalSettingName[DbsPartitionChangePeriod] = "server_chain.dbs.settings.partition_change_period"
	GlobalSettingName[DbsPartitionKeepCount] = "server_chain.dbs.settings.partition_keep_count"
	GlobalSettingName[DbsAggregatePageLimit] = "server_chain.dbs.settings.page_limit" +
		""
	GlobalSettingName[HealthCheckDeepScanEnabled] = "server_chain.health_check.deep_scan.enabled"
	GlobalSettingName[HealthCheckDeepScanBatchSize] = "server_chain.health_check.deep_scan.batch_size"
	GlobalSettingName[HealthCheckDeepScanWindow] = "server_chain.health_check.deep_scan.window"
	GlobalSettingName[HealthCheckDeepScanSettleSecs] = "server_chain.health_check.deep_scan.settle_secs"
	GlobalSettingName[HealthCheckDeepScanIntervalMins] = "server_chain.health_check.deep_scan.repeat_interval_mins"
	GlobalSettingName[HealthCheckDeepScanReportStatusMins] = "server_chain.health_check.deep_scan.report_status_mins"

	GlobalSettingName[HealthCheckProximityScanEnabled] = "server_chain.health_check.proximity_scan.enabled"
	GlobalSettingName[HealthCheckProximityScanBatchSize] = "server_chain.health_check.proximity_scan.batch_size"
	GlobalSettingName[HealthCheckProximityScanWindow] = "server_chain.health_check.proximity_scan.window"
	GlobalSettingName[HealthCheckProximityScanSettleSecs] = "server_chain.health_check.proximity_scan.settle_secs"
	GlobalSettingName[HealthCheckProximityScanRepeatIntervalMins] = "server_chain.health_check.proximity_scan.repeat_interval_mins"
	GlobalSettingName[HealthCheckProximityScanRejportStatusMins] = "server_chain.health_check.proximity_scan.report_status_mins"

	GlobalSettingName[HealthCheckShowCounters] = "server_chain.health_check.show_counters"

	GlobalSettingName[NumOfGlobalSettings] = "invalid"
}

func initGlobalSettingsIgnored() {
	GlobalSettingsIgnored = map[string]bool{
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
}

func initGlobalSettings() {
	// GlobalSettingInfo Indicates the type of each global settings, and whether it is possible to change each setting
	GlobalSettingInfo = map[string]struct {
		SettingType ConfigType

		// Indicates that the settings cannot be changed by a transaction
		// This includes both true immutable settings and settings that are local
		// and are changed by restarting editing 0chain.yaml and restarting the module
		// todo we need to split up immutable and stored in MPT and local so can't be changed in transaction
		Mutable bool
	}{
		GlobalSettingName[State]:        {Boolean, false},
		GlobalSettingName[Dkg]:          {Boolean, false},
		GlobalSettingName[ViewChange]:   {Boolean, false},
		GlobalSettingName[BlockRewards]: {Boolean, false},
		GlobalSettingName[Storage]:      {Boolean, false},
		GlobalSettingName[Faucet]:       {Boolean, false},
		GlobalSettingName[Miner]:        {Boolean, false},
		GlobalSettingName[Multisig]:     {Boolean, true},
		GlobalSettingName[Vesting]:      {Boolean, true},
		GlobalSettingName[Zcn]:          {Boolean, false},

		GlobalSettingName[Owner]: {String, false},

		GlobalSettingName[BlockMinSize]:                      {Int32, true},
		GlobalSettingName[BlockMaxSize]:                      {Int32, true},
		GlobalSettingName[BlockMaxCost]:                      {Int, true},
		GlobalSettingName[BlockMaxByteSize]:                  {Int64, true},
		GlobalSettingName[BlockReplicators]:                  {Int, true},
		GlobalSettingName[BlockGenerationTimout]:             {Int, true},
		GlobalSettingName[BlockGenerationRetryWaitTime]:      {Int, true},
		GlobalSettingName[BlockProposalMaxWaitTime]:          {Duration, true},
		GlobalSettingName[BlockProposalWaitMode]:             {String, true},
		GlobalSettingName[BlockConsensusThresholdByCount]:    {Int, true},
		GlobalSettingName[BlockConsensusThresholdByStake]:    {Int, true},
		GlobalSettingName[BlockShardingMinActiveSharders]:    {Int, true},
		GlobalSettingName[BlockShardingMinActiveReplicators]: {Int, true},
		GlobalSettingName[BlockValidationBatchSize]:          {Int, true},
		GlobalSettingName[BlockReuseTransactions]:            {Boolean, true},
		GlobalSettingName[BlockFinalizationTimeout]:          {Duration, true},
		GlobalSettingName[BlockMinGenerators]:                {Int, true},
		GlobalSettingName[BlockGeneratorsPercent]:            {Float64, true},

		GlobalSettingName[RoundRange]:                    {Int64, true},
		GlobalSettingName[RoundTimeoutsSofttoMin]:        {Int, true},
		GlobalSettingName[RoundTimeoutsSofttoMult]:       {Int, true},
		GlobalSettingName[RoundTimeoutsRoundRestartMult]: {Int, true},
		GlobalSettingName[RoundTimeoutsTimeoutCap]:       {Int, true},

		GlobalSettingName[TransactionPayloadMaxSize]: {Int, true},
		GlobalSettingName[TransactionTimeout]:        {Int, false},
		GlobalSettingName[TransactionMinFee]:         {Int64, true},
		GlobalSettingName[TransactionMaxFee]:         {Int64, false},
		GlobalSettingName[TransactionExempt]:         {Strings, true},
		GlobalSettingName[TransactionCostFeeCoeff]:   {Int, true},
		GlobalSettingName[TransactionFutureNonce]:    {Int, true},

		GlobalSettingName[ClientSignatureScheme]: {String, true},
		GlobalSettingName[ClientDiscover]:        {Boolean, false},

		GlobalSettingName[MessagesVerificationTicketsTo]: {String, true},

		GlobalSettingName[StatePruneBelowCount]: {Int, true},
		GlobalSettingName[StateSyncTimeout]:     {Duration, false},

		GlobalSettingName[StuckCheckInterval]: {Duration, false},
		GlobalSettingName[StuckTimeThreshold]: {Duration, false},

		GlobalSettingName[SmartContractTimeout]:             {Duration, true},
		GlobalSettingName[SmartContractSettingUpdatePeriod]: {Int64, true},

		GlobalSettingName[LfbTicketRebroadcastTimeout]: {Duration, false},
		GlobalSettingName[LfbTicketAhead]:              {Int, false},

		GlobalSettingName[AsyncFetchingMaxSimultaneousFromMiners]:   {Int, false},
		GlobalSettingName[AsyncFetchingMaxSimultaneousFromSharders]: {Int, false},

		GlobalSettingName[DbsEventsEnabled]:         {Boolean, false},
		GlobalSettingName[DbsEventsName]:            {String, false},
		GlobalSettingName[DbsEventsUser]:            {String, false},
		GlobalSettingName[DbsEventsPassword]:        {String, false},
		GlobalSettingName[DbsEventsHost]:            {String, false},
		GlobalSettingName[DbsEventsPort]:            {String, false},
		GlobalSettingName[DbsEventsMaxIdleConns]:    {Int, false},
		GlobalSettingName[DbsEventsMaxOpenConns]:    {Int, false},
		GlobalSettingName[DbsEventsConnMaxLifetime]: {Duration, false},

		GlobalSettingName[DbsAggregateDebug]:        {Boolean, true},
		GlobalSettingName[DbsAggregatePeriod]:       {Int64, true},
		GlobalSettingName[DbsPartitionChangePeriod]: {Int64, true},
		GlobalSettingName[DbsPartitionKeepCount]:    {Int64, true},
		GlobalSettingName[DbsAggregatePageLimit]:    {Int64, true},

		GlobalSettingName[HealthCheckDeepScanEnabled]:          {Boolean, false},
		GlobalSettingName[HealthCheckDeepScanBatchSize]:        {Int64, false},
		GlobalSettingName[HealthCheckDeepScanWindow]:           {Int64, false},
		GlobalSettingName[HealthCheckDeepScanSettleSecs]:       {Duration, false},
		GlobalSettingName[HealthCheckDeepScanIntervalMins]:     {Duration, false},
		GlobalSettingName[HealthCheckDeepScanReportStatusMins]: {Duration, false},

		GlobalSettingName[HealthCheckProximityScanEnabled]:            {Boolean, false},
		GlobalSettingName[HealthCheckProximityScanBatchSize]:          {Int64, false},
		GlobalSettingName[HealthCheckProximityScanWindow]:             {Int64, false},
		GlobalSettingName[HealthCheckProximityScanSettleSecs]:         {Duration, false},
		GlobalSettingName[HealthCheckProximityScanRepeatIntervalMins]: {Duration, false},
		GlobalSettingName[HealthCheckProximityScanRejportStatusMins]:  {Duration, false},
		GlobalSettingName[HealthCheckShowCounters]:                    {Boolean, false},
	}
}
