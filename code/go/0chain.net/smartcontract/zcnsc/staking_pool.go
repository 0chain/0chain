package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

func (zcn *ZCNSmartContract) DistributeRewards(t *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (string, error) {
	return "", nil
}

func (zcn *ZCNSmartContract) AddToDelegatePool(t *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (string, error) {
	return "", nil
}

func (zcn *ZCNSmartContract) DeleteFromDelegatePool(t *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (string, error) {
	return "", nil
}
