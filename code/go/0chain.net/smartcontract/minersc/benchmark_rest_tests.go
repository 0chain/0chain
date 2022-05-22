package minersc

import (
	benchmark "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/stakepool/spenum"
)

func BenchmarkRestTests(
	data benchmark.BenchData, _ benchmark.SignatureScheme,
) benchmark.TestSuite {
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "getNodepool",
			},
			{
				FuncName: "getUserPools",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
			},
			{
				FuncName: "globalSettings",
			},
			{
				FuncName: "getSharderKeepList",
			},
			{
				FuncName: "getMinerList",
			},
			{
				FuncName: "get_miners_stats",
			},
			{
				FuncName: "get_miners_stake",
			},
			{
				FuncName: "getSharderList",
			},
			{
				FuncName: "get_sharders_stats",
			},
			{
				FuncName: "get_sharders_stake",
			},
			{
				FuncName: "getPhase",
			},
			{
				FuncName: "getDkgList",
			},
			{
				FuncName: "getMpksList",
			},
			{
				FuncName: "getGroupShareOrSigns",
			},
			{
				FuncName: "getEvents",
				Params: map[string]string{
					"block_number": "",
				},
			},
			{
				FuncName: "getMagicBlock",
			},
			{
				FuncName: "nodeStat",
				Params: map[string]string{
					"id": GetMockNodeId(0, spenum.Miner),
				},
			},
			{
				FuncName: "nodePoolStat",
				Params: map[string]string{
					"id":      GetMockNodeId(0, spenum.Miner),
					"pool_id": getMinerDelegatePoolId(0, 0, spenum.Miner),
				},
			},

			{
				FuncName: "get_miner_geolocations",
				Params: map[string]string{
					"offset": "",
					"limit":  "",
					"active": "",
				},
			},
			{
				FuncName: "get_sharder_geolocations",
				Params: map[string]string{
					"offset": "",
					"limit":  "",
					"active": "",
				},
			},
			{
				FuncName: "configs",
			},
		},
		ADDRESS,
	)
}
