//go:build !integration_tests
// +build !integration_tests

package miner

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	setupHandlers(x2mRespondersMap())
}
