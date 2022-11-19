//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (
	interface{}, error) {

	var state = crpc.Client().State()
	if state.FinalizedBlock != nil {
		// bad
		var lfbs = GetServerChain().GetLatestFinalizedBlockSummary()
		lfbs.Hash = util.RevertString(lfbs.Hash)
		return lfbs, nil
	}

	return GetServerChain().GetLatestFinalizedBlockSummary(), nil
}

/*LatestFinalizedMagicBlockHandler - provide the latest finalized magic block by this miner */
func LatestFinalizedMagicBlockHandler(c Chainer) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		nodeLFMBHash := r.FormValue("node-lfmb-hash")
		lfmb := c.GetLatestFinalizedMagicBlockClone(ctx)
		if lfmb == nil {
			return nil, errors.New("could not find latest finalized magic block")
		}

		if lfmb.Hash == nodeLFMBHash {
			return nil, common.ErrNotModified
		}

		return lfmb, nil
	}
}

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers(c Chainer) {
	hMap := handlersMap(c)

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		hMap[GetBlockV1Pattern] = BlockStats(
			hMap[GetBlockV1Pattern],
			BlockStatsConfigurator{
				HashKey: "block",
			},
		)
	}

	setupHandlers(hMap)
}

type (
	// BlockStatsConfigurator contains needed for the BlockStats middleware information.
	BlockStatsConfigurator struct {
		HashKey      string
		SenderHeader string
	}
)

// BlockStats represents middleware for collecting nodes blocks servers stats.
func BlockStats(handler func(http.ResponseWriter, *http.Request), cfg BlockStatsConfigurator) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !crpc.Client().State().ServerStatsCollectorEnabled {
			handler(w, r)
			return
		}

		roundStr := r.FormValue("round")
		roundNum := 0
		if roundStr != "" {
			var err error
			roundNum, err = strconv.Atoi(roundStr)
			if err != nil {
				log.Panicf("Conductor: error while converting round from string: %v", err)
			}
		}
		ss := &stats.BlockRequest{
			NodeID:   node.Self.ID,
			Hash:     r.FormValue(cfg.HashKey),
			Round:    roundNum,
			SenderID: r.Header.Get(cfg.SenderHeader),
		}
		if err := crpc.Client().AddBlockServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		handler(w, r)
	}
}

// LatestFinalizedMagicBlockSummaryHandler - provide the latest finalized magic block summary by this miner */
func LatestFinalizedMagicBlockSummaryHandler(ctx context.Context, r *http.Request) (interface{}, error) {

	var state = crpc.Client().State()
	if state.MagicBlock != nil {
		var lfmb = GetServerChain().GetLatestFinalizedMagicBlockClone(ctx)
		lfmb.Hash = util.RevertString(lfmb.Hash)
		return lfmb.GetSummary(), nil
	}

	return GetServerChain().GetLatestFinalizedMagicBlockClone(ctx), nil
}
