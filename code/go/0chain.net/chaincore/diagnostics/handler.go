package diagnostics

import (
	"fmt"
	"net/http"

	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"github.com/0chain/common/core/util"
	metrics "github.com/rcrowley/go-metrics"
)

/*SetupHandlers - setup diagnostics handlers */
func SetupHandlers() {
	http.HandleFunc("/_diagnostics/info", common.UserRateLimit(chain.InfoWriter))
	http.HandleFunc("/v1/diagnostics/get/info", common.UserRateLimit(common.ToJSONResponse(chain.InfoHandler)))
	http.HandleFunc("/_diagnostics/logs", common.UserRateLimit(logging.LogWriter))
	http.HandleFunc("/_diagnostics/n2n_logs", common.UserRateLimit(logging.N2NLogWriter))
	http.HandleFunc("/_diagnostics/mem_logs", common.UserRateLimit(logging.MemLogWriter))
	sc := chain.GetServerChain()
	http.HandleFunc("/_diagnostics/n2n/info", common.UserRateLimit(sc.N2NStatsWriter))
	http.HandleFunc("/_diagnostics/miner_stats", common.UserRateLimit(sc.MinerStatsHandler))
	http.HandleFunc("/_diagnostics/block_chain", common.UserRateLimit(sc.WIPBlockChainHandler))
}

/*GetStatistics - write the statistics of the given timer */
func GetStatistics(c *chain.Chain, timer metrics.Timer, scaleBy float64) interface{} {
	scale := func(n float64) float64 {
		return (n / scaleBy)
	}
	percentiles := []float64{0.5, 0.9, 0.95, 0.99}
	pvals := timer.Percentiles(percentiles)
	stats := make(map[string]interface{})
	stats["delta"] = chain.DELTA
	stats["block_size"] = c.BlockSize()
	stats["current_round"] = c.GetCurrentRound()
	lfb := c.GetLatestFinalizedBlock()
	stats["latest_finalized_round"] = lfb.Round
	stats["count"] = timer.Count()
	stats["min"] = scale(float64(timer.Min()))
	stats["mean"] = scale(timer.Mean())
	stats["std_dev"] = scale(timer.StdDev())
	stats["max"] = scale(float64(timer.Max()))
	stats["total_txns"] = lfb.RunningTxnCount

	for idx, p := range percentiles {
		stats[fmt.Sprintf("percentile_%v", 100*p)] = scale(pvals[idx])
	}
	stats["rate_1_min"] = timer.Rate1()
	stats["rate_5_min"] = timer.Rate5()
	stats["rate_15_min"] = timer.Rate15()
	stats["rate_mean"] = timer.RateMean()
	return stats
}

/*WriteStatisticsCSS - write the css for the statistics html */
func WriteStatisticsCSS(w http.ResponseWriter) {
	fmt.Fprintf(w, "<style>.sheader { color: orange; font-weight: bold; }</style>")
}

/*WriteConfiguration - write summary information */
func WriteConfiguration(w http.ResponseWriter, c *chain.Chain) {
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='tname'>Round Generators/Replicators</td><td>%d/%d</td></tr>", c.GetGeneratorsNum(), c.NumReplicators())
	fmt.Fprintf(w, "<tr><td class='tname'>Block Size</td><td>%v - %v</td></tr>", c.MinBlockSize(), c.BlockSize())
	fmt.Fprintf(w, "<tr><td class='tname'>Network Latency (Delta)</td><td>%v</td></tr>", chain.DELTA)
	proposalMode := "dynamic"
	if c.BlockProposalWaitMode() == chain.BlockProposalWaitStatic {
		proposalMode = "static"
	}
	fmt.Fprintf(w, "<tr><td class='tname'>Block Proposal Wait Time</td><td>%v (%v)</td>", c.BlockProposalMaxWaitTime(), proposalMode)

	fmt.Fprintf(w, "<tr><td class='tname'>Validation Batch Size</td><td>%d</td>", c.ValidationBatchSize())
	fmt.Fprintf(w, "</table>")
}

/*WriteTimerStatistics - write the statistics of the given timer */
func WriteTimerStatistics(w http.ResponseWriter, c *chain.Chain, timer metrics.Timer, scaleBy float64) {
	scale := func(n float64) float64 {
		return (n / scaleBy)
	}
	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := timer.Percentiles(percentiles)
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Metrics</td></tr>")
	fmt.Fprintf(w, "<tr><td>Count</td><td>%v</td></tr>", timer.Count())
	fmt.Fprintf(w, "<tr><td class='sheader' colspan='2'>Time taken</td></tr>")
	fmt.Fprintf(w, "<tr><td>Min</td><td>%.2f ms</td></tr>", scale(float64(timer.Min())))
	fmt.Fprintf(w, "<tr><td>Mean</td><td>%.2f &plusmn;%.2f ms</td></tr>", scale(timer.Mean()), scale(timer.StdDev()))
	fmt.Fprintf(w, "<tr><td>Max</td><td>%.2f ms</td></tr>", scale(float64(timer.Max())))
	for idx, p := range percentiles {
		fmt.Fprintf(w, "<tr><td>%.2f%%</td><td>%.2f ms</td></tr>", 100*p, scale(pvals[idx]))
	}
	fmt.Fprintf(w, "<tr><td class='sheader' colspan='2'>Rate per second</td></tr>")
	fmt.Fprintf(w, "<tr><td>Last 1-min rate</td><td>%.2f</td></tr>", timer.Rate1())
	fmt.Fprintf(w, "<tr><td>Last 5-min rate</td><td>%.2f</td></tr>", timer.Rate5())
	fmt.Fprintf(w, "<tr><td>Last 15-min rate</td><td>%.2f</td></tr>", timer.Rate15())
	fmt.Fprintf(w, "<tr><td>Overall mean rate</td><td>%.2f</td></tr>", timer.RateMean())
	fmt.Fprintf(w, "</table>")
}

/*WriteHistogramStatistics - write the statistics of the given histogram */
func WriteHistogramStatistics(w http.ResponseWriter, c *chain.Chain, metric metrics.Histogram) {
	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := metric.Percentiles(percentiles)
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Metrics</td></tr>")
	fmt.Fprintf(w, "<tr><td>Count</td><td>%v</td></tr>", metric.Count())
	fmt.Fprintf(w, "<tr><td class='sheader' colspan='2'>Metric Value</td></tr>")
	fmt.Fprintf(w, "<tr><td>Min</td><td>%.2f</td></tr>", float64(metric.Min()))
	fmt.Fprintf(w, "<tr><td>Mean</td><td>%.2f &plusmn;%.2f</td></tr>", metric.Mean(), metric.StdDev())
	fmt.Fprintf(w, "<tr><td>Max</td><td>%.2f</td></tr>", float64(metric.Max()))
	for idx, p := range percentiles {
		fmt.Fprintf(w, "<tr><td>%.2f%%</td><td>%.2f</td></tr>", 100*p, pvals[idx])
	}
	fmt.Fprintf(w, "</table>")
}

/*WriteCurrentStatus - write the current status of the chain */
func WriteCurrentStatus(w http.ResponseWriter, c *chain.Chain) {
	fmt.Fprintf(w, "<table width='100%%' >")
	fmt.Fprintf(w, "<tr><td class='tname'>Current Round</td><td>%v</td></tr>", c.GetCurrentRound())
	lfb := c.GetLatestFinalizedBlock()
	if lfb != nil {
		fmt.Fprintf(w, "<tr><td class='tname'>Finalized Round</td><td>%v (%v)</td></tr>", lfb.Round, len(lfb.UniqueBlockExtensions))
	}
	if c.LatestDeterministicBlock != nil {
		fmt.Fprintf(w, "<tr><td class='tname'>Deterministic Finalized Round</td><td>%v (%v)</td></tr>", c.LatestDeterministicBlock.Round, len(c.LatestDeterministicBlock.UniqueBlockExtensions))
		if c.LatestDeterministicBlock != lfb {
			var maxUBE int
			var maxUBERound int64
			for b := lfb; b != nil && b != c.LatestDeterministicBlock; b = b.PrevBlock {
				var ube = len(b.UniqueBlockExtensions)
				if ube > maxUBE {
					maxUBE = ube
					maxUBERound = b.Round
				}
			}
			fmt.Fprintf(w, "<tr><td class='tname'>Next round to be deterministic</td><td>%v (%v)</td></tr>", maxUBERound, maxUBE)
		}
	}
	fmt.Fprintf(w, "</table>")
}

//WritePruneStats - write the last prune stats
func WritePruneStats(w http.ResponseWriter, ps *util.PruneStats) {
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td>Stage</td><td>%v</td>", ps.Stage)
	fmt.Fprintf(w, "<tr><td>Pruned Below Round</td><td class='number'>%v</td></tr>", ps.Version)
	fmt.Fprintf(w, "<tr><td>Missing Nodes</td><td class='number'>%v</td></tr>", ps.MissingNodes)
	fmt.Fprintf(w, "<tr><td>Total nodes</td><td class='number'>%v</td></tr>", ps.Total)
	fmt.Fprintf(w, "<tr><td>Leaf Nodes</td><td class='number'>%v</td></tr>", ps.Leaves)
	fmt.Fprintf(w, "<tr><td>Nodes Below Pruned Round</td><td class='number'>%v</td></tr>", ps.BelowVersion)
	fmt.Fprintf(w, "<tr><td>Update Time</td><td class='number'>%v</td>", ps.UpdateTime)
	fmt.Fprintf(w, "<tr><td>Deleted Nodes</td><td class='number'>%v</td></tr>", ps.Deleted)
	fmt.Fprintf(w, "<tr><td>Delete Time</td><td class='number'>%v</td>", ps.DeleteTime)
	fmt.Fprintf(w, "</table>")
}
