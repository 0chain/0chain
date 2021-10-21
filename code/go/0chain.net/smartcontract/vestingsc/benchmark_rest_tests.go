package vestingsc

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
	error  string
}

func (rbt RestBenchTest) Error() string {
	return rbt.error
}

func (rbt RestBenchTest) Name() string {
	return rbt.name
}

func (rbt RestBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (rbt RestBenchTest) Run(balances cstate.StateContextI, _ *testing.B) error {
	_, err := rbt.endpoint(context.TODO(), rbt.params, balances)
	return err
}

func BenchmarkRestTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var vsc = VestingSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	vsc.setSC(vsc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "vesting_rest.getConfig",
			endpoint: vsc.getConfigHandler,
		},
		{
			name:     "vesting_rest.getPoolInfo",
			endpoint: vsc.getPoolInfoHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("pool_id", geMockVestingPoolId(0))
				return values
			}(),
		},
		{
			name:     "vesting_rest.getClientPools",
			endpoint: vsc.getClientPoolsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.VestingRest,
		Benchmarks: testsI,
	}
}
