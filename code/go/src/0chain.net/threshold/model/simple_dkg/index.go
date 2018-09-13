package simple_dkg

import (
	. "0chain.net/threshold/model"
)

type Receipt struct {
	m Key
	v VerificationKey
}

type DKG struct {
	t        T
	n        N
	receipts []Receipt
}

func New(t T, n N) DKG {
	return DKG{
		t:        t,
		n:        n,
		receipts: make([]Receipt, n),
	}
}
func (dkg *DKG) GetShareFor(i PartyId) (m Key, v VerificationKey) {
	return Key{}, VerificationKey{}
}
func (dkg *DKG) ReceiveShare(i PartyId, m Key, v VerificationKey) bool {
	dkg.receipts = append(dkg.receipts, Receipt{m: m, v: v})
	return true
}
func (dkg *DKG) IsDone() bool { return dkg.n == N(len(dkg.receipts)) }
