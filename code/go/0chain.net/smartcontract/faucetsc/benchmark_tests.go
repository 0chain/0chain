package faucetsc

import (
	"testing"

	sc "0chain.net/smartcontract"
	"github.com/stretchr/testify/require"

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

func (bt BenchTest) Run(balances cstate.StateContextI, b *testing.B) {
	var fsc = FaucetSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	fsc.setSC(fsc.SmartContract, &smartcontract.BCContext{})
	gn, err := fsc.getGlobalVariables(bt.txn, balances)
	require.NoError(b, err)
	switch bt.endpoint {
	case "updateSettings":
		_, err = fsc.updateSettings(bt.Transaction(), bt.input, balances, gn)
	case "pour":
		_, err = fsc.pour(bt.Transaction(), bt.input, balances, gn)
	case "refill":
		_, _ = fsc.refill(bt.Transaction(), balances, gn)
	default:
		require.Fail(b, "unknown endpoint"+bt.endpoint)
	}
	require.NoError(b, err)
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuit {
	var tests = []BenchTest{
		{
			name:     "faucet.update-settings",
			endpoint: "updateSettings",
			txn: &transaction.Transaction{
				Value: 3,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					Settings[PourAmount]:      "1",
					Settings[MaxPourAmount]:   "2",
					Settings[PeriodicLimit]:   "3",
					Settings[GlobalLimit]:     "5",
					Settings[IndividualReset]: "7s",
					Settings[GlobalReset]:     "11m",
				},
			}).Encode(),
		},
		{
			name:     "faucet.pour",
			endpoint: "pour",
			txn: &transaction.Transaction{
				Value: 3,
			},
			input: nil,
		},
		{
			name:     "faucet.refill",
			endpoint: "refill",
			txn: &transaction.Transaction{
				Value:      23,
				ToClientID: ADDRESS,
			},
			input: nil,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		test.txn.ClientID = data.Clients[0]
		testsI = append(testsI, test)
	}
	return bk.TestSuit{
		Source:     bk.Faucet,
		Benchmarks: testsI,
	}
}
