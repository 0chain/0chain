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
	secSharesMap map[PartyId]Key
	numReceived  int
}

func MakeSimpleDKG(t, n int) BLSSimpleDKG {

	dkg := BLSSimpleDKG{
		T:            t,
		N:            n,
		mSec:         make([]Key, t),
		mVec:         make([]VerificationKey, t),
		secSharesMap: make(map[PartyId]Key, n),
		numReceived:  0,
	}

	var sec Key
	gobls.Init(gobls.CurveFp254BNb)
	sec.SetByCSPRNG()

	sec.GetMasterSecretKey(t)
	dkg.mSec = sec.GetMasterSecretKey(t)
	dkg.mVec = gobls.GetMasterPublicKey(dkg.mSec)

	dkg.numReceived += 1
	return dkg
}

/* ComputeKeyShare - Derive the share for each miner through gobls.Set() which calls the polynomial substitution method */

func (dkg *BLSSimpleDKG) ComputeKeyShare(forIDs []PartyId) ([]Key, error) {

	secVec := make([]Key, dkg.N)
	for i := 0; i < dkg.N; i++ {
		err := secVec[i].Set(dkg.mSec, &forIDs[i])
		if err != nil {
			return nil, nil
		}
		dkg.secSharesMap[forIDs[i]] = secVec[i]
	}

	return secVec, nil
}

/* GetKeyShareForOther - Get the DKGKeyShare for this Miner specified by the PartyId */
func (dkg *BLSSimpleDKG) GetKeyShareForOther(to PartyId) *DKGKeyShare {

	indivShare, ok := dkg.secSharesMap[to]
	if !ok {
		fmt.Println("Share not derived for the miner")
	}

	dShare := &DKGKeyShare{m: indivShare}
	pubShare := indivShare.GetPublicKey()
	dShare.v = *pubShare
	return dShare
}

/* ReceiveAndValidateShare - Get the share from a specific PartyId */
func (dkg *BLSSimpleDKG) ReceiveAndValidateShare(from PartyId, d *DKGKeyShare) error {

	isValid := dkg.ValidateShare(from, d)
	if isValid != true {
		return fmt.Errorf("The private key share from %s which is %s is not valid\n", from.GetHexString(), d.m.GetHexString())

	}
	return nil
}

/* ValidateShare -  Validate the share received from PartyId */
func (dkg *BLSSimpleDKG) ValidateShare(from PartyId, d *DKGKeyShare) bool {

	indivShare, _ := dkg.secSharesMap[from]

	pubKey := indivShare.GetPublicKey()
	dkg.numReceived++
	if pubKey.IsEqual(&d.v) {
		return true
	}

	return false

}

/* ReceiveKeyShareFromParty - Gets the seckey share */
func (dkg *BLSSimpleDKG) ReceiveKeyShareFromParty(d *DKGKeyShare) error {

	dkg.numReceived++

	if d == nil {
		return fmt.Errorf("Error in share received")
	}
	return nil
}

/* IsDone - Function to check if the DKG is done */

func (dkg *BLSSimpleDKG) IsDone() bool {
	return dkg.numReceived == dkg.N
}
