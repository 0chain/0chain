//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"log"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	handlers := x2mRespondersMap()
	handlers[getNotarizedBlockX2MV1Pattern] = chain.BlockStats(
		handlers[getNotarizedBlockX2MV1Pattern],
		chain.BlockStatsConfigurator{
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
			VRFSStats(VRFShareHandler),
			nil,
		),
	)
	setupHandlers(handlers)
}

// VRFSStats represents middleware for datastore.JSONEntityReqResponderF handlers.
// Collects vrfs requests stats.
func VRFSStats(handler datastore.JSONEntityReqResponderF) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		if !crpc.Client().State().ServerStatsCollectorEnabled {
			return handler(ctx, entity)
		}

		vrfs, ok := entity.(*round.VRFShare)
		if !ok {
			log.Panicf("Conductor: unexpected entity type is provided")
		}

		ss := &stats.VRFSRequest{
			NodeID:   node.Self.ID,
			Round:    vrfs.Round,
			SenderID: node.GetSender(ctx).GetKey(),
		}
		if err := crpc.Client().AddVRFSServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		return handler(ctx, entity)
	}
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
