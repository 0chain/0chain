package bls

import (
	"fmt"
)

/* BLS implementation */

type SimpleBLS struct {
	T                int
	N                int
	Msg              Message
	SigShare         Sign
	gpPubKey         GroupPublicKey
	verifications    []VerificationKey
	SecKeyShareGroup Key
	GpSign           Sign
	ID               PartyId
}

func MakeSimpleBLS(dkg *BLSSimpleDKG) SimpleBLS {
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

func (bs *SimpleBLS) SignMsg() Sign {

	aggSecKey := bs.SecKeyShareGroup
	sigShare := *aggSecKey.Sign(bs.Msg)
	return sigShare
}

func (bs *SimpleBLS) RecoverGroupSig(from []PartyId, shares []Sign) (Sign, error) {

	signVec := shares
	idVec := from

	var sig Sign
	err := sig.Recover(signVec, idVec)

	if err != nil {
		fmt.Println("Recover Gp Sig not done")
		return sig, nil
	}
	bs.GpSign = sig
	return sig, nil

}

func (bs *SimpleBLS) VerifyGroupSig(GroupSig) bool {
	//TODO
	return true
}
