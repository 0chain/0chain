/* BLS interface */
package bls

import (
	"github.com/pmer/gobls"
)

type PartyId = gobls.ID
type GroupPublicKey = gobls.PublicKey
type Message = string
type Sign = gobls.Sign

type SignShare interface{}

type GroupSig interface{}

type Party interface {
	SignMsg() Sign
	VerifySign(share Sign) bool
	RecoverGroupSig(from []PartyId, shares []SignShare) Sign
	VerifyGroupSig(GroupSig) bool
}

/* AfterDKGKeyShare - Denotes the share of each party after DKG, ie, after aggregating the shares */
type AfterDKGKeyShare struct {
	m Key
	v VerificationKey
}
