//go:build integration_tests
// +build integration_tests

package chain

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
)

func SetupX2MRequestors() {
	setupX2MRequestors()

	if crpc.Client().State().ClientStatsCollectorEnabled {
		BlockStateChangeRequestor = middleware.BlockStateChangeRequestor(BlockStateChangeRequestor)
		MinerNotarizedBlockRequestor = middleware.MinerNotarisedBlockRequestor(MinerNotarizedBlockRequestor)
	}
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	cfg := crpc.Client().State().BlockStateChangeRequestor
	if cfg == nil {
		return c.blockStateChangeHandler(ctx, r)
	}

	minerInformer := createMinerInformer(r)
	requestorID := r.Header.Get(node.HeaderNodeID)

	cfg.Lock()
	defer cfg.Unlock()

	switch {
	case cfg.IgnoringRequestsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound) && cfg.Ignored < 1:
		cfg.Ignored++
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

	case cfg.ChangedMPTNodeBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return changeMPTNode(ctx, r)

	case cfg.DeletedMPTNodeBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return deleteMPTNode(ctx, r)

	case cfg.AddedMPTNodeBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return addMPTNode(ctx, r)

	case cfg.PartialStateFromAnotherBlockBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		return changePartialState(ctx, r)

	case cfg.CorrectResponseBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound):
		fallthrough

	default:
		return c.blockStateChangeHandler(ctx, r)
	}
}

func createMinerInformer(r *http.Request) cases.MinerInformer {
	sChain := GetServerChain()
	bl, err := sChain.getNotarizedBlock(context.Background(), r.FormValue("round"), r.FormValue("block"))
	if err != nil {
		return nil
	}
	miners := sChain.GetMiners(bl.Round)

	roundI := round.NewRound(bl.Round)
	roundI.SetRandomSeed(bl.RoundRandomSeed, len(miners.Nodes))

	return cases.NewMinerInformer(roundI, miners, sChain.GetGeneratorsNum())
}

func changeMPTNode(ctx context.Context, r *http.Request) (*block.StateChange, error) {
	stChange, err := GetServerChain().blockStateChangeHandler(ctx, r)
	if err != nil {
		return nil, err
	}

	if len(stChange.Nodes) == 0 {
		log.Panicf("Conductor: mpt is empty")
	}

	stChange.Nodes[len(stChange.Nodes)-1] = stChange.Nodes[0].Clone()
	return stChange, nil
}

func deleteMPTNode(ctx context.Context, r *http.Request) (*block.StateChange, error) {
	stChange, err := GetServerChain().blockStateChangeHandler(ctx, r)
	if err != nil {
		return nil, err
	}

	if len(stChange.Nodes) == 0 {
		log.Panicf("Conductor: mpt is empty")
	}

	stChange.Nodes = stChange.Nodes[:len(stChange.Nodes)-2]
	return stChange, nil
}

func addMPTNode(ctx context.Context, r *http.Request) (*block.StateChange, error) {
	stChange, err := GetServerChain().blockStateChangeHandler(ctx, r)
	if err != nil {
		return nil, err
	}

	if len(stChange.Nodes) == 0 {
		log.Panicf("Conductor: mpt is empty")
	}

	stChange.AddNode(stChange.Nodes[0])
	return stChange, nil
}

func changePartialState(ctx context.Context, r *http.Request) (*block.StateChange, error) {
	chain := GetServerChain()

	bl, err := chain.getNotarizedBlock(ctx, r.FormValue("round"), r.FormValue("block"))
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}
	prevBlock, err := chain.getNotarizedBlock(ctx, "", bl.PrevHash)
	if err != nil {
		log.Panicf("Conductor: error while getting previous notarised block: %v", err)
	}

	prevBSC := block.NewBlockStateChange(prevBlock)
	prevBSC.Block = bl.Hash
	return prevBSC, nil
}
