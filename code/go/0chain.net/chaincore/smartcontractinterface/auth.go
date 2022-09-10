package smartcontractinterface

import (
	"0chain.net/core/common"
)

func AuthorizeWithOwner(funcName string, hasAccess func() bool) error {
	if !hasAccess() {
		return common.NewError(funcName,
			"unauthorized access - only the owner can access")
	}
	return nil
}

func AuthorizeWithDelegate(funcName string, hasAccess func() bool) error {
	if !hasAccess() {
		return common.NewError(funcName,
			"unauthorized access - only managing wallet can access")
	}
	return nil
}
