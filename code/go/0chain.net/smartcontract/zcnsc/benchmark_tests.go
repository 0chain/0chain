package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
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
	// TODO: Add name here
	return ""
}

func (bt benchTest) Transaction() *transaction.Transaction {
	return nil
}

func (bt benchTest) Run(_ cstate.StateContextI, _ *testing.B) {
	// TODO: Create smart contract here
}

func BenchmarkTests(
	_ bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var testsI []bk.BenchTestI

	var tests []benchTest

	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.ZCNSCBridge,
		Benchmarks: testsI,
	}
}
