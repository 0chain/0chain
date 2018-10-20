package bls

/* DKG implementation */

import (
	"fmt"
	"strconv"

	//	. "0chain.net/logging"
	"github.com/pmer/gobls"
)

type BLSSimpleDKG struct {
	T                 int
	N                 int
	SecKey            Key
	pubKey            VerificationKey
	mSec              []Key
	Vvec              []VerificationKey
	secSharesMap      map[PartyId]Key
	receivedSecShares []Key
	GpPubKey          GroupPublicKey
	SecKeyShareGroup  Key
	SelfShare         Key
	ID                PartyId
}

func init() {
	gobls.Init(gobls.CurveFp254BNb)

}
func MakeSimpleDKG(t, n int) BLSSimpleDKG {

	dkg := BLSSimpleDKG{
		T:                 t,
		N:                 n,
		SecKey:            Key{},
		pubKey:            VerificationKey{},
		mSec:              make([]Key, t),
		Vvec:              make([]VerificationKey, t),
		secSharesMap:      make(map[PartyId]Key, n),
		receivedSecShares: make([]Key, n),
		GpPubKey:          GroupPublicKey{},
		SecKeyShareGroup:  Key{},
		SelfShare:         Key{},
		ID:                PartyId{},
	}

	dkg.SecKey.SetByCSPRNG()
	dkg.pubKey = *(dkg.SecKey.GetPublicKey())
	dkg.mSec = dkg.SecKey.GetMasterSecretKey(t)
	dkg.Vvec = gobls.GetMasterPublicKey(dkg.mSec)
	dkg.GpPubKey = dkg.Vvec[0]

	return dkg
}

/* threshold_helper.go calls this function to compute the ID */
func ComputeIDdkg(minerID int) PartyId {
	var forID PartyId
	err := forID.SetDecString(strconv.Itoa(minerID + 1))
	if err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
	}

	return forID
}

/* ComputeDKGKeyShare - Derive the share for each miner through gobls.Set() which calls the polynomial substitution method used in threshold_helper.go */

func (dkg *BLSSimpleDKG) ComputeDKGKeyShare(forID PartyId) (Key, error) {

	var secVec Key
	err := secVec.Set(dkg.mSec, &forID)
	if err != nil {
		return Key{}, nil
	}
	dkg.secSharesMap[forID] = secVec

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

/* Each party aggregates the received shares from other party */
func (dkg *BLSSimpleDKG) AggregateShares() {
	var sec Key

	for i := 0; i < len(dkg.receivedSecShares); i++ {
		sec.Add(&dkg.receivedSecShares[i])
	}
	dkg.SecKeyShareGroup = sec

}
