package zcnsc

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/rest"
)

func BenchmarkRestTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	rh := rest.NewRestHandler(&rest.TestQueryChainer{})
	zrh := NewZcnRestHandler(rh)
	common.ConfigRateLimits()
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
			{
				FuncName: "mint_nonce",
				Params: map[string]string{
					"client_id": data.Clients[0],
				},
				Endpoint: zrh.MintNonceHandler,
			},
			{
				FuncName: "not_processed_burn_tickets",
				Params: map[string]string{
					"ethereum_address": data.Clients[0],
				},
				Endpoint: zrh.NotProcessedBurnTicketsHandler,
			},
		},
		ADDRESS,
		zrh,
		benchmark.ZCNSCBridgeRest,
	)
}
