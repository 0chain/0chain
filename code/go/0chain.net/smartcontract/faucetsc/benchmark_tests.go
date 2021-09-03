package faucetsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	bk "0chain.net/smartcontract/benchmark"
)

type BenchTest struct {
	name     string
	endpoint string
	txn      *transaction.Transaction
	input    []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: bt.txn.Hash,
		},
		ClientID:     bt.txn.ClientID,
		ToClientID:   bt.txn.ToClientID,
		Value:        bt.txn.Value,
		CreationDate: bt.txn.CreationDate,
	}
}

func (bt BenchTest) Run(balances cstate.StateContextI) {
	var fsc = FaucetSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	fsc.setSC(fsc.SmartContract, &smartcontract.BCContext{})
	gn := fsc.getGlobalVariables(bt.txn, balances)
	var err error
	switch bt.endpoint {
	case "updateLimits":
		_, err = fsc.updateLimits(bt.Transaction(), bt.input, balances, gn)
	case "pour":
		_, err = fsc.pour(bt.Transaction(), bt.input, balances, gn)
	case "refill":
		_, _ = fsc.refill(bt.Transaction(), balances, gn)
	default:
		panic("unknown endpoint: " + bt.endpoint)
	}
	if err != nil {
		panic(err)
	}
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuit {
	var tests = []BenchTest{
		// todo updateLimits waiting for Pr 484
		{
			name:     "faucet.pour",
			endpoint: "pour",
			txn: &transaction.Transaction{
				Value:    3,
				ClientID: data.Clients[0],
			},
			input: nil,
		},
		{
			name:     "faucet.refill",
			endpoint: "refill",
			txn: &transaction.Transaction{
				Value:      23,
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: nil,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{
		Source:     bk.Faucet,
		Benchmarks: testsI,
	}
}
