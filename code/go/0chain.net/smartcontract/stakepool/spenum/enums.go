package spenum

import "errors"

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

func (p PoolStatus) String() (string, error) {
	if int(p) < len(poolString) && int(p) >= 0 {
		return poolString[p], nil
	}
	return "", errors.New("unknown pool status")
}

func (p PoolStatus) Size() int {
	return len(poolString)
}
