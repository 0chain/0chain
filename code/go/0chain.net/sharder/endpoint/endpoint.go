package endpoint

const (
	NodesState = "/v1/state/nodes"
)

const (
	GetSharderStats = "/v1/sharder/get/stats"
)

const (
	HealthCheckFunction = "/_healthcheck"
	HealthCheck         = "/v1/healthcheck"
)

const (
	SmartContractFunction = "/v1/screst"
)

const (
	GetMagicBlock = "/v1/block/magic/get"
)

const (
	GetTransaction             = "/v1/transaction/get"
	GetTransactionConfirmation = "/v1/transaction/get/confirmation"
)

const (
	MinerToSharderGetFinalizedBlock       = "/v1/_m2s/block/finalized"
	MinerToSharderGetNotarisedBlock       = "/v1/_m2s/block/notarized"
	MinerToSharderKickNotarisedBlock      = "/v1/_m2s/block/notarized/kick"
	MinerToSharderGetLatestFinalizedBlock = "/v1/_m2s/block/latest_finalized/get"
)

const (
	SharderToSharderGetRound          = "/v1/_s2s/round/get"
	SharderToSharderGetLatestRound    = "/v1/_s2s/latest_round/get"
	SharderToSharderGetBlock          = "/v1/_s2s/block/get"
	SharderToSharderGetBlockSummary   = "/v1/_s2s/blocksummary/get"
	SharderToSharderGetBlockSummaries = "/v1/_s2s/blocksummaries/get"
	SharderToSharderGetRoundSummaries = "/v1/_s2s/roundsummaries/get"
)

const (
	AnyServiceToSharderGetBlock = "/v1/_x2s/block/get"
)
