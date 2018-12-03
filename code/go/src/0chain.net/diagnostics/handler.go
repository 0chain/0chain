package diagnostics

import (
	"fmt"
	"net/http"

	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/logging"
	metrics "github.com/rcrowley/go-metrics"
)

/*SetupHandlers - setup diagnostics handlers */
func SetupHandlers() {
	http.HandleFunc("/_diagnostics/info", chain.InfoWriter)
	http.HandleFunc("/v1/diagnostics/get/info", common.ToJSONResponse(chain.InfoHandler))
	http.HandleFunc("/_diagnostics/logs", logging.LogWriter)
	http.HandleFunc("/_diagnostics/n2n_logs", logging.N2NLogWriter)
	http.HandleFunc("/_diagnostics/mem_logs", logging.MemLogWriter)
	sc := chain.GetServerChain()
	http.HandleFunc("/_diagnostics/n2n/info", sc.N2NStatsWriter)
	http.HandleFunc("/_diagnostics/miner_stats", sc.MinerStatsHandler)
	http.HandleFunc("/_diagnostics/block_chain", sc.WIPBlockChainHandler)
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
	stats["block_size"] = c.BlockSize
	stats["current_round"] = c.CurrentRound
	stats["latest_finalized_round"] = c.LatestFinalizedBlock.Round
	stats["count"] = timer.Count()
	stats["min"] = scale(float64(timer.Min()))
	stats["mean"] = scale(timer.Mean())
	stats["std_dev"] = scale(timer.StdDev())
	stats["max"] = scale(float64(timer.Max()))
	stats["total_txns"] = c.LatestFinalizedBlock.RunningTxnCount

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
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Round Generators/Sharders</td><td>%d/%d</td></tr>", c.NumGenerators, c.NumGenerators)
	fmt.Fprintf(w, "<tr><td class='sheader' colspan='2'>Configuration <a href='v1/config/get'>...</a></td></tr>")
	fmt.Fprintf(w, "<tr><td>Block Size</td><td>%v - %v</td></tr>", c.MinBlockSize, c.BlockSize)
	fmt.Fprintf(w, "<tr><td>Network Latency (Delta)</td><td>%v</td></tr>", chain.DELTA)
	proposalMode := "dynamic"
	if c.BlockProposalWaitMode == chain.BlockProposalWaitStatic {
		proposalMode = "static"
	}
	fmt.Fprintf(w, "<tr><td>Block Proposal Wait Time</td><td>%v (%v)</td>", c.BlockProposalMaxWaitTime, proposalMode)

	fmt.Fprintf(w, "<tr><td>Validation Batch Size</td><td>%d</td>", c.ValidationBatchSize)
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
	fmt.Fprintf(w, "<tr><td class='sheader' colspan='2'>Block rate per second</td></tr>")
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
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><th class='sheader' colspan='2'>Current Status</th></tr>")
	fmt.Fprintf(w, "<tr><td>Current Round</td><td>%v</td></tr>", c.CurrentRound)
	if c.LatestFinalizedBlock != nil {
		fmt.Fprintf(w, "<tr><td>Latest Finalized Round</td><td>%v</td></tr>", c.LatestFinalizedBlock.Round)
	}
	fmt.Fprintf(w, "</table>")
}
