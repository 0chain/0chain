package storagesc

import (
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"testing"
)

func TestFreeAllocationRequest(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var txn = &transaction.Transaction{}
			var balances = &mocks.StateContextI{}
			var ssc = &StorageSmartContract{
				SmartContract: sci.NewSC(ADDRESS),
			}
			var input = []byte{}
			ssc.freeAllocationRequest(txn, input, balances)
		})
	}
}
