package bls

/* DKG implementation */

import (
	"fmt"
	"strconv"

	"github.com/pmer/gobls"
)

/*SimpleDKG - to manage DKG process */
type SimpleDKG struct {
	T                 int
	N                 int
	SecKey            Key
	pubKey            VerificationKey
	mSec              []Key
	Vvec              []VerificationKey
	secSharesMap      map[PartyID]Key
	receivedSecShares []Key
	GpPubKey          GroupPublicKey
	SecKeyShareGroup  Key
	ID                PartyID
}

/* init -  To initialize a point on the curve */
func init() {
	gobls.Init(gobls.CurveFp254BNb)

}

/*MakeSimpleDKG - to create a dkg object */
func MakeSimpleDKG(t, n int) SimpleDKG {

	dkg := SimpleDKG{
		T:                 t,
		N:                 n,
		SecKey:            Key{},
		pubKey:            VerificationKey{},
		mSec:              make([]Key, t),
		Vvec:              make([]VerificationKey, t),
		secSharesMap:      make(map[PartyID]Key, n),
		receivedSecShares: make([]Key, n),
		GpPubKey:          GroupPublicKey{},
		SecKeyShareGroup:  Key{},
		ID:                PartyID{},
	}

	dkg.SecKey.SetByCSPRNG()
	dkg.pubKey = *(dkg.SecKey.GetPublicKey())
	dkg.mSec = dkg.SecKey.GetMasterSecretKey(t)
	dkg.Vvec = gobls.GetMasterPublicKey(dkg.mSec)
	dkg.GpPubKey = dkg.Vvec[0]

	return dkg
}

/*ComputeIDdkg - to create an ID of party of type PartyID */
func ComputeIDdkg(minerID int) PartyID {
	var forID PartyID
	err := forID.SetDecString(strconv.Itoa(minerID + 1))
	if err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
	}

	return forID
}

/*ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method */
func (dkg *SimpleDKG) ComputeDKGKeyShare(forID PartyID) (Key, error) {

	var secVec Key
	err := secVec.Set(dkg.mSec, &forID)
	if err != nil {
		return Key{}, nil
	}
	dkg.secSharesMap[forID] = secVec

	return secVec, nil
}

/*GetKeyShareForOther - Get the DKGKeyShare for this Miner specified by the PartyID */
func (dkg *SimpleDKG) GetKeyShareForOther(to PartyID) *DKGKeyShare {

	indivShare, ok := dkg.secSharesMap[to]
	if !ok {
		fmt.Println("Share not derived for the miner")
	}

	dShare := &DKGKeyShare{m: indivShare}
	pubShare := indivShare.GetPublicKey()
	dShare.v = *pubShare
	return dShare
}

/*AggregateShares - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *SimpleDKG) AggregateShares() {
	var sec Key

	for i := 0; i < len(dkg.receivedSecShares); i++ {
		sec.Add(&dkg.receivedSecShares[i])
	}
	dkg.SecKeyShareGroup = sec

}
