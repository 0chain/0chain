//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"0chain.net/core/logging"
	"context"
	"errors"
	"go.uber.org/zap"
	"net/http"

	"0chain.net/core/common"
)

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().GetLatestFinalizedBlockSummary(), nil
}

/*LatestFinalizedMagicBlockHandler - provide the latest finalized magic block by this miner */
func LatestFinalizedMagicBlockHandler(c Chainer) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		logging.Logger.Info("piers LatestFinalizedMagicBlockHandler")
		nodeLFMBHash := r.FormValue("node-lfmb-hash")
		lfmb := c.GetLatestFinalizedMagicBlockClone(ctx)
		if lfmb == nil {
			logging.Logger.Info("piers LatestFinalizedMagicBlockHandler error",
				zap.Error(errors.New("could not find latest finalized magic block")))
			return nil, errors.New("could not find latest finalized magic block")
		}

		if lfmb.Hash == nodeLFMBHash {
			logging.Logger.Info("piers LatestFinalizedMagicBlockHandler error",
				zap.Error(common.ErrNotModified))
			return nil, common.ErrNotModified
		}
		logging.Logger.Info("piers LatestFinalizedMagicBlockHandler end", zap.Any("lfmb", lfmb))
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

// SetupHandlers sets up the necessary API end points.
func SetupHandlers(c Chainer) {
	setupHandlers(handlersMap(c))
}
