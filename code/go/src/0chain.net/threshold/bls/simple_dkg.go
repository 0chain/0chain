package bls

/* DKG implementation */

import (
	"github.com/pmer/gobls"
)

const CurveFp254BNb = 0

type PartyId gobls.ID
type Key gobls.SecretKey
type VerificationKey gobls.PublicKey
type Sign gobls.Sign

type BLSSimpleDKG struct {
	T    int
	N    int
	mSec []Key
	mVec []VerificationKey
}

type BLSKeyShare struct {
	m Key
	v VerificationKey
}

func MakeSimpleDKG(t, n int) BLSSimpleDKG {

	dkg := BLSSimpleDKG{
		T:       t,
		N:       n,
		mSec:    make([]Key, t),
		mVec:    make([]VerificationKey, t),
		sending: make([]BLSKeyShare, n),
	}

	var sec Key
	gobls.Init(CurveFp254BNb)

	(*gobls.SecretKey)(&sec).SetByCSPRNG()

	return dkg
}

func (dkg *BLSSimpleDKG) GetKeyShareForOther(t int, to PartyId) KeyShare {
	//TODO
	return KeyShare
}

func (dkg *BLSSimpleDKG) ReceiveKeyShare(from PartyId, share KeyShare) error {
	// TODO
	return nil
}
