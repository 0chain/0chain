//go:build !integration_tests
// +build !integration_tests

package miner

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	setupHandlers(x2mRespondersMap())
}

// NotarizationReceiptHandler - handles the receipt of a notarization
// for a block.
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	return notarizationReceiptHandler(ctx, entity)
}
