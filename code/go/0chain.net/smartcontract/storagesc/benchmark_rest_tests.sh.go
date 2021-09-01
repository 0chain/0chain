package storagesc

import (
	"context"
	"net/url"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"

	cstate "0chain.net/chaincore/chain/state"
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
}

func (rbt RestBenchTest) Name() string {
	return rbt.name
}

func (bt RestBenchTest) Transaction() transaction.Transaction {
	return transaction.Transaction{}
}

func (rbt RestBenchTest) Run(balances cstate.StateContextI) {
	_, err := rbt.endpoint(context.TODO(), rbt.params, balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkRestTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuit {
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	ssc.setSC(ssc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "getConfig",
			endpoint: ssc.getConfigHandler,
		},
		{
			name:     "latestreadmarker",
			endpoint: ssc.LatestReadMarkerHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client", data.Clients[0])
				values.Set("blobber", getMockBlobberId(0))
				return values
			}(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.StorageRest, testsI}
}
