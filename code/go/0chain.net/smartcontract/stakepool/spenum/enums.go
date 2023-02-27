package spenum

//go:generate msgp -v -io=false -tests=false

func init() {
	initRewardString()
}

type Provider int

const (
	Invalid Provider = iota
	Miner
	Sharder
	Blobber
	Validator
	Authorizer
)

var providerString = []string{"unknown", "miner", "sharder", "blobber", "validator", "authorizer"}

var Providers = map[string]Provider{
	"invalid":    Invalid,
	"miner":      Miner,
	"sharder":    Sharder,
	"blobber":    Blobber,
	"validator":  Validator,
	"authorizer": Authorizer,
}

func (p Provider) String() string {
	if p < 0 || int(p) >= len(providerString) {
		return "unknown"
	}

	return providerString[p]
}

func ToProviderType(ps string) Provider {
	for i, s := range providerString {
		if s == ps {
			return Provider(i)
		}
	}
	return 0
}

type PoolStatus int

const (
	Active PoolStatus = iota
	Pending
	Unstaking
	Deleted
)

var poolString = []string{"active", "pending", "inactive", "unstaking", "deleting", "deleted"}

func (p PoolStatus) String() string {
	if int(p) < len(poolString) && int(p) >= 0 {
		return poolString[p]
	}
	return "unknown pool status"
}

func (p PoolStatus) Int() int {
	return int(p)
}

type Reward int

var rewardString []string

const (
	MinLockDemandReward Reward = iota
	BlockRewardMiner
	BlockRewardSharder
	BlockRewardBlobber
	FeeRewardMiner
	FeeRewardSharder
	ValidationReward
	FileDownloadReward
	ChallengePassReward
	ChallengeSlashPenalty
	CancellationChargeReward
	NumOfRewards
)

func initRewardString() {
	rewardString = make([]string, NumOfRewards+1)
	rewardString[MinLockDemandReward] = "min_lock_demand"
	rewardString[BlockRewardMiner] = "block_reward_miner"
	rewardString[BlockRewardSharder] = "block_reward_sharder"
	rewardString[BlockRewardBlobber] = "block_reward_blobber"
	rewardString[FeeRewardMiner] = "fee_miner"
	rewardString[FeeRewardSharder] = "fee_sharder"
	rewardString[ValidationReward] = "validation_reward"
	rewardString[FileDownloadReward] = "file_download_reward"
	rewardString[ChallengePassReward] = "challenge_pass_reward"
	rewardString[ChallengeSlashPenalty] = "challenge_slash"
	rewardString[CancellationChargeReward] = "cancellation_charge"
	rewardString[NumOfRewards] = "invalid"
}

func (r Reward) String() string {
	if int(r) < len(rewardString) && int(r) >= 0 {
		return rewardString[r]
	}
	return "unknown_reward"
}

func (r Reward) Int() int {
	return int(r)
}
