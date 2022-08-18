package endpoint

const (
	Root                       = "/"
	HashFunction               = "/_hash"
	SignFunction               = "/_sign"
	ChainStatsFunction         = "/_chain_stats"
	SmartContractStatsFunction = "/_smart_contract_stats"
)

const (
	WhoAmI         = "/_nh/whoami"
	Status         = "/_nh/status"
	GetPoolMembers = "/_nh/getpoolmembers"
	ListMiners     = "/_nh/list/m"
	ListSharders   = "/_nh/list/s"
)

const (
	Diagnostics               = "/_diagnostics"
	DiagnosticsInfoJson       = "/v1/diagnostics/get/info"
	DiagnosticsInfo           = "/_diagnostics/info"
	WalletStatsDiagnostics    = "/_diagnostics/wallet_stats"
	CurrentMbNodesDiagnostics = "/_diagnostics/current_mb_nodes"
	DkgProcessDiagnostics     = "/_diagnostics/dkg_process"
	RoundInfoDiagnostics      = "/_diagnostics/round_info"
	StateDumpDiagnostics      = "/_diagnostics/state_dump"
	MinerStatsDiagnostics     = "/_diagnostics/miner_stats"
	DiagnosticsLogs           = "/_diagnostics/logs"
	DiagnosticsNodeToNodeLogs = "/_diagnostics/n2n_logs"
	DiagnosticsNodeToNodeInfo = "/_diagnostics/n2n/info"
	DiagnosticsMemoryLogs     = "/_diagnostics/mem_logs"
	BlockChainDiagnostics     = "/_diagnostics/block_chain"
)

const (
	GetConfig       = "/v1/config/get"
	UpdateConfig    = "/v1/config/update"
	UpdateAllConfig = "/v1/config/update_all"
)

const (
	GetSmartContractState = "/v1/scstate/get"
	GetSmartContractStats = "/v1/scstats"
)

const (
	GetBlock                            = "/v1/block/get"
	GetBlockStateChange                 = "/v1/block/state_change"
	GetLatestFinalizedBlock             = "/v1/block/get/latest_finalized"
	GetLatestFinalizedTicket            = "/v1/block/get/latest_finalized_ticket"
	GetLatestFinalizedMagicBlock        = "/v1/block/get/latest_finalized_magic_block"
	GetLatestFinalizedMagicBlockSummary = "/v1/block/get/latest_finalized_magic_block_summary"
	GetRecentFinalizedBlock             = "/v1/block/get/recent_finalized"
	GetBlockFeeStats                    = "/v1/block/get/fee_stats"
)

const (
	GetChain      = "/v1/chain/get"
	GetChainStats = "/v1/chain/get/stats"
	PutChain      = "/v1/chain/put"
)

const (
	NodeToNodePostEntity = "/v1/_n2n/entity/post"
	NodeToNodeGetEntity  = "/v1/_n2n/entity_pull/get"
)

const (
	AnyServiceToAnyServiceGetBlockStateChange = "/v1/_x2x/block/state_change"
	AnyServiceToAnyServiceGetNodes            = "/v1/_x2x/state/get_nodes"
)
