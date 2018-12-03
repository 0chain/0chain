package bls

import "github.com/herumi/bls/ffi/go/bls"

/*PartyID - Is of type bls.ID*/
type PartyID = bls.ID

/*GroupPublicKey - Is of type bls.PublicKey*/
type GroupPublicKey = bls.PublicKey

/*Message - Is of type string*/
type Message = string

/*Sign - Is of type bls.Sign*/
type Sign = bls.Sign

/*PartyI - Interface for BLS*/
type PartyI interface {
	SignMsg() Sign
	VerifyGroupSignShare(grpSignShare Sign, fromID PartyID) bool
	RecoverGroupSig(from []PartyID, shares []Sign) Sign
}
