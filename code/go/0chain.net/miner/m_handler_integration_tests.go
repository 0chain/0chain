//go:build integration_tests
// +build integration_tests

package miner

import (
	"0chain.net/conductor/conductrpc/stats/middleware"
)

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	handlers := x2mRespondersMap()
	handlers[getNotarizedBlockX2MV1Pattern] = middleware.BlockStatsMiddleware(
		handlers[getNotarizedBlockX2MV1Pattern],
		"block",
		getNotarizedBlockX2MV1Pattern,
	)
	setupHandlers(handlers)
}
