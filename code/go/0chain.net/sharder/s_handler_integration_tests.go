//go:build integration_tests
// +build integration_tests

package sharder

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"0chain.net/chaincore/chain"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config/cases"
	"0chain.net/core/common"
	"0chain.net/core/util"
	sharderEndpoint "0chain.net/sharder/endpoint"
)

// SetupX2SResponders setups sharders responders for miner and sharders.
func SetupX2SResponders() {
	handlers := x2sRespondersMap()

	handlers[sharderEndpoint.AnyServiceToSharderGetBlock] = chain.BlockStats(
		handlers[sharderEndpoint.AnyServiceToSharderGetBlock],
		chain.BlockStatsConfigurator{
			HashKey:      "hash",
			SenderHeader: node.HeaderNodeID,
		},
	)

	setupHandlers(handlers)
}

func RoundBlockRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	cfg := crpc.Client().State().FBRequestor
	if cfg == nil {
		return roundBlockRequestHandler(ctx, r)
	}

	var (
		minerInformer = createMinerInformer(r.FormValue("hash"))
		requestorID   = r.Header.Get(node.HeaderNodeID)
		selfInfo      = cases.SelfInfo{
			IsSharder: node.Self.Type == node.NodeTypeSharder,
			ID:        node.Self.ID,
			SetIndex:  node.Self.SetIndex,
		}
	)

	cfg.Lock()
	cfg.Unlock()

	switch {
	case cfg.IgnoringRequestsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo) && cfg.Ignored < 1:
		cfg.Ignored++
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

	case cfg.ValidBlockWithChangedHashBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return validBlockWithChangedHash(r)

	case cfg.InvalidBlockWithChangedHashBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return invalidBlockWithChangedHash(r)

	case cfg.BlockWithoutVerTicketsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return blockWithoutVerTickets(r)

	case cfg.CorrectResponseBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		fallthrough

	default:
		return roundBlockRequestHandler(ctx, r)
	}
}

func createMinerInformer(blockHash string) *cases.MinerInformer {
	sChain := GetSharderChain()

	bl, err := sChain.GetBlock(context.Background(), blockHash)
	if err != nil {
		return nil
	}
	miners := sChain.GetMiners(bl.Round)

	roundI := round.NewRound(bl.Round)
	roundI.SetRandomSeed(bl.RoundRandomSeed, len(miners.Nodes))

	return cases.NewMinerInformer(
		chain.NewRanker(roundI, miners),
		sChain.GetGeneratorsNum(),
	)
}

func validBlockWithChangedHash(r *http.Request) (*block.Block, error) {
	bl, err := GetSharderChain().GetBlock(context.Background(), r.FormValue("hash"))
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
	bl, err := GetSharderChain().GetBlock(context.Background(), r.FormValue("hash"))
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.Hash = util.Hash("invalid hash")
	return bl, nil
}

func blockWithoutVerTickets(r *http.Request) (*block.Block, error) {
	bl, err := GetSharderChain().GetBlock(context.Background(), r.FormValue("hash"))
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.VerificationTickets = nil

	return bl, nil
}
