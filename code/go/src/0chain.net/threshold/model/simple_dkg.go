package model

import (
    "github.com/pmer/gobls"
)

type KeyShare struct {
	m Key
	v VerificationKey
}

var ValidKeyShare = KeyShare{}
var InvalidKeyShare = KeyShare{}
var EmptyKeyShare = KeyShare{}

func InitSimpleDKG() {
    (*gobls.SecretKey)(&ValidKeyShare.m).SetArray([]uint64{1, 2, 3, 4})
    (*gobls.SecretKey)(&ValidKeyShare.v).SetArray([]uint64{1, 2, 3, 4})
    (*gobls.SecretKey)(&InvalidKeyShare.m).SetArray([]uint64{5, 6, 7, 8})
    (*gobls.SecretKey)(&InvalidKeyShare.v).SetArray([]uint64{5, 6, 7, 8})
}

type SimpleDKG struct {
	T           int
	N           int
	sending     []KeyShare
	received    []KeyShare
	numReceived int
}

// Requires that 0 < t <= n.
func NewSimpleDKG(t, n int) SimpleDKG {
	dkg := SimpleDKG{
		T:           t,
		N:           n,
		sending:     make([]KeyShare, n),
		received:    make([]KeyShare, n),
		numReceived: 0,
	}

	dkg.generateKeySharesForOthers()

	// Give ourselves our own key share.
	dkg.received[0] = dkg.sending[0]
	dkg.numReceived += 1

	return dkg
}

func (d *SimpleDKG) generateKeySharesForOthers() {
	// TODO
	for i := range d.sending {
		d.sending[i] = ValidKeyShare
	}
}

func (d *SimpleDKG) GetShareFor(to PartyId) KeyShare {
	return d.sending[to]
}

func (d *SimpleDKG) GetShareFrom(from PartyId) KeyShare {
	return d.received[from]
}

func (d *SimpleDKG) ReceiveShare(from PartyId, share KeyShare) error {
	err := d.validateShare(from, share)
	if err != nil {
		return err
	}

	if d.received[from] == EmptyKeyShare {
		d.received[from] = share
		d.numReceived += 1
	}
	return nil
}

func (d *SimpleDKG) validateShare(from PartyId, share KeyShare) error {
	// TODO

	notARealValidationTest := func(share KeyShare) bool {
		return share != ValidKeyShare
	}

	if notARealValidationTest(share) {
		return ThresholdError{
			By:    from,
			Cause: "Invalid signature",
		}
	}

	return nil
}

func (d *SimpleDKG) IsDone() bool {
	return d.numReceived == d.N
}

// Used in unit tests. Provide a deep copy of the received key shares.
func (d *SimpleDKG) clone() SimpleDKG {
	received := make([]KeyShare, d.N)
	copy(received, d.received)

	return SimpleDKG{
		T:           d.T,
		N:           d.N,
		sending:     d.sending,
		received:    received,
		numReceived: d.numReceived,
	}
}
