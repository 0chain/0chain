//go:build integration_tests
// +build integration_tests

package chain

import (
	"0chain.net/core/common"
	"context"
	"errors"
	"net/http"

	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats/middleware"
)

func revertString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (
	interface{}, error) {

	var state = crpc.Client().State()
	if state.FinalizedBlock != nil {
		// bad
		var lfbs = GetServerChain().GetLatestFinalizedBlockSummary()
		lfbs.Hash = revertString(lfbs.Hash)
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
func SetupHandlers() {
	hMap := handlersMap()

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		hMap[getBlockV1Pattern] = middleware.BlockStats(
			hMap[getBlockV1Pattern],
			middleware.BlockStatsConfigurator{
				HashKey: "block",
				Handler: getBlockV1Pattern,
			},
		)
	}

	setupHandlers(hMap)
}
