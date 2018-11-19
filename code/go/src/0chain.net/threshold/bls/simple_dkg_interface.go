package bls

import (
	"github.com/pmer/gobls"
)

/*Key - Is of type gobls.SecretKey*/
type Key = gobls.SecretKey

/*VerificationKey - Is of type gobls.PublicKey*/
type VerificationKey = gobls.PublicKey

/*SimpleDKGI - Interface for DKG*/
type SimpleDKGI interface {
	ComputeDKGKeyShare(forID PartyID) (Key, error)
	ComputeGpPublicKeyShareShares(recVvec []VerificationKey, fromID PartyID) (VerificationKey, error)
	GetKeyShareForOther(to PartyID) *DKGKeyShare
	AggregateShares()
	CalcGroupsVvec(vVec []VerificationKey)
}

/*DKGKeyShare - DKG share of each party */
type DKGKeyShare struct {
	m Key
	v VerificationKey
}
