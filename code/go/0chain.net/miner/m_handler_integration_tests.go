//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"log"

	"0chain.net/chaincore/node"
	"0chain.net/conductor/conductrpc/stats/middleware"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
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

// SetupM2MReceivers - setup receivers for miner to miner communication.
func SetupM2MReceivers(c node.Chainer) {
	handlers := x2mReceiversMap(c)
	handlers[vrfsShareRoundM2MV1Pattern] = common.N2NRateLimit(
		node.ToN2NReceiveEntityHandler(
			middleware.VRFSStats(VRFShareHandler),
			nil,
		),
	)
	setupHandlers(handlers)
}

// NotarizationReceiptHandler - handles the receipt of a notarization
// for a block.
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	not, ok := entity.(*Notarization)
	if !ok {
		log.Panicf("unexpected type")
	}

	if isDelayingBlock(not.Round) {
		go func() {
			for bl := range delayedBlock {
				GetMinerChain().sendBlock(context.Background(), bl)
				close(delayedBlock)
			}
		}()
	}

	return notarizationReceiptHandler(ctx, entity)
}
