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
	if lfmb := GetServerChain().GetLatestFinalizedMagicBlockClone(ctx); lfmb != nil {
		return lfmb, nil
	}

	return nil, errors.New("could not find latest finalized magic block")
}
