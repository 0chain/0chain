/* BLS interface */

package model

type ID int

type SignShare interface{}

type GroupSig interface{}

type Party interface {
	SignMsg() SignShare
	VerifySign(from ID, share SignShare) bool
	RecoverGroupSig(from []ID, shares []SignShare) GroupSig
	VerifyGroupSig(GroupSig) bool
}
