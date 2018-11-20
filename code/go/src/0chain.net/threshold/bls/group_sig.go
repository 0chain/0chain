package bls

import (
	. "0chain.net/logging"
)

/*SimpleBLS - to manage BLS process */
type SimpleBLS struct {
	T                int
	N                int
	Msg              Message
	SigShare         Sign
	gpPubKey         GroupPublicKey
	SecKeyShareGroup Key
	GpSign           Sign
	ID               PartyID
}

/*MakeSimpleBLS - to create bls object */
func MakeSimpleBLS(dkg *DKG) SimpleBLS {
	bs := SimpleBLS{
		T:        dkg.T,
		N:        dkg.N,
		Msg:      " ",
		SigShare: Sign{},
		gpPubKey: dkg.GpPubKey,

		SecKeyShareGroup: dkg.SecKeyShareGroup,
		GpSign:           Sign{},
		ID:               dkg.ID,
	}
	return bs

}

/*SignMsg - Bls sign share is computed by signing the message r||RBO(r-1) with secret key share of group of that party */
func (bs *SimpleBLS) SignMsg() Sign {

	aggSecKey := bs.SecKeyShareGroup
	sigShare := *aggSecKey.Sign(bs.Msg)
	return sigShare
}

/*RecoverGroupSig - To compute the Gp sign with any k number of BLS sig shares */
func (bs *SimpleBLS) RecoverGroupSig(from []PartyID, shares []Sign) Sign {

	signVec := shares
	idVec := from

	var sig Sign
	err := sig.Recover(signVec, idVec)

	if err == nil {
		bs.GpSign = sig
		return sig
	}

	Logger.Debug("Recover Gp Sig not done, check party.go")

	return sig

}
