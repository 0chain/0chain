package bls

/* DKG implementation */

import (
	"github.com/pmer/gobls"
)

const CurveFp254BNb = 0

type PartyId = gobls.ID
type Key = gobls.SecretKey
type VerificationKey = gobls.PublicKey
type Sign = gobls.Sign

type BLSSimpleDKG struct {
	T       int
	N       int
	mSec    []Key
	mVec    []VerificationKey
	sending []BLSKeyShare
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
	gobls.Init(gobls.CurveFp254BNb)
	sec.SetByCSPRNG()

	sec.GetMasterSecretKey(t)
	dkg.mSec = sec.GetMasterSecretKey(t)
	dkg.mVec = gobls.GetMasterPublicKey(dkg.mSec)

	return dkg
}

/* Derive the share for each miner through Set() which calls the polynomial substitution method */
func (dkg *BLSSimpleDKG) ComputeKeyShare(forID []PartyId) ([]Key, error) {

	var eachShare Key
	secShares := make([]Key, dkg.N)
	for i := 0; i < dkg.N; i++ {
		err := eachShare.Set(dkg.mSec, &forID[i])
		if err != nil {
			return nil, nil
		}
		secShares[i] = eachShare
	}
	return secShares, nil
}

func (dkg *BLSSimpleDKG) GetKeyShareForOther(t int, to PartyId) KeyShare {
	//TODO
	return KeyShare
}

func (dkg *BLSSimpleDKG) ReceiveKeyShare(from PartyId, share KeyShare) error {
	// TODO
	return nil
}
