//go:build !integration_tests
// +build !integration_tests

package sharder

// SetupX2SResponders setups sharders responders for miner and sharders.
func SetupX2SResponders() {
	setupHandlers(x2sRespondersMap())
}
