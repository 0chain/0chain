package zcnsc

import (
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/rest"
)

func BenchmarkRestTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	rh := rest.NewTestRestHandler()
	zrh := NewZcnRestHandler(rh)
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "getAuthorizerNodes",
				Endpoint: zrh.getAuthorizerNodes,
			},
			{
				FuncName: "getGlobalConfig",
				Endpoint: zrh.GetGlobalConfig,
			},
			{
				FuncName: "getAuthorizer",
				Params: map[string]string{
					"id": data.Clients[0],
				},
				Endpoint: zrh.getAuthorizer,
			},
		},
		ADDRESS,
		zrh,
	)
}
