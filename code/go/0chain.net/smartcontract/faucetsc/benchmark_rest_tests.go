package faucetsc

import (
	"context"
	"net/url"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
)

type RestBenchTest struct {
	name     string
	endpoint func(
		context.Context,
		url.Values,
		cstate.StateContextI,
	) (interface{}, error)
	params url.Values
	error
}

func (rbt RestBenchTest) Error() error {
	return rbt.error
}

func (rbt RestBenchTest) Name() string {
	return rbt.name
}

func (rbt RestBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (rbt RestBenchTest) Run(balances cstate.StateContextI, _ *testing.B) {
	_, rbt.error = rbt.endpoint(context.TODO(), rbt.params, balances)
}

func BenchmarkRestTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var fsc = FaucetSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	fsc.setSC(fsc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "faucet_rest.personalPeriodicLimit",
			endpoint: fsc.personalPeriodicLimit,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "faucet_rest.globalPeriodicLimit",
			endpoint: fsc.globalPeriodicLimit,
		},
		{
			name:     "faucet_rest.pourAmount",
			endpoint: fsc.pourAmount,
		},
		{
			name:     "faucet_rest.getConfig",
			endpoint: fsc.getConfigHandler,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.FaucetRest,
		Benchmarks: testsI,
	}
}
