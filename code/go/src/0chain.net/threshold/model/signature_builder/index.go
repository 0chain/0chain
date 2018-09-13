package signature_builder

import (
	. "0chain.net/threshold/model"
	. "0chain.net/threshold/model/party"
)

type SignatureBuilder struct{
	party *Party
	msg Msg
}

func New(p *Party, m Msg) SignatureBuilder {
	return SignatureBuilder{
		party: p,
		msg: m,
	}
}
func (b *SignatureBuilder) selfSign() {
}
func (b *SignatureBuilder) receiveShare(i PartyId, share SignatureShare) bool {
	return false
}

func (b *SignatureBuilder) isErr() bool  {
	return false
}
func (b *SignatureBuilder) isDone() bool {
	return false
}
