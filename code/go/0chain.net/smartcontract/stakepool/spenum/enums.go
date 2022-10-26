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

var rewardString = []string{"block_reward", "fees", "validation", "read_file", "write_file"}

func (r Reward) String() string {
	return rewardString[r]
}
