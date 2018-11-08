package miner

import (
	"fmt"
	"net/http"

	"0chain.net/chain"
	"0chain.net/config"
	"0chain.net/diagnostics"
	"0chain.net/node"
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

	self := node.Self.Node
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)

	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Steady state)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Start to Finish)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<tr><td col='2'>")
	fmt.Fprintf(w, "<p>Block finalization time = block generation + block verification + network time (1*large message + 2*small message)</p>")
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Txn Finalization Statistics (Start to Finish)</h2>")
	if config.Development() {
		diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTxnTimer, 1000000.0)
	} else {
		fmt.Fprintf(w, "Available only in development mode")
	}
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h2>Finalization Lag Statistics</h2>")
	diagnostics.WriteHistogramStatistics(w, c, chain.FinalizationLagMetric)
	fmt.Fprintf(w, "</td><td></td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Generation Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bgTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Verification Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bvTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>State Save Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StateSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td></td></tr>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>State Prune Update Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneUpdateTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>State Prune Delete Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneDeleteTimer, 1000000.0)
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")
}
