package model_simple_dkg

import (
	"0chain.net/threshold/model"
)

type KeyShare struct {
	m model.Key
	v model.VerificationKey
}

var EmptyKeyShare = KeyShare{}

type DKG struct {
	T           int
	N           int
	sending     []KeyShare
	received    []KeyShare
	numReceived int
}

func New(t, n int) DKG {
	return DKG{
		T:           t,
		N:           n,
		sending:     make([]KeyShare, n),
		received:    make([]KeyShare, n),
		numReceived: 0,
	}
}

func (d *DKG) GetShareFor(i model.PartyId) KeyShare {
	return d.sending[i]
}

func (d *DKG) GetShareFrom(i model.PartyId) KeyShare {
	return d.received[i]
}

func (d *DKG) ReceiveShare(i model.PartyId, share KeyShare) error {
	// TODO: Check validity and return error if invalid.
	if d.received[i] != (KeyShare{}) {
		d.received[i] = share
		d.numReceived += 1
	}
	return nil
}

func (d *DKG) IsDone() bool {
	return d.numReceived == d.N
}
