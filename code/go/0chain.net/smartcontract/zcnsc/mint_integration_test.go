package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
)

func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (resp string, err error) {
	return zcn.mint(trans, inputData, crpc.Client().State().RoundRandomSeed.RandomSeed, ctx)
}
