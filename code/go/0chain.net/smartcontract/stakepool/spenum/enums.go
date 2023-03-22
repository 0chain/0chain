package spenum

//go:generate msgp -v -io=false -tests=false

func init() {
	initRewardString()
}

type Provider int

const (
	Miner Provider = iota + 1
	Sharder
	Blobber
	Validator
	Authorizer
)

var providerString = []string{"invalid", "miner", "sharder", "blobber", "validator", "authorizer"}

var Providers = map[string]Provider{
	"miner":      Miner,
	"sharder":    Sharder,
	"blobber":    Blobber,
	"validator":  Validator,
	"authorizer": Authorizer,
}

func (p Provider) String() string {
	if p < 1 || int(p) >= len(providerString) {
		return "unknown"
	}
	return providerString[p]
}

func ToProviderType(ps string) Provider {
	for name, pType := range Providers {
		if name == ps {
			return pType
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
