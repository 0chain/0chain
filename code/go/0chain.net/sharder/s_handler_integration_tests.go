//go:build integration_tests
// +build integration_tests

package sharder

import (
	"0chain.net/conductor/conductrpc/stats/middleware"
)

// SetupX2SResponders setups sharders responders for miner and sharders.
func SetupX2SResponders() {
	handlers := x2sRespondersMap()

	handlers[getBlockX2SV1Pattern] = middleware.BlockStatsMiddleware(
		handlers[getBlockX2SV1Pattern],
		"hash",
		getBlockX2SV1Pattern,
	)

	setupHandlers(x2sRespondersMap())
}
