//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"errors"
	"net/http"

	"0chain.net/core/common"
)

// swagger:route GET /v1/block/get/latest_finalized miner sharder GetLatestFinalizedBlock
// Get latest finalized block.
// Retrieves the latest finalized block. No parameters needed.
//
// responses:
//  200: BlockSummary
/*LatestFinalizedBlockHandlerSummary - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandlerSummary(ctx context.Context, r *http.Request) (interface{}, error) {
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

// LatestFinalizedMagicBlockSummaryHandler - provide the latest finalized magic block summary by this miner */
func LatestFinalizedMagicBlockSummaryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetServerChain()
	if lfmb := c.GetLatestFinalizedMagicBlockClone(ctx); lfmb != nil {
		return lfmb.GetSummary(), nil
	}

	return nil, errors.New("could not find latest finalized magic block")
}
