package model_party

import (
	. "0chain.net/threshold/model"
	"0chain.net/threshold/model/simple_dkg"
)

type Msg []byte

type GroupPublicKey Key
type GroupSignature Key

type PrivateKeyShare Key
type SignatureShare Key

type Party struct{}

func New(*model_simple_dkg.DKG) Party {
	return Party{}
}

func (p *Party) sign(m Msg) SignatureShare {
	return SignatureShare{}
}

func (p *Party) verifyShare(m Msg, i PartyId, s SignatureShare) error {
	return nil
}

func (p *Party) recoverGroup(m Msg, ss []SignatureShare) error {
	return nil
}

func (p *Party) verifyGroup(m Msg, s GroupSignature) error {
	return nil
}
