package sharder

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/diagnostics"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/block/get", common.ToJSONResponse(BlockHandler))
	http.HandleFunc("/_block_stats", BlockStatsHandler)
}

//BlockHandler - a handler to respond to block queries */
func BlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("block")
	round := r.FormValue("round")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	var roundNumber int64 = -1
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
	if roundNumber == -1 {
		if round != "" {
			roundNumber, err = strconv.ParseInt(round, 10, 63)
			if err != nil {
				return nil, err
			}
		} else {
			// TODO: Get the round from the hash
		}
	}
	b, err = sc.GetBlockFromStore(hash, roundNumber)
	if err != nil {
		return nil, err
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
