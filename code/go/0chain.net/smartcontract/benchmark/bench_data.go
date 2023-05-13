package benchmark

import "0chain.net/core/common"

//go:generate msgp -io=false -tests=false -unexported=true -v

type BenchDataMpt struct {
	Clients              []string         `json:"clients"`
	PublicKeys           []string         `json:"public_keys"`
	PrivateKeys          []string         `json:"private_keys"`
	Miners               []string         `json:"miners"`
	Sharders             []string         `json:"sharders"`
	SharderKeys          []string         `json:"sharder_keys"`
	ValidatorIds         []string         `json:"validator_ids"`
	ValidatorPublicKeys  []string         `json:"validator_public_keys"`
	ValidatorPrivateKeys []string         `json:"validator_private_keys"`
	InactiveSharder      string           `json:"inactive_sharder"`
	InactiveSharderPK    string           `json:"inactive_sharder_pk"`
	Now                  common.Timestamp `json:"now"`
}
