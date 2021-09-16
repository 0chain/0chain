// +build !integration_tests

package chain

import (
	"context"
	"net/http"
)

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().GetLatestFinalizedBlockSummary(), nil
}

/*LatestFinalizedMagicBlockHandler - provide the latest finalized magic block by this miner */
func LatestFinalizedMagicBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetServerChain()
	return c.GetLatestFinalizedMagicBlockRound(c.GetCurrentRound()), nil
}

// LatestFinalizedMagicBlockSummaryHandler - provide the latest finalized magic block summary by this miner */
func LatestFinalizedMagicBlockSummaryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetServerChain()
	lfmb := c.GetLatestFinalizedMagicBlockRound(c.GetCurrentRound())
	if lfmb != nil {
		return lfmb.GetSummary(), nil
	}

	return nil, nil
}
