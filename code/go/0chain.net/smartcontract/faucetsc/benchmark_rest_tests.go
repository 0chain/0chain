package faucetsc

import (
	benchmark "0chain.net/smartcontract/benchmark"
)

func BenchmarkRestTests(
	data benchmark.BenchData, _ benchmark.SignatureScheme,
) benchmark.TestSuite {
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "personalPeriodicLimit",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
			},
			{
				FuncName: "globalPeriodicLimit",
			},
			{
				FuncName: "pourAmount",
			},
			{
				FuncName: "getConfig",
			},
		},
		ADDRESS,
	)
}
