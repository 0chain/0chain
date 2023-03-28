package sharder

import (
	"fmt"
	"net/http"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
)

// HealthCheckWriter - a handler to provide block statistics
func HealthCheckWriter(w http.ResponseWriter, r *http.Request) {
	sc := GetSharderChain()
	c := sc.Chain
	w.Header().Set("Content-Type", "text/html")
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Underlying()
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)
	fmt.Fprintf(w, "<table>")

	fmt.Fprintf(w, "<tr><td valign='top'><h2>General Info</h2>")
	diagnostics.WriteConfiguration(w, c)
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "</td><td valign='top'><h2>Minio Info</h2>")
	sc.WriteMinioStats(w)
	fmt.Fprintf(w, "</td></tr>")

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
