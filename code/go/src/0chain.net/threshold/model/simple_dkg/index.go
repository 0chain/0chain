package model_simple_dkg

import (
	. "0chain.net/threshold/model"
)

type Receipt struct {
	m Key
	v VerificationKey
}

type DKG struct {
	T
	N
	receipts []Receipt
}

func New(t T, n N) DKG {
	return DKG{
		T:        t,
		N:        n,
		receipts: make([]Receipt, n),
	}
}

func (dkg *DKG) GetShareFor(i PartyId) (m Key, v VerificationKey) {
	return Key{}, VerificationKey{}
}

func (dkg *DKG) ReceiveShare(i PartyId, m Key, v VerificationKey) error {
	dkg.receipts = append(dkg.receipts, Receipt{m: m, v: v})
	return nil
}

func (dkg *DKG) IsDone() bool {
	return dkg.N == N(len(dkg.receipts))
}
