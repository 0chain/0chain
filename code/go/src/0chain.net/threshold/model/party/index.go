package party

import (
	. "0chain.net/threshold/model"
	. "0chain.net/threshold/model/simple_dkg"
)

type Msg []byte

type GroupPublicKey Key
type GroupSignature Key

type PrivateKeyShare Key
type SignatureShare Key

type Party struct{}

func New(dkg *DKG) Party                   { return Party{} }
func (p *Party) sign(m Msg) SignatureShare { return SignatureShare{} }
func (p *Party) verifyShare(m Msg, i PartyId, s SignatureShare) bool {
	return false
}
func (p *Party) recoverGroup(m Msg, ss []SignatureShare) bool { return false }
func (p *Party) verifyGroup(m Msg, s GroupSignature) bool     { return false }
