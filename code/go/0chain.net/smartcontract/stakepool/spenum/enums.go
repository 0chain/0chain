package spenum

import "strconv"

//go:generate msgp -v -io=false -tests=false

type Provider int

const (
	Unknown Provider = iota
	Miner
	Sharder
	Blobber
	Validator
	Authorizer
)

var providerString = []string{"unknown", "miner", "sharder", "blobber", "validator", "authorizer"}
var Providers = map[string]Provider{
	"miner":      Miner,
	"sharder":    Sharder,
	"blobber":    Blobber,
	"validator":  Validator,
	"authorizer": Authorizer,
}

func (p Provider) String() string {
	if int(p) >= len(providerString) {
		return "invalid: " + strconv.Itoa(int(p))
	}
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
