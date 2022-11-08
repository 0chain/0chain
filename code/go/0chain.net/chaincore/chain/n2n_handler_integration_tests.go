//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/conductor/config/cases"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

func SetupX2MRequestors() {
	setupX2MRequestors()

	if crpc.Client().State().ClientStatsCollectorEnabled {
		BlockStateChangeRequestor = BlockRequestor(BlockStateChangeRequestor, stats.BRBlockStateChange, "block")
		MinerNotarizedBlockRequestor = BlockRequestor(MinerNotarizedBlockRequestor, stats.BRMinerNotarisedBlock, "block")
	}
}

func SetupX2SRequestors() {
	setupX2SRequestors()

	if crpc.Client().State().ClientStatsCollectorEnabled {
		FBRequestor = BlockRequestor(FBRequestor, stats.BRFB, "hash")
	}
}

func BlockRequestor(requestor node.EntityRequestor, requestorType stats.BlockRequestor, blockHashKey string) node.EntityRequestor {
	return func(urlParams *url.Values, handler datastore.JSONEntityReqResponderF) node.SendHandler {
		if !crpc.Client().State().ClientStatsCollectorEnabled {
			return requestor(urlParams, handler)
		}

		rNum, _ := strconv.Atoi(urlParams.Get("round"))
		rs := &stats.BlockRequest{
			NodeID: node.Self.ID,
			Round:  rNum,
			Hash:   urlParams.Get(blockHashKey),
		}
		if err := crpc.Client().AddBlockClientStats(rs, requestorType); err != nil {
			log.Panicf("Conductor: error while adding client stats: %v", err)
		}

		return requestor(urlParams, handler)
	}
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	cfg := crpc.Client().State().BlockStateChangeRequestor
	if cfg == nil {
		return c.blockStateChangeHandler(ctx, r)
	}

	minerInformer := createMinerInformer(r)
	requestorID := r.Header.Get(node.HeaderNodeID)
	selfInfo := cases.SelfInfo{
		IsSharder: node.Self.Type == node.NodeTypeSharder,
		ID:        node.Self.ID,
		SetIndex:  node.Self.SetIndex,
	}

	cfg.Lock()
	defer cfg.Unlock()

	switch {
	case cfg.IgnoringRequestsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo) && cfg.Ignored < 1:
		cfg.Ignored++
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

	case cfg.ChangedMPTNodeBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return changeMPTNode(r)

	case cfg.DeletedMPTNodeBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return deleteMPTNode(ctx, r)

	case cfg.AddedMPTNodeBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return addMPTNode(r)

	case cfg.PartialStateFromAnotherBlockBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return changePartialState(ctx, r)

	case cfg.CorrectResponseBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		fallthrough

	default:
		return c.blockStateChangeHandler(ctx, r)
	}
}

func createMinerInformer(r *http.Request) *cases.MinerInformer {
	sChain := GetServerChain()
	bl, err := sChain.getNotarizedBlock(context.Background(), r.FormValue("round"), r.FormValue("block"))
	if err != nil {
		return nil
	}

	miners := sChain.GetMiners(bl.Round)

	roundI := round.NewRound(bl.Round)
	roundI.SetRandomSeed(bl.RoundRandomSeed, len(miners.Nodes))

	return cases.NewMinerInformer(
		NewRanker(roundI, miners),
		sChain.GetGeneratorsNum(),
	)
}

func changeMPTNode(r *http.Request) (*block.StateChange, error) {
	sChain := GetServerChain()
	bl, err := sChain.getNotarizedBlock(context.Background(), r.FormValue("round"), r.FormValue("block"))
	if err != nil {
		log.Panicf("Conductor: error while fetching notarized block: %v", err)
	}

	bsc, err := block.NewBlockStateChange(bl)
	if err != nil {
		log.Panicf("Conductor: error while createing block state change: %v", err)
	}

	st := state.State{
		TxnHashBytes: encryption.RawHash("txn hash"),
		Round:        bl.Round,
		Balance:      1000000000,
	}

	for _, n := range bsc.Nodes {
		if n.GetNodeType() == util.NodeTypeLeafNode {
			ln, ok := n.(*util.LeafNode)
			if !ok {
				log.Panic("Conductor: unexpected node type")
			}
			ln.SetValue(&st)
		}
	}

	return bsc, nil
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

func addMPTNode(r *http.Request) (*block.StateChange, error) {
	sChain := GetServerChain()
	bl, err := sChain.getNotarizedBlock(context.Background(), r.FormValue("round"), r.FormValue("block"))
	if err != nil {
		log.Panicf("Conductor: error while fetching notarized block: %v", err)
	}

	bsc, err := block.NewBlockStateChange(bl)
	if err != nil {
		log.Panicf("Conductor: error while createing block state change: %v", err)
	}

	lastNode := bsc.Nodes[len(bsc.Nodes)-1]
	st := state.State{
		TxnHashBytes: encryption.RawHash("txn hash"),
		Round:        bl.Round,
		Balance:      1000000000,
	}
	bsc.AddNode(util.NewLeafNode(util.Path(""), util.Path(lastNode.GetHash()), lastNode.GetOrigin(), &st))

	return bsc, nil
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

	prevBSC, err := block.NewBlockStateChange(prevBlock)
	if err != nil {
		log.Panicf("Conductor: error while creating block state changes: %v", err)
	}
	prevBSC.Block = bl.Hash
	return prevBSC, nil
}
