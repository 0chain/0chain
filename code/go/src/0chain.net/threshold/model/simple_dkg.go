package model

type KeyShare struct {
	m Key
	v VerificationKey
}

var EmptyKeyShare = KeyShare{}

type SimpleDKG struct {
	T           int
	N           int
	sending     []KeyShare
	received    []KeyShare
	numReceived int
}

func NewSimpleDKG(t, n int) SimpleDKG {
	return SimpleDKG{
		T:           t,
		N:           n,
		sending:     make([]KeyShare, n),
		received:    make([]KeyShare, n),
		numReceived: 0,
	}
}

func (d *SimpleDKG) GetShareFor(i PartyId) KeyShare {
	return d.sending[i]
}

func (d *SimpleDKG) GetShareFrom(i PartyId) KeyShare {
	return d.received[i]
}

func (d *SimpleDKG) ReceiveShare(i PartyId, share KeyShare) error {
	// TODO: Check validity and return error if invalid.
	if d.received[i] != EmptyKeyShare {
		d.received[i] = share
		d.numReceived += 1
	}
	return nil
}

func (d *SimpleDKG) IsDone() bool {
	return d.numReceived == d.N
}
