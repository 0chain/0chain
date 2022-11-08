//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
	"net/http"

	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
)

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	setupHandlers(x2mRespondersMap())
}

// SetupM2MReceivers - setup receivers for miner to miner communication.
func SetupM2MReceivers(c node.Chainer) {
	setupHandlers(x2mReceiversMap(c))
}

// NotarizationReceiptHandler - handles the receipt of a notarization
// for a block.
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	return notarizationReceiptHandler(ctx, entity)
}

// NotarizedBlockSendHandler - handles a request for a notarized block.
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return notarizedBlockSendHandler(ctx, r)
}

// VRFShareHandler - handle the vrf share.
func VRFShareHandler(ctx context.Context, entity datastore.Entity) (
	interface{}, error) {
	return vrfShareHandler(ctx, entity)
}
