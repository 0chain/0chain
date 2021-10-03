package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/benchmark"
	"testing"
)

type benchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		cstate.StateContextI,
	) (string, error)
	txn   *transaction.Transaction
	input []byte
}

func (bt benchTest) Name() string {
	return bt.name
}

func (bt benchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (bt benchTest) Run(_ cstate.StateContextI, _ *testing.B) {
	// TODO: Complete
}

func BenchmarkTests(_ benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	return createTestSuite(
		[]benchTest{
			{
				name:     "zcnsc_rest.getAuthorizerNodes",
				endpoint: nil,
			},
		},
	)
}

func createTestSuite(restTests []benchTest) benchmark.TestSuite {
	var tests []benchmark.BenchTestI

	for _, test := range restTests {
		tests = append(tests, test)
	}

	return benchmark.TestSuite{
		Source:     benchmark.ZCNSCBridge,
		Benchmarks: tests,
	}
}
