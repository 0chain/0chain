package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
	"context"
	"net/url"
	"testing"
)

type restBenchTest struct {
	name     string
	endpoint func(
		context.Context,
		url.Values,
		cstate.StateContextI,
	) (interface{}, error)
	params url.Values
}

func (bt restBenchTest) Name() string {
	return ""
}

func (bt restBenchTest) Transaction() *transaction.Transaction {
	return nil
}

func (bt restBenchTest) Run(_ cstate.StateContextI, _ *testing.B) {
}

func BenchmarkRestTests(
	_ bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var testsI []bk.BenchTestI

	var tests []restBenchTest

	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.ZCNSCBridgeRest,
		Benchmarks: testsI,
	}
}
