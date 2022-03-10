package minersc

import (
	"context"
	"net/url"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/stakepool/spenum"
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
	var msc = MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	msc.setSC(msc.SmartContract, &smartcontract.BCContext{})
	var tests = []RestBenchTest{
		{
			name:     "miner_rest.getNodepool",
			endpoint: msc.GetNodepoolHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("id", GetMockNodeId(0, spenum.Miner))
				values.Set("n2n_host", "n2n_host")
				return values
			}(),
		},
		{
			name:     "miner_rest.getUserPools",
			endpoint: msc.GetUserPoolsHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("client_id", data.Clients[0])
				return values
			}(),
		},
		{
			name:     "miner_rest.globals",
			endpoint: msc.getGlobalsHandler,
		},
		{
			name:     "miner_rest.getSharderKeepList",
			endpoint: msc.GetSharderKeepListHandler,
		},
		{
			name:     "miner_rest.getMinerList",
			endpoint: msc.GetMinerListHandler,
		},
		{
			name:     "miner_rest.getSharderList",
			endpoint: msc.GetSharderListHandler,
		},
		{
			name:     "miner_rest.getPhase",
			endpoint: msc.GetPhaseHandler,
		},
		{
			name:     "miner_rest.getDkgList",
			endpoint: msc.GetDKGMinerListHandler,
		},
		{
			name:     "miner_rest.getMpksList",
			endpoint: msc.GetMinersMpksListHandler,
		},
		{
			name:     "miner_rest.getGroupShareOrSigns",
			endpoint: msc.GetGroupShareOrSignsHandler,
		},
		{
			name:     "miner_rest.getMagicBlock",
			endpoint: msc.GetMagicBlockHandler,
		},
		{
			name:     "miner_rest.nodeStat",
			endpoint: msc.nodeStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("id", GetMockNodeId(0, spenum.Miner))
				return values
			}(),
		},
		{
			name:     "miner_rest.nodePoolStat",
			endpoint: msc.nodePoolStatHandler,
			params: func() url.Values {
				var values url.Values = make(map[string][]string)
				values.Set("id", GetMockNodeId(0, spenum.Miner))
				values.Set("pool_id", getMinerDelegatePoolId(0, 0, spenum.Miner))
				return values
			}(),
		},
		{
			name:     "miner_rest.configs",
			endpoint: msc.configHandler,
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.MinerRest,
		Benchmarks: testsI,
	}
}
