// go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"errors"
	"net/http"
)

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().GetLatestFinalizedBlockSummary(), nil
}

/*LatestFinalizedMagicBlockHandler - provide the latest finalized magic block by this miner */
func LatestFinalizedMagicBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	if lfmb := GetServerChain().GetLatestFinalizedMagicBlock(); lfmb != nil {
		return lfmb, nil
	}

	return nil, errors.New("could not find latest finalized magic block")
}

// LatestFinalizedMagicBlockSummaryHandler - provide the latest finalized magic block summary by this miner */
func LatestFinalizedMagicBlockSummaryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetServerChain()
	if lfmb := c.GetLatestFinalizedMagicBlock(); lfmb != nil {
		return lfmb.GetSummary(), nil
	}

	return nil, errors.New("could not find latest finalized magic block")
}
