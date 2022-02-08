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

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/conductor/config/cases"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

func SetupX2MRequestors() {
	setupX2MRequestors()

	if crpc.Client().State().ClientStatsCollectorEnabled {
		BlockStateChangeRequestor = BlockStateChangeRequestorStats(BlockStateChangeRequestor)
	}
}

// BlockStateChangeRequestorStats represents a middleware for collecting stats about client's block state change requests.
func BlockStateChangeRequestorStats(requestor node.EntityRequestor) node.EntityRequestor {
	return func(urlParams *url.Values, handler datastore.JSONEntityReqResponderF) node.SendHandler {
		if !crpc.Client().State().ClientStatsCollectorEnabled {
			return requestor(urlParams, handler)
		}

		rs := &stats.BlockStateChangeRequest{
			NodeID: node.Self.ID,
			Block:  urlParams.Get("block"),
		}
		if err := crpc.Client().AddBlockStateChangeRequestorStats(rs); err != nil {
			log.Panicf("Conductor: error while adding client stats: %v", err)
		}

		return requestor(urlParams, handler)
	}
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	switch {
	case isIgnoringBlockStateChangeRequest(r):
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

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

func isActingOnBlockStateChangeRequest(r *http.Request, sharders cases.Sharders, miners cases.Miners, onRound int64) bool {
	sChain := GetServerChain()
	bl, err := sChain.getNotarizedBlock(context.Background(), r)
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
