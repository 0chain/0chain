package bls

import "github.com/herumi/bls/ffi/go/bls"

/*Key - Is of type mcl.SecretKey*/
type Key = bls.SecretKey

/*PartyID - Is of type bls.ID*/
type PartyID = bls.ID

/*Message - Is of type string*/
type Message = string

/*Sign - Is of type bls.Sign*/
type Sign = bls.Sign

type PublicKey = bls.PublicKey

/*DKGI - Interface for DKG*/
type DKGI interface {
	ComputeDKGKeyShare(forID PartyID) (Key, error)
	GetKeyShareForOther(to PartyID) *DKGKeyShare
	AggregateSecretKeyShares(qual []PartyID)
}
