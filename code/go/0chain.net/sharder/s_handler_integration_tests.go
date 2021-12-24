//go:build integration_tests
// +build integration_tests

package sharder

import (
	"0chain.net/chaincore/node"
	"0chain.net/conductor/conductrpc/stats/middleware"
)

// SetupX2SResponders setups sharders responders for miner and sharders.
func SetupX2SResponders() {
	handlers := x2sRespondersMap()

	handlers[getBlockX2SV1Pattern] = middleware.BlockStats(
		handlers[getBlockX2SV1Pattern],
		middleware.BlockStatsConfigurator{
			HashKey:      "hash",
			Handler:      getBlockX2SV1Pattern,
			SenderHeader: node.HeaderNodeID,
		},
	)

	setupHandlers(x2sRespondersMap())
}
