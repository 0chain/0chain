package bls

/* DKG implementation */

import (
	"fmt"

	"github.com/pmer/gobls"
)

const CurveFp254BNb = 0

type BLSSimpleDKG struct {
	T            int
	N            int
	mSec         []Key
	mVec         []VerificationKey
	SecSharesMap map[PartyId]Key
}

func MakeSimpleDKG(t, n int) BLSSimpleDKG {

	dkg := BLSSimpleDKG{
		T:            t,
		N:            n,
		mSec:         make([]Key, t),
		mVec:         make([]VerificationKey, t),
		SecSharesMap: make(map[PartyId]Key, n),
	}

	var sec Key
	gobls.Init(gobls.CurveFp254BNb)
	sec.SetByCSPRNG()

	sec.GetMasterSecretKey(t)
	dkg.mSec = sec.GetMasterSecretKey(t)
	dkg.mVec = gobls.GetMasterPublicKey(dkg.mSec)
	return dkg
}

/* ComputeKeyShare - Derive the share for each miner through gobls.Set() which calls the polynomial substitution method */
func (dkg *BLSSimpleDKG) ComputeKeyShare(forIDs []PartyId) ([]Key, error) {

	var eachShare Key
	secShares := make([]Key, dkg.N)
	for i := 0; i < dkg.N; i++ {
		err := eachShare.Set(dkg.mSec, &forIDs[i])
		if err != nil {
			return nil, nil
		}
		dkg.SecSharesMap[forIDs[i]] = eachShare
		secShares[i] = eachShare
	}

	return secShares, nil
}

/* GetKeyShareForOther - Get the DKGKeyShare for this Miner specified by the PartyId */
func (dkg *BLSSimpleDKG) GetKeyShareForOther(to PartyId) *DKGKeyShare {
	indivShare, ok := dkg.SecSharesMap[to]
	if !ok {
		fmt.Println("Share not derived for the miner")
	}
	dShare := &DKGKeyShare{m: indivShare}
	pubShare := indivShare.GetPublicKey()
	dShare.v = *pubShare
	return dShare
}

/* ReceiveKeyShare - Get the share from a specific PartyId */
func (dkg *BLSSimpleDKG) ReceiveKeyShare(from PartyId, d *DKGKeyShare) error {

	isValid := dkg.ValidateShare(from, d)
	if isValid != true {
		return fmt.Errorf("The private key share from %s which is %s is not valid\n", from.GetHexString(), d.m.GetHexString())

	}
	return nil
}

/* ValidateShare -  Validate the share received from PartyId */
func (dkg *BLSSimpleDKG) ValidateShare(from PartyId, d *DKGKeyShare) bool {

	indivShare, _ := dkg.SecSharesMap[from]
	pubKey := indivShare.GetPublicKey()

	if pubKey.IsEqual(&d.v) {
		return true
	}

	return false

}
