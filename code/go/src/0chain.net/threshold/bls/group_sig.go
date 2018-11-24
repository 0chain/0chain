package bls

import (
	"0chain.net/encryption"
	. "0chain.net/logging"
	"go.uber.org/zap"
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

	Logger.Error("Recover Gp Sig not done", zap.Error(err))

	return sig

}

// CalcRandomBeacon - Calculates the random beacon output
func (bs *SimpleBLS) CalcRandomBeacon(recSig []string, recIDs []string) string {

	Logger.Debug("Threshold number of bls sig shares are received ...")
	bs.CalBlsGpSign(recSig, recIDs)
	rboOutput := encryption.Hash(bs.GpSign.GetHexString())
	return rboOutput
}

// CalBlsGpSign - The function calls the RecoverGroupSig function which calculates the Gp Sign
func (bs *SimpleBLS) CalBlsGpSign(recSig []string, recIDs []string) {
	//Move this to group_sig.go
	signVec := make([]Sign, 0)
	var signShare Sign

	for i := 0; i < len(recSig); i++ {
		err := signShare.SetHexString(recSig[i])

		if err == nil {
			signVec = append(signVec, signShare)
		} else {
			Logger.Error("signVec not computed correctly", zap.Error(err))
		}
	}

	idVec := make([]PartyID, 0)
	var forID PartyID
	for i := 0; i < len(recIDs); i++ {
		err := forID.SetDecString(recIDs[i])
		if err == nil {
			idVec = append(idVec, forID)
		}
	}
	/*
		Logger.Debug("Printing bls shares and respective party IDs who sent used for computing the Gp Sign")

		for _, sig := range signVec {
			Logger.Debug(" Printing bls shares", zap.Any("sig_shares", sig.GetHexString()))

		}
		for _, fromParty := range idVec {
			Logger.Debug(" Printing party IDs", zap.Any("from_party", fromParty.GetHexString()))

		}
	*/
	bs.RecoverGroupSig(idVec, signVec)

}
