package spenum

//go:generate msgp -v -io=false -tests=false

type Provider int

const (
	Miner Provider = iota + 1
	Sharder
	Blobber
	Validator
	Authorizer
)

var providerString = []string{"unknown", "miner", "sharder", "blobber", "validator", "authorizer"}

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
	Inactive
	Unstaking
	Deleting
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
)

var rewardString = []string{
	"block_reward_miner",
	"block_reward_sharder",
	"block_reward_blobber",
	"fees",
	"validation",
	"file download",
	"challenge pass",
	"challenge slash",
	"cancellation charge",
	"min lock demand",
}

func (r Reward) String() string {
	return rewardString[r]
}

func (r Reward) Int() int {
	return int(r)
}
