package miner

import (
	"fmt"
	"net/http"

	"0chain.net/chain"
	"0chain.net/diagnostics"
)

/*SetupHandlers - setup miner handlers */
func SetupHandlers() {
	http.HandleFunc("/_chain_stats", ChainStatsHandler)
}

/*ChainStatsHandler - a handler to provide block statistics */
func ChainStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := GetMinerChain().Chain
	diagnostics.WriteStatisticsCSS(w)
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Steady state)</h2>")
	diagnostics.WriteStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Start to Finish)</h2>")
	diagnostics.WriteStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "<p>Block finalization time = block generation + block verification + k*(network latency)</p>")
	fmt.Fprintf(w, "<h2>Block Generation Statistics</h2>")
	diagnostics.WriteStatistics(w, c, bgTimer, 1000000.0)
	fmt.Fprintf(w, "<h2>Block Verification Statistics</h2>")
	diagnostics.WriteStatistics(w, c, bvTimer, 1000000.0)
	fmt.Fprintf(w, "<h2>State Save Statistics</h2>")
	diagnostics.WriteStatistics(w, c, chain.StateSaveTimer, 1000000.0)
}
