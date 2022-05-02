package zcnsc

import (
	"context"
	"net/url"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/benchmark"
)

type restBenchTest struct {
	name     string
	endpoint smartcontractinterface.SmartContractRestHandler
	params   url.Values
}

func (bt restBenchTest) Name() string {
	return bt.name
}

func (bt restBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (bt restBenchTest) Run(balances cstate.StateContextI, _ *testing.B) error {
	_, err := bt.endpoint(context.TODO(), bt.params, balances)
	return err
}

func BenchmarkRestTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	sc := createSmartContract()

	return createRestTestSuite(
		[]restBenchTest{
			{
				name:     "zcnsc_rest.getAuthorizerNodes",
				endpoint: sc.GetAuthorizerNodes,
			},
			{
				name:     "zcnsc_rest.getGlobalConfig",
				endpoint: sc.GetGlobalConfig,
			},
			{
				name:     "zcnsc_rest.getAuthorizer",
				endpoint: sc.GetAuthorizer,
				params: func() url.Values {
					var values url.Values = make(map[string][]string)
					values.Set("id", data.Clients[0])
					return values
				}(),
			},
		},
	)
}

func createRestTestSuite(restTests []restBenchTest) benchmark.TestSuite {
	var tests []benchmark.BenchTestI

	for _, test := range restTests {
		tests = append(tests, test)
	}

	return benchmark.TestSuite{
		Source:     benchmark.ZCNSCBridgeRest,
		Benchmarks: tests,
	}
}
