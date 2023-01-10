package benchmark

import "0chain.net/core/common"

//go:generate msgp -io=false -tests=false -unexported=true -v

type BenchDataMpt struct {
	Clients           []string         `json:"clients"`
	PublicKeys        []string         `json:"publicKeys"`
	PrivateKeys       []string         `json:"privateKeys"`
	Miners            []string         `json:"miners"`
	Sharders          []string         `json:"sharders"`
	SharderKeys       []string         `json:"sharderKeys"`
	InactiveSharder   string           `json:"inactiveSharder"`
	InactiveSharderPK string           `json:"inactiveSharderPK"`
	Now               common.Timestamp `json:"now"`
}
