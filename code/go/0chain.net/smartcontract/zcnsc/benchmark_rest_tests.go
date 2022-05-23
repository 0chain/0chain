package zcnsc

import (
	"0chain.net/smartcontract/benchmark"
)

func BenchmarkRestTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "getAuthorizerNodes",
			},
			{
				FuncName: "getGlobalConfig",
			},
			{
				FuncName: "getAuthorizer",
				Params: map[string]string{
					"id": data.Clients[0],
				},
			},
		},
		ADDRESS,
	)
}
