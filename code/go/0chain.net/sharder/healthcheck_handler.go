package sharder

import (
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"fmt"
	"net/http"
)


/*ChainStatsWriter - a handler to provide block statistics */
func HealthCheckWriter(w http.ResponseWriter, r *http.Request) {
	sc := GetSharderChain()
	c := sc.Chain
	w.Header().Set("Content-Type", "text/html")
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Node
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)

	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "<br>")
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "<br>")

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td valign='top'><h2>Deep Scan Configuration</h2>")
	sc.WriteHealthCheckConfiguration(w, DeepScan)
	fmt.Fprintf(w, "</td><td valign='top'><h2>Proximity Scan Configuration</h2>")
	sc.WriteHealthCheckConfiguration(w, ProximityScan)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td valign='top'><h2>Deep Scan Results - Block Summary</h2>")
	sc.WriteHealthCheckBlockSummary(w, DeepScan)
	fmt.Fprintf(w, "</td><td valign='top'><h2>Proximity Scan Results - Block Summary</h2>")
	sc.WriteHealthCheckBlockSummary(w, ProximityScan)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td><h2>Deep Scan Block Statistics</h2>")
	sc.WriteBlockSyncStatistics(w, DeepScan)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td><h2>Proximity Scan Block Statistics</h2>")
	sc.WriteBlockSyncStatistics(w, ProximityScan)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "</table>")

}
