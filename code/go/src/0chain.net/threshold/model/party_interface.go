/* BLS interface */

package model


type ID int

type SignShare interface {}

type GroupSig interface {}

type Party interface {

func SignMsg() SignShare
func VerifySign(from ID, share SignShare) bool
func RecoverGroupSig(from []ID, shares []SignShare) GroupSig
func VerifyGroupSig(GroupSig) bool

}


