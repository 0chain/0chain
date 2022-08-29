package endpoint

const (
	GetMinerStats = "/v1/miner/get/stats"
)

const (
	GetClient        = "/v1/client/get"
	PutClient        = "/v1/client/put"
	GetClientBalance = "/v1/client/get/balance"
)

const (
	PutTransaction = "/v1/transaction/put"
)

const (
	MinerToMinerDkgShare           = "/v1/_m2m/dkg/share"
	MinerToMinerRoundVRFSender     = "/v1/_m2m/round/vrf_share"
	MinerToMinerVerifyBlock        = "/v1/_m2m/block/verify"
	MinerToMinerNotarizedBlock          = "/v1/_m2m/block/notarized_block"
	MinerToMinerBlockVerificationTicket = "/v1/_m2m/block/verification_ticket"
	MinerToMinerBlockNotarization       = "/v1/_m2m/block/notarization"
	MinerToMinerChainStart         = "/v1/_m2m/chain/start"
)

const (
	AnyServiceToMinerGetNotarizedBlock = "/v1/_x2m/block/notarized_block/get"
	AnyServiceToMinerGetStateChange    = "/v1/_x2m/block/state_change/get"
	AnyServiceToMinerGetState          = "/v1/_x2m/state/get"
)