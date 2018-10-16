package bls

/* DKG implementation */

import (
	"fmt"
	"strconv"

	//. "0chain.net/logging"
	"github.com/pmer/gobls"
)

type BLSSimpleDKG struct {
	T                    int
	N                    int
	SecKey               Key
	pubKey               VerificationKey
	mSec                 []Key
	Vvec                 []VerificationKey
	secSharesMap         map[PartyId]Key
	receivedSecShares    []Key
	receivedSecDKGShares []Key
	receivedPubShares    []VerificationKey
	numReceived          int
	GpPubKey             GroupPublicKey
	SecKeyShareGroup     Key
	SelfShare            Key
}

func init() {
	gobls.Init(gobls.CurveFp254BNb)

}
func MakeSimpleDKG(t, n int) BLSSimpleDKG {

	dkg := BLSSimpleDKG{
		T:                    t,
		N:                    n,
		SecKey:               Key{},
		pubKey:               VerificationKey{},
		mSec:                 make([]Key, t),
		Vvec:                 make([]VerificationKey, t),
		secSharesMap:         make(map[PartyId]Key, n),
		receivedSecShares:    make([]Key, n),
		receivedPubShares:    make([]VerificationKey, n),
		numReceived:          0,
		GpPubKey:             GroupPublicKey{},
		SecKeyShareGroup:     Key{},
		SelfShare:            Key{},
		receivedSecDKGShares: make([]Key, n),
	}

	dkg.SecKey.SetByCSPRNG()
	dkg.pubKey = *(dkg.SecKey.GetPublicKey())
	dkg.mSec = dkg.SecKey.GetMasterSecretKey(t)
	dkg.Vvec = gobls.GetMasterPublicKey(dkg.mSec)
	dkg.GpPubKey = dkg.Vvec[0]
	dkg.numReceived += 1
	return dkg
}

func ComputeIDdkg(minerID int) PartyId {
	var forID PartyId
	err := forID.SetDecString(strconv.Itoa(minerID))
	if err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
	}

	return forID
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

/* ComputeDKGKeyShare - Derive the share for each miner through gobls.Set() which calls the polynomial substitution method */

func (dkg *BLSSimpleDKG) ComputeDKGKeyShare(forID PartyId) (Key, error) {

	var secVec Key
	err := secVec.Set(dkg.mSec, &forID)
	if err != nil {
		return Key{}, nil
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

	dkg.receivedSecShares = append(dkg.receivedSecShares, d.m)
	dkg.receivedPubShares = append(dkg.receivedPubShares, d.v)
	dkg.numReceived++

	if d == nil || dkg.receivedSecShares == nil || dkg.receivedPubShares == nil {
		return fmt.Errorf("Error in share received")
	}
	return nil
}

/* IsDone - Function to check if the DKG is done */

func (dkg *BLSSimpleDKG) IsDone() bool {
	return dkg.numReceived == dkg.N

}

/* Each party aggregates the received shares from other party */
func (dkg *BLSSimpleDKG) AggregateShares() (*AfterDKGKeyShare, error) {

	for _, pri := range dkg.receivedSecShares {
		dkg.SecKey.Add(&pri)
	}

	for _, pub := range dkg.receivedPubShares {
		dkg.pubKey.Add(&pub)
	}

	aggShare := &AfterDKGKeyShare{m: dkg.SecKey}
	aggShare.v = dkg.pubKey
	return aggShare, nil
}
