//go:build !integration_tests
// +build !integration_tests

package sharder

import (
	"context"
	"net/http"
)

// SetupX2SResponders setups sharders responders for miner and sharders.
func SetupX2SResponders() {
	setupHandlers(x2sRespondersMap())
}

func RoundBlockRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return roundBlockRequestHandler(ctx, r)
}
