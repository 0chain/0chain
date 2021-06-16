// +build !integration_tests

package chain

import (
	"context"
	"net/http"

	"0chain.net/miner/minerGRPC"
)

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(svc *minerChainGRPCService) func(ctx context.Context, r *http.Request) (interface{}, error) {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		resp, err := svc.GetLatestFinalizedBlockSummary(ctx, &minerGRPC.GetLatestFinalizedBlockSummaryRequest{})
		if err != nil {
			return nil, err
		}

		return BlockSummaryGRPCToBlockSummary(resp.BlockSummary), nil
	}
}

/*LatestFinalizedMagicBlockHandler - provide the latest finalized magic block by this miner */
func LatestFinalizedMagicBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().GetLatestFinalizedMagicBlock(), nil
}
