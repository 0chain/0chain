package model_byzantine_dkg

import (
	. "0chain.net/threshold/model"
	simple "0chain.net/threshold/model/simple_dkg"
)

type Receipt struct {
	m Key
	v VerificationKey
}

type DKG struct{
	dkg simple.DKG
}

func New(t T, n N) DKG {
	return DKG{
		dkg: simple.New(t, n),
	}
}

func (dkg *DKG) ReceiveComplaint(from, against PartyId) {
}

func (dkg *DKG) ReceiveDefend(from, against PartyId,
	m Key, v VerificationKey) {
}

func (dkg *DKG) GetDisqualified() []PartyId {
	return []PartyId{}
}
