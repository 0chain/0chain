package bls

import (
	"github.com/pmer/gobls"
)

/*Key - Is of type gobls.SecretKey*/
type Key = gobls.SecretKey

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
