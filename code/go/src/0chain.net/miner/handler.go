package miner

import (
	"fmt"
	"net/http"

	"0chain.net/diagnostics"
)

/*SetupHandlers - setup miner handlers */
func SetupHandlers() {
	http.HandleFunc("/_block_stats", BlockStatsHandler)
}

/*BlockStatsHandler - a handler to provide block statistics */
func BlockStatsHandler(w http.ResponseWriter, r *http.Request) {
	c := &GetMinerChain().Chain
	fmt.Fprintf(w, "<h2>Block Generation Statistics</h2>")
	diagnostics.WriteStatistics(w, c, bgTimer)
	fmt.Fprintf(w, "<h2>Block Verification Statistics</h2>")
	diagnostics.WriteStatistics(w, c, bvTimer)
}
