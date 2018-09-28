package bls

/* BLS implementation */

type SimpleBLS struct {
	// TODO
}

func MakeSimpleBLS() SimpleBLS {
	// TODO
	return SimpleBLS{}
}

func (bs *SimpleBLS) SignMsg() interface{} {
	// TODO
	return nil
}

func (bs *SimpleBLS) VerifySign(from ID, share SignShare) bool {
	//TODO
	return true
}

func (bs *SimpleBLS) RecoverGroupSig(from []ID, shares []SignShare) interface{} {
	//TODO
	return GroupSig
}

func (bs *SimpleBLS) VerifyGroupSig(GroupSig) bool {
	//TODO
	return true
}
