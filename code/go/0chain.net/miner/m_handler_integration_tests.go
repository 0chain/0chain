//go:build integration_tests
// +build integration_tests

package miner

import (
	"0chain.net/chaincore/node"
	"0chain.net/conductor/conductrpc/stats/middleware"
)

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	handlers := x2mRespondersMap()
	handlers[getNotarizedBlockX2MV1Pattern] = middleware.BlockStats(
		handlers[getNotarizedBlockX2MV1Pattern],
		middleware.BlockStatsConfigurator{
			HashKey:      "block",
			Handler:      getNotarizedBlockX2MV1Pattern,
			SenderHeader: node.HeaderNodeID,
		},
	)
	setupHandlers(handlers)
}
