package model

type Msg []byte

type GroupPublicKey Key
type GroupSignature Key

type PrivateKeyShare Key
type SignatureShare Key

var EmptyGroupSignature = GroupSignature{}

type Party struct {
	t int
	n int

	public        GroupPublicKey
	verifications []VerificationKey

	privateShare PrivateKeyShare
}

func NewParty(dkg *SimpleDKG) Party {
	return Party{
		t: dkg.T,
		n: dkg.N,

		// TODO: Initialize remaining items from SimpleDKG.
		public:        GroupPublicKey{},
		verifications: nil,
		privateShare:  PrivateKeyShare{},
	}
}

func (p *Party) sign(m Msg) SignatureShare {
	// TODO
	return SignatureShare{}
}

func (p *Party) verifyShare(m Msg, i PartyId, s SignatureShare) error {
	// TODO
	return nil
}

func (p *Party) recoverGroup(m Msg, ss []SignatureShare) (GroupSignature, error) {
	// TODO
	return EmptyGroupSignature, nil
}

func (p *Party) verifyGroup(m Msg, s GroupSignature) error {
	// TODO
	return nil
}
