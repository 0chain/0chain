package faucetsc

import (
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	sc "0chain.net/smartcontract"
	bk "0chain.net/smartcontract/benchmark"
)

const owner = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

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

func (bt BenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
	var fsc = FaucetSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	fsc.setSC(fsc.SmartContract, &smartcontract.BCContext{})
	gn, err := fsc.getGlobalVariables(bt.txn, balances)
	if err != nil {
		return err
	}
	switch bt.endpoint {
	case "updateSettings":
		_, err = fsc.updateSettings(bt.Transaction(), bt.input, balances, gn)
	case "pour":
		_, err = fsc.pour(bt.Transaction(), bt.input, balances, gn)
	case "refill":
		_, err = fsc.refill(bt.Transaction(), balances, gn)
	default:
		b.Errorf("unknown endpoint" + bt.endpoint)
	}

	return err
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var tests = []BenchTest{
		{
			name:     "faucet.update-settings",
			endpoint: "updateSettings",
			txn: &transaction.Transaction{
				ClientID: owner,
				Value:    3,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					Settings[PourAmount]:      "1",
					Settings[MaxPourAmount]:   "2",
					Settings[PeriodicLimit]:   "3",
					Settings[GlobalLimit]:     "5",
					Settings[IndividualReset]: "7s",
					Settings[GlobalReset]:     "11m",
					Settings[OwnerId]:         owner,
				},
			}).Encode(),
		},
		{
			name:     "faucet.pour",
			endpoint: "pour",
			txn: &transaction.Transaction{
				ClientID: data.Clients[0],
				Value:    3,
			},
			input: nil,
		},
		{
			name:     "faucet.refill",
			endpoint: "refill",
			txn: &transaction.Transaction{
				ClientID:   data.Clients[0],
				Value:      23,
				ToClientID: ADDRESS,
			},
			input: nil,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.Faucet,
		Benchmarks: testsI,
	}
}
