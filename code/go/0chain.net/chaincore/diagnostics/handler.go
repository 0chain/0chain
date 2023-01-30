package diagnostics

import (
	"fmt"
	"net/http"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
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
	http.HandleFunc("/_diagnostics/txns_in_pool", common.UserRateLimit(sc.TxnsInPoolHandler))
	http.HandleFunc("/_diagnostics/block_chain", common.UserRateLimit(sc.WIPBlockChainHandler))
}

// swagger:model ChainStats
type ChainStats struct {
	Delta              time.Duration `json:"delta"`
	CurrentRound       int64         `json:"current_round"`
	LastFinalizedRound int64         `json:"latest_finalized_round"`
	Count              int64         `json:"count"`
	Min                float64       `json:"min"`
	Max                float64       `json:"max"`
	Mean               float64       `json:"mean"`
	StdDev             float64       `json:"std_dev"`
	RunningTxnCount    int64         `json:"total_txns"`
	Rate1              float64       `json:"rate_1_min"`
	Rate5              float64       `json:"rate_5_min"`
	Rate15             float64       `json:"rate_15_min"`
	RateMean           float64       `json:"rate_mean"`
	Percentile50       float64       `json:"percentile_50"`
	Percentile90       float64       `json:"percentile_90"`
	Percentile95       float64       `json:"percentile_95"`
	Percentile99       float64       `json:"percentile_99"`
}

/*GetStatistics - write the statistics of the given timer */
func GetStatistics(c *chain.Chain, timer metrics.Timer, scaleBy float64) ChainStats {
	scale := func(n float64) float64 {
		return (n / scaleBy)
	}

	percentiles := []float64{0.5, 0.9, 0.95, 0.99}
	pvals := timer.Percentiles(percentiles)
	lfb := c.GetLatestFinalizedBlock()

	stats := ChainStats{
		Delta:              chain.DELTA,
		CurrentRound:       c.GetCurrentRound(),
		LastFinalizedRound: lfb.Round,
		Count:              timer.Count(),
		Min:                scale(float64(timer.Min())),
		Mean:               scale(timer.Mean()),
		StdDev:             scale(timer.StdDev()),
		Max:                scale(float64(timer.Max())),
		RunningTxnCount:    lfb.RunningTxnCount,
		Rate1:              timer.Rate1(),
		Rate5:              timer.Rate5(),
		Rate15:             timer.Rate15(),
		RateMean:           timer.RateMean(),
		Percentile50:       scale(pvals[0]),
		Percentile90:       scale(pvals[1]),
		Percentile95:       scale(pvals[2]),
		Percentile99:       scale(pvals[3]),
	}
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
		fmt.Fprintf(w, "<tr><td class='tname'>Finalized Round</td><td>%v (%v)</td></tr>", lfb.Round, len(lfb.uniqueBlockExtensions))
	}
	if c.LatestDeterministicBlock != nil {
		fmt.Fprintf(w, "<tr><td class='tname'>Deterministic Finalized Round</td><td>%v (%v)</td></tr>", c.LatestDeterministicBlock.Round, len(c.LatestDeterministicBlock.uniqueBlockExtensions))
		if c.LatestDeterministicBlock != lfb {
			var maxUBE int
			var maxUBERound int64
			for b := lfb; b != nil && b != c.LatestDeterministicBlock; b = b.PrevBlock {
				var ube = len(b.uniqueBlockExtensions)
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

// WritePruneStats - write the last prune stats
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
