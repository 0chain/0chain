package spenum

//go:generate msgp -v -io=false -tests=false

type Provider int

const (
	Miner Provider = iota
	Sharder
	Blobber
	Validator
	Authorizer
)

var providerString = []string{"miner", "sharder", "blobber", "validator", "authorizer"}

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

var poolString = []string{"active", "pending", "inactive", "unstaking", "deleting"}

func (p PoolStatus) String() string {
	return poolString[p]
}
