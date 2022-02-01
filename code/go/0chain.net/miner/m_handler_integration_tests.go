//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats/middleware"
	"0chain.net/conductor/config/cases"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
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

// NotarizedBlockSendHandler - handles a request for a notarized block.
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	cfg := crpc.Client().State().MinerNotarisedBlockRequestor
	if cfg == nil {
		return notarizedBlockSendHandler(ctx, r)
	}

	minerInformer := createMinerInformer(r)
	requestorID := r.Header.Get(node.HeaderNodeID)

	cfg.Lock()
	cfg.Unlock()

	switch {
	case cfg.IgnoringRequestsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound) && cfg.Ignored < 1:
		cfg.Ignored++
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

	case cfg.ValidBlockWithChangedHashBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return validBlockWithChangedHash(r)

	case cfg.InvalidBlockWithChangedHashBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return invalidBlockWithChangedHash(r)

	case cfg.BlockWithoutVerTicketsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return blockWithoutVerTickets(r)

	case cfg.CorrectResponseBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		fallthrough

	default:
		return notarizedBlockSendHandler(ctx, r)
	}
}

func createMinerInformer(r *http.Request) cases.MinerInformer {
	mChain := GetMinerChain()
	bl, err := getNotarizedBlock(context.Background(), r)
	if err != nil {
		return nil
	}
	miners := mChain.GetMiners(bl.Round)

	roundI := round.NewRound(bl.Round)
	roundI.SetRandomSeed(bl.RoundRandomSeed, len(miners.Nodes))

	return cases.NewMinerInformer(roundI, miners, mChain.GetGeneratorsNum())
}

func validBlockWithChangedHash(r *http.Request) (*block.Block, error) {
	bl, err := getNotarizedBlock(context.Background(), r)
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.CreationDate++
	bl.HashBlock()
	if bl.MinerID != node.Self.ID {
		log.Printf("miner id is unexpected, block miner %s, self %s", bl.MinerID, node.Self.ID)
	}
	if bl.Signature, err = node.Self.Sign(bl.Hash); err != nil {
		log.Panicf("Conductor: error while signing block: %v", err)
	}
	return bl, nil
}

func invalidBlockWithChangedHash(r *http.Request) (*block.Block, error) {
	bl, err := getNotarizedBlock(context.TODO(), r)
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.Hash = util.Hash("invalid hash")
	return bl, nil
}

func blockWithoutVerTickets(r *http.Request) (*block.Block, error) {
	bl, err := getNotarizedBlock(context.TODO(), r)
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.VerificationTickets = nil

	return bl, nil
}
