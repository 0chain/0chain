package vestingsc

import (
	benchmark "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/rest"
)

const owner = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

func BenchmarkRestTests(
	data benchmark.BenchData, _ benchmark.SignatureScheme,
) benchmark.TestSuite {
	rh := rest.NewRestHandler(&rest.TestQueryChainer{})
	vrh := NewVestingRestHandler(rh)
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "vesting_config",
				Endpoint: vrh.getConfig,
			},
			{
				FuncName: "getPoolInfo",
				Params: map[string]string{
					"pool_id": geMockVestingPoolId(0),
				},
				Endpoint: vrh.getPoolInfo,
			},
			{
				FuncName: "getClientPools",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
				Endpoint: vrh.getClientPools,
			},
		},
		ADDRESS,
		vrh,
	)

}
