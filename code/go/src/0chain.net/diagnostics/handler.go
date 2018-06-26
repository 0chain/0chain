package diagnostics

import (
	"fmt"
	"net/http"

	"0chain.net/chain"
	metrics "github.com/rcrowley/go-metrics"
)

/*WriteStatistics - write the statistics of the given timer */
func WriteStatistics(w http.ResponseWriter, c *chain.Chain, timer metrics.Timer) {
	scale := func(n float64) float64 {
		return (n / 1000000.0)
	}
	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := timer.Percentiles(percentiles)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Delta</td><td>%v</td></tr>", chain.DELTA)
	fmt.Fprintf(w, "<tr><td>Block Size</td><td>%v</td></tr>", c.BlockSize)
	fmt.Fprintf(w, "<tr><td>Rounds</td><td>%v</td></tr>", c.CurrentRound)
	fmt.Fprintf(w, "<tr><td>Count</td><td>%v</td></tr>", timer.Count())
	fmt.Fprintf(w, "<tr><td>Min</td><td>%.2f</td></tr>", scale(float64(timer.Min())))
	fmt.Fprintf(w, "<tr><td>Mean</td><td>%.2f &plusmn;%.2f</td></tr>", scale(timer.Mean()), scale(timer.StdDev()))
	fmt.Fprintf(w, "<tr><td>Max</td><td>%.2f</td></tr>", scale(float64(timer.Max())))
	for idx, p := range percentiles {
		fmt.Fprintf(w, "<tr><td>%.2f%%</td><td>%.2f</td></tr>", 100*p, scale(pvals[idx]))
	}
	fmt.Fprintf(w, "<tr><td>1-min rate</td><td>%.2f</td></tr>", timer.Rate1())
	fmt.Fprintf(w, "<tr><td>5-min rate</td><td>%.2f</td></tr>", timer.Rate5())
	fmt.Fprintf(w, "<tr><td>15-min rate</td><td>%.2f</td></tr>", timer.Rate15())
	fmt.Fprintf(w, "<tr><td>mean rate</td><td>%.2f</td></tr>", timer.RateMean())
	fmt.Fprintf(w, "</table>")
}
