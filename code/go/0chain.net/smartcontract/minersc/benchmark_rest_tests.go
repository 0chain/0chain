package minersc

import (
	"context"
	"net/url"

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
		//	panic(err)
	}
}

func BenchmarkRestTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuit {
	var msc = MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	msc.setSC(msc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "getNodepool",
			endpoint: msc.GetNodepoolHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("id", GetMockNodeId(0, NodeTypeMiner))
				values.Set("n2n_host", "n2n_host")
				return values
			}(),
		},
		{
			name:     "getUserPools",
			endpoint: msc.GetUserPoolsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "getMinerList",
			endpoint: msc.GetMinerListHandler,
		},
		{
			name:     "getSharderList",
			endpoint: msc.GetSharderListHandler,
		},
		{
			name:     "getPhase",
			endpoint: msc.GetPhaseHandler,
		},
		{
			name:     "getDkgList",
			endpoint: msc.GetDKGMinerListHandler,
		},
		{
			name:     "getMpksList",
			endpoint: msc.GetMinersMpksListHandler,
		},
		{
			name:     "getGroupShareOrSigns",
			endpoint: msc.GetGroupShareOrSignsHandler,
		},
		{
			name:     "getMagicBlock",
			endpoint: msc.GetMagicBlockHandler,
		},
		{
			name:     "nodeStat",
			endpoint: msc.nodeStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("id", GetMockNodeId(0, NodeTypeMiner))
				return values
			}(),
		},
		{
			name:     "nodePoolStat",
			endpoint: msc.nodePoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("id", GetMockNodeId(0, NodeTypeMiner))
				values.Set("pool_id", getMinerDelegatePoolId(0, 0, NodeTypeMiner))
				return values
			}(),
		},
		{
			name:     "configs",
			endpoint: msc.configsHandler,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.MinerRest, testsI}
}
