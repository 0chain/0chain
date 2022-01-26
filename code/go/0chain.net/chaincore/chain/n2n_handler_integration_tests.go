//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

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
	}
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	switch {
	case isIgnoringBlockStateChangeRequest(r):
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

	case isChangingMPTNode(r):
		return changeMPTNode(ctx, r)

	case isDeletingMPTNode(r):
		return deleteMPTNode(ctx, r)

	case isAddingMPTNode(r):
		return addMPTNode(ctx, r)

	case isChangingPartialState(r):
		return changePartialState(ctx, r)

	case isRespondingCorrectlyOnBlockStateChangeRequest(r):
		fallthrough

	default:
		return c.blockStateChangeHandler(ctx, r)
	}
}

func isIgnoringBlockStateChangeRequest(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil || cfg.Ignored >= 1 {
		return false
	}

	ignoring := isActingOnBlockStateChangeRequest(
		r,
		cfg.IgnoringRequestsBy.Sharders,
		cfg.IgnoringRequestsBy.Miners,
		cfg.OnRound,
	)
	if ignoring {
		cfg.Ignored++
	}
	return ignoring
}

func isRespondingCorrectlyOnBlockStateChangeRequest(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil {
		return false
	}

	return isActingOnBlockStateChangeRequest(
		r,
		cfg.CorrectResponseBy.Sharders,
		cfg.CorrectResponseBy.Miners,
		cfg.OnRound,
	)
}

func isChangingMPTNode(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil {
		return false
	}

	return isActingOnBlockStateChangeRequest(
		r,
		cfg.ChangedMPTNodeBy.Sharders,
		cfg.ChangedMPTNodeBy.Miners,
		cfg.OnRound,
	)
}

func isDeletingMPTNode(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil {
		return false
	}

	return isActingOnBlockStateChangeRequest(
		r,
		cfg.DeletedMPTNodeBy.Sharders,
		cfg.DeletedMPTNodeBy.Miners,
		cfg.OnRound,
	)
}

func isAddingMPTNode(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil {
		return false
	}

	return isActingOnBlockStateChangeRequest(
		r,
		cfg.AddedMPTNodeBy.Sharders,
		cfg.AddedMPTNodeBy.Miners,
		cfg.OnRound,
	)
}

func isChangingPartialState(r *http.Request) bool {
	cfg := crpc.Client().State().BlockStateChangeRequestor

	cfg.Lock()
	defer cfg.Unlock()

	if cfg == nil {
		return false
	}

	return isActingOnBlockStateChangeRequest(
		r,
		cfg.PartialStateFromAnotherBlockBy.Sharders,
		cfg.PartialStateFromAnotherBlockBy.Miners,
		cfg.OnRound,
	)
}

func isActingOnBlockStateChangeRequest(r *http.Request, sharders cases.Sharders, miners cases.Miners, onRound int64) bool {
	sChain := GetServerChain()
	bl, err := sChain.getNotarizedBlock(context.Background(), r.FormValue("round"), r.FormValue("block"))
	if err != nil || bl.Round != onRound {
		return false
	}

	if node.Self.Type == node.NodeTypeSharder {
		selfName := "sharder-" + strconv.Itoa(node.Self.SetIndex)
		return sharders.Contains(selfName)
	}

	// node type miner

	var (
		roundMiners                             = sChain.GetMiners(bl.Round)
		isRequestorGenerator, requestorTypeRank = getMinerTypeAndTypeRank(bl.Round, bl.RoundRandomSeed, roundMiners, r.Header.Get(node.HeaderNodeID))
		isSelfGenerator, selfTypeRank           = getMinerTypeAndTypeRank(bl.Round, bl.RoundRandomSeed, roundMiners, node.Self.ID)
	)
	return !isRequestorGenerator && requestorTypeRank == 0 && // replica0
		miners.Get(isSelfGenerator, selfTypeRank) != nil
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

// getMinerTypeAndTypeRank return true if the provided miner is generator and type rank of the provided miner.
//
// 	Explaining type rank example:
//		Generators num = 2
// 		len(miners) = 4
// 		Generator0:	rank = 0; typeRank = 0; isGenerator = true.
// 		Generator1:	rank = 1; typeRank = 1; isGenerator = true.
// 		Replica0:	rank = 2; typeRank = 0; isGenerator = false.
// 		Replica0:	rank = 3; typeRank = 1; isGenerator = false.
func getMinerTypeAndTypeRank(roundNum, seed int64, miners *node.Pool, minerID string) (isGenerator bool, typeRank int) {
	roundI := round.NewRound(roundNum)
	roundI.SetRandomSeed(seed, len(miners.Nodes))
	genNum := GetServerChain().GetGeneratorsNum()
	miner := miners.GetNode(minerID)
	minerRank := roundI.GetMinerRank(miner)
	isGenerator = minerRank < genNum
	typeRank = minerRank
	if !isGenerator {
		typeRank = typeRank - genNum
	}
	return isGenerator, typeRank
}
