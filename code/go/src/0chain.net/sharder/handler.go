package sharder

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/diagnostics"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/block/get", common.ToJSONResponse(BlockHandler))
	http.HandleFunc("/_block_stats", BlockStatsHandler)
}

/*BlockHandler - a handler to respond to block queries */
func BlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("block")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	var err error
	var b *block.Block
	if hash == "" {
		return nil, common.InvalidRequest("Block hash is required")
	}
	b, err = chain.GetServerChain().GetBlock(ctx, hash)
	if err == nil {
		return chain.GetBlockResponse(b, parts)
	}
	sc := GetSharderChain()
	/*NOTE: We store DIR_ROUND_RANGE number of blocks in the same directory and that's a large number (10M).
	So, as long as people query the last 10M blocks most of the time, we only end up with 1 or 2 iterations.
	Anything older than that, there is a cost to query the database and get the round informatio anyway.
	*/
	for r := sc.LatestFinalizedBlock.Round; r > 0; r -= blockstore.DIR_ROUND_RANGE {
		b, err = sc.GetBlockFromStore(hash, r)
		if err != nil {
			return nil, err
		}
	}
	return chain.GetBlockResponse(b, parts)
}

/*BlockStatsHandler - a handler to provide block statistics */
func BlockStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := &GetSharderChain().Chain
	fmt.Fprintf(w, "<h2>Block Finalization Statistics</h2>")
	diagnostics.WriteStatistics(w, c, timer, 1000000.0)
}
