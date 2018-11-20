package bls

/* DKG implementation */

import (
	"fmt"
	"strconv"

	"github.com/pmer/gobls"
)

/*DKG - to manage DKG process */
type DKG struct {
	T      int
	N      int
	secKey Key
	mSec   []Key

	secSharesMap      map[PartyID]Key
	receivedSecShares []Key
	GpPubKey          GroupPublicKey

	SecKeyShareGroup Key
	ID               PartyID
}

/* init -  To initialize a point on the curve */
func init() {
	gobls.Init(gobls.CurveFp254BNb)

}

/*MakeDKG - to create a dkg object */
func MakeDKG(t, n int) DKG {

	dkg := DKG{
		T:                 t,
		N:                 n,
		secKey:            Key{},
		mSec:              make([]Key, t),
		secSharesMap:      make(map[PartyID]Key, n),
		receivedSecShares: make([]Key, n),
		GpPubKey:          GroupPublicKey{},
		SecKeyShareGroup:  Key{},
		ID:                PartyID{},
	}

	dkg.secKey.SetByCSPRNG()

	dkg.mSec = dkg.secKey.GetMasterSecretKey(t)

	return dkg
}

/*ComputeIDdkg - to create an ID of party of type PartyID */
func ComputeIDdkg(minerID int) PartyID {

	//TODO: minerID here is the index. Change it to miner ID. Neha has fix for this
	var forID PartyID
	err := forID.SetDecString(strconv.Itoa(minerID + 1))
	if err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
	}

	return forID
}

/*ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method */
func (dkg *DKG) ComputeDKGKeyShare(forID PartyID) (Key, error) {

	var secVec Key
	err := secVec.Set(dkg.mSec, &forID)
	if err != nil {
		return Key{}, nil
	}
	dkg.secSharesMap[forID] = secVec

	return secVec, nil
}

/*GetKeyShareForOther - Get the DKGKeyShare for this Miner specified by the PartyID */
func (dkg *DKG) GetKeyShareForOther(to PartyID) *DKGKeyShare {

	indivShare, ok := dkg.secSharesMap[to]
	if !ok {
		fmt.Println("Share not derived for the miner")
	}

	dShare := &DKGKeyShare{m: indivShare}

	return dShare
}

/*AggregateShares - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *DKG) AggregateShares() {
	var sec Key

	for i := 0; i < len(dkg.receivedSecShares); i++ {
		sec.Add(&dkg.receivedSecShares[i])
	}
	dkg.SecKeyShareGroup = sec

}
