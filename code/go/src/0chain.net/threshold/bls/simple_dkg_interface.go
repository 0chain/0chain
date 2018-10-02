package bls

import (
	"github.com/pmer/gobls"
)

type PartyId = gobls.ID
type Key = gobls.SecretKey
type VerificationKey = gobls.PublicKey
type Sign = gobls.Sign
type KeyShare interface{}

type SignI interface{}

type SimpleDKG interface {
	ComputeKeyShare(forIDs []PartyId) ([]Key, error)
	GetKeyShareForOther(to PartyId) *DKGKeyShare
	ReceiveKeyShare(from PartyId, d *DKGKeyShare) error
}

type DKGKeyShare struct {
	m Key
	v VerificationKey
}
