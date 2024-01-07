//go:build integration_tests
// +build integration_tests

package zcnsc

import (
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
	cstate "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/logging"
)

func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (resp string, err error) {
	logging.Logger.Debug("mint from CT")
	return zcn.mint(trans, inputData, crpc.Client().State().RoundRandomSeed.RandomSeed, ctx)
}
