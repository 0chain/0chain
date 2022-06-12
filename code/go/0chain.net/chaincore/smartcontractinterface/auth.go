package smartcontractinterface

import (
	"0chain.net/core/common"
)

func AuthorizeWithOwner(funcName string, hasAccess func() (bool, error)) error {
	has_access, err := hasAccess()
	if err != nil {
		return err
	}
	if !has_access {
		return common.NewError(funcName,
			"unauthorized access - only the owner can access")
	}
	return nil
}
