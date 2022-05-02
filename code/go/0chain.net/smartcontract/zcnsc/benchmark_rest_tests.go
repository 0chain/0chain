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

func BenchmarkRestTests(_ benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	sc := createSmartContract()

	return createRestTestSuite(
		[]restBenchTest{
			// todo add tests for
			// getGlobalConfig
			// getAuthorizer
			{
				name:     "zcnsc_rest.getAuthorizerNodes",
				endpoint: sc.GetAuthorizerNodes,
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
