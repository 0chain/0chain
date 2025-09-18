package vestingsc

import (
	benchmark "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/rest"
)

const owner = "c4a3573c7f7c2e31210c1988d49ee2b6c2009fe67613457904ea7109cae1f4c9" //nolint:unused

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
		benchmark.VestingRest,
	)

}
