package minersc

import (
	"strconv"

	benchmark "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/stakepool/spenum"
)

func BenchmarkRestTests(
	data benchmark.BenchData, _ benchmark.SignatureScheme,
) benchmark.TestSuite {
	rh := rest.NewRestHandler(&rest.TestQueryChainer{})
	mrh := NewMinerRestHandler(rh)
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "getNodepool",
				Endpoint: mrh.getNodePool,
			},
			{
				FuncName: "getUserPools",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
				Endpoint: mrh.getUserPools,
			},
			{
				FuncName: "getStakePoolStat",
				Params: map[string]string{
					"miner_id":      data.Miners[0],
					"provider_type": strconv.Itoa(int(spenum.Miner)),
				},
				Endpoint: mrh.getStakePoolStat,
			},
			{
				FuncName: "globalSettings",
				Endpoint: mrh.getGlobalSettings,
			},
			{
				FuncName: "getSharderKeepList",
				Endpoint: mrh.getSharderKeepList,
			},
			{
				FuncName: "getMinerList",
				Endpoint: mrh.getMinerList,
			},
			{
				FuncName: "get_miners_stats",
				Endpoint: mrh.getMinersStats,
			},
			{
				FuncName: "getSharderList",
				Endpoint: mrh.getSharderList,
			},
			{
				FuncName: "get_sharders_stats",
				Endpoint: mrh.getShardersStats,
			},
			{
				FuncName: "getPhase",
				Endpoint: mrh.getPhase,
			},
			{
				FuncName: "getDkgList",
				Endpoint: mrh.getDkgList,
			},
			{
				FuncName: "getMpksList",
				Endpoint: mrh.getMpksList,
			},
			{
				FuncName: "getGroupShareOrSigns",
				Endpoint: mrh.getGroupShareOrSigns,
			},
			{
				FuncName: "getEvents",
				Params: map[string]string{
					"block_number": "1",
					"type":         "2",
					"tag":          "3",
				},
				Endpoint: mrh.getEvents,
			},
			{
				FuncName: "getMagicBlock",
				Endpoint: mrh.getMagicBlock,
			},
			{
				FuncName: "nodeStat.miner",
				Params: map[string]string{
					"id": data.Miners[0],
				},
				Endpoint: mrh.getNodeStat,
			},
			{
				FuncName: "test.nodeStat.miner.delegates",
				Params: map[string]string{
					"id":                data.Miners[0],
					"include_delegates": "true",
				},
				Endpoint: mrh.testNodeStat,
			},
			{
				FuncName: "nodeStat.sharder",
				Params: map[string]string{
					"id": data.Sharders[0],
				},
				Endpoint: mrh.getNodeStat,
			},
			{
				FuncName: "test.nodeStat.sharer.delegates",
				Params: map[string]string{
					"id":                data.Sharders[0],
					"include_delegates": "true",
				},
				Endpoint: mrh.testNodeStat,
			},
			{
				FuncName: "nodePoolStat",
				Params: map[string]string{
					"id":      data.Miners[0],
					"pool_id": getMinerDelegatePoolId(0, 0, data.Clients),
				},
				Endpoint: mrh.getNodePoolStat,
			},

			{
				FuncName: "configs",
				Endpoint: mrh.getConfigs,
			},
			{
				FuncName: "provider-rewards",
				Params: map[string]string{
					"id":    data.Miners[0],
					"limit": "20",
					"start": "25",
					"end":   "25",
				},
				Endpoint: mrh.getProviderRewards,
			},
			{
				FuncName: "delegate-rewards",
				Params: map[string]string{
					"limit":  "20",
					"offset": "1",
					"start":  "25",
					"end":    "75",
				},
				Endpoint: mrh.getDelegateRewards,
			},
		},
		ADDRESS,
		mrh,
		benchmark.MinerRest,
	)
}
