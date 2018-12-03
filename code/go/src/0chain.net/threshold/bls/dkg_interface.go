package bls

import "github.com/herumi/bls/ffi/go/bls"

/*Key - Is of type mcl.SecretKey*/
type Key = bls.SecretKey

/*DKGI - Interface for DKG*/
type DKGI interface {
	ComputeDKGKeyShare(forID PartyID) (Key, error)
	GetKeyShareForOther(to PartyID) *DKGKeyShare
	AggregateShares()
}

/*DKGKeyShare - DKG share of each party */
type DKGKeyShare struct {
	m Key
}
