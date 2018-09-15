package model

type SignatureBuilder struct {
	party *Party
	msg   Msg

	GroupSignature
}

func NewSignatureBuilder(p *Party, m Msg) SignatureBuilder {
	return SignatureBuilder{
		party:          p,
		msg:            m,
		GroupSignature: EmptyGroupSignature,
	}
}

func (b *SignatureBuilder) selfSign() {
	// TODO
}

func (b *SignatureBuilder) receiveShare(i PartyId, share SignatureShare) error {
	// TODO
	return nil
}

func (b *SignatureBuilder) isDone() bool {
	return b.GroupSignature != EmptyGroupSignature
}
