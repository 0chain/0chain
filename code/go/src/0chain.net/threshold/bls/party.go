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
	verifications    []VerificationKey
	SecKeyShareGroup Key
	GpSign           Sign
	ID               PartyID
}

/*MakeSimpleBLS - to create bls object */
func MakeSimpleBLS(dkg *SimpleDKG) SimpleBLS {
	bs := SimpleBLS{
		T:                dkg.T,
		N:                dkg.N,
		Msg:              " ",
		SigShare:         Sign{},
		gpPubKey:         dkg.GpPubKey,
		verifications:    nil,
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

	Logger.Info("Recover Gp Sig not done, check party.go")

	return sig

}

/*VerifyGroupSig - Verify the Gp sign with gp public key */
func (bs *SimpleBLS) VerifyGroupSig(GroupSig) bool {
	//TODO
	return true
}

/*VerifySign - Verify the BLS sign */
func (bs *SimpleBLS) VerifySign(share Sign) bool {
	//TODO
	return true
}
