//go:build !integration_tests
// +build !integration_tests

package zcnsc

import (
	"0chain.net/chaincore/transaction"
	cstate "0chain.net/smartcontract/common"
)

func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (resp string, err error) {
	return zcn.mint(trans, inputData, ctx.GetBlock().GetRoundRandomSeed(), ctx)
}
