package miner

import (
	"context"
	"net/http"

	"0chain.net/common"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/latest_finalized_block", common.ToJSONResponse(LatestFinalizedBlockHandler))
}

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetMinerChain().LatestFinalizedBlock.GetSummary(), nil
}
