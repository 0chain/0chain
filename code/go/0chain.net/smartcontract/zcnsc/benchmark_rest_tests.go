package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
	"context"
	"net/url"
	"testing"
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

func (bt restBenchTest) Run(balances cstate.StateContextI, _ *testing.B) {
	_, err := bt.endpoint(context.TODO(), bt.params, balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkRestTests(
	_ bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {

	sc := createSmartContract()

	return createSuite([]restBenchTest{
		{
			name:     "zcnsc_rest.getAuthorizerNodes",
			endpoint: sc.getAuthorizerNodes,
		},
	})
}

func createSmartContract() ZCNSmartContract {
	sc := ZCNSmartContract{
		SmartContract: smartcontractinterface.NewSC(ADDRESS),
	}

	sc.setSC(sc.SmartContract, &smartcontract.BCContext{})
	return sc
}

func createSuite(restTests []restBenchTest) bk.TestSuite {
	var tests []bk.BenchTestI

	for _, test := range restTests {
		tests = append(tests, test)
	}

	return bk.TestSuite{
		Source:     bk.ZCNSCBridgeRest,
		Benchmarks: tests,
	}
}
