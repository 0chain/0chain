package smartcontractinterface

import "0chain.net/core/common"

type Owned struct {
	owner string
}

func (o *Owned) Owner() string {
	return o.owner
}

func NewOwned(owner string) *Owned {
	if len(owner) == 0 {
		panic("owner must be set")
	}
	return &Owned{owner: owner}
}

func (o *Owned) Authorize(userId, funcName string) error {
	if o.owner != userId {
		return common.NewError(funcName,
			"unauthorized access - only the owner can access")
	}
	return nil
}
