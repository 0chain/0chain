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
	return providerString[p]
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
	return poolString[p]
}

func (p PoolStatus) Int() int {
	return int(p)
}

type Reward int

const (
	BlockReward Reward = iota
	Fees
	Validation
	FileDownload
	ChallengePass
	ChallengeSlash
	CancellationCharge
	MinLockDemand
)

var rewardString = []string{
	"block_reward",
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
