package smartcontractinterface

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/rcrowley/go-metrics"
)

func (sc *SmartContract) HandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	response := `<!DOCTYPE html><html><body>`
	response += PrintCSS()
	var keys []string
	for k := range sc.SmartContractExecutionStats {
		keys = append(keys, k)
	}
	response += fmt.Sprintf("<table width='100%%'>")
	sort.Strings(keys)
	idx := 0
	for _, k := range keys {
		if idx%2 == 0 {
			response += "<tr><td>"
		} else if idx%2 == 1 {
			response += "</td><td>"
		}

		// response += fmt.Sprintf("<tr><td>")
		response += fmt.Sprintf("<h2>%v</h2>", k)
		switch stats := sc.SmartContractExecutionStats[k].(type) {
		case metrics.Histogram:
			response += WriteHistogramStatisticsWithoutChain(stats)
		case metrics.Timer:
			response += WriteTimerStatisticsWithoutChain(stats, 1000000.0)
		case metrics.Counter:
			response += WriteCounterStatisticsWithoutChain(stats)
		default:
			response += "This is wrong. You should not be seeing this"
		}
		// response += WriteTimerStatisticsWithoutChain(sc.SmartContractExecutionTimer[k], 1000000.0)
		if idx%2 == 1 {
			response += "</td></tr>"
		}
		idx++
	}
	response += `</body></html>`
	return response, nil
}

// WriteCounterStatisticsWithoutChain writes the statistics of the given counter.
func WriteCounterStatisticsWithoutChain(counter metrics.Counter) (resp string) {
	return fmt.Sprintf(`
	<table width='100%%'>
		<tr><td>Count</td><td>%v</td></tr>
	</table>
`, counter.Count())
}

/*WriteTimerStatistics - write the statistics of the given timer */
func WriteTimerStatisticsWithoutChain(timer metrics.Timer, scaleBy float64) string {
	scale := func(n float64) float64 {
		return (n / scaleBy)
	}
	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := timer.Percentiles(percentiles)
	response := fmt.Sprintf(`<table width='100%%'>`)
	response += `<tr><td class='sheader' colspan=2'>Metrics</td></tr>`
	response += fmt.Sprintf(`<tr><td>Count</td><td>%v</td></tr>`, timer.Count())
	response += `<tr><td class='sheader' colspan='2'>Time taken</td></tr>`
	response += fmt.Sprintf(`<tr><td>Min</td><td>%.2f ms</td></tr>`, scale(float64(timer.Min())))
	response += fmt.Sprintf(`<tr><td>Mean</td><td>%.2f &plusmn;%.2f ms</td></tr>`, scale(timer.Mean()), scale(timer.StdDev()))
	response += fmt.Sprintf(`<tr><td>Max</td><td>%.2f ms</td></tr>`, scale(float64(timer.Max())))
	for idx, p := range percentiles {
		response += fmt.Sprintf(`<tr><td>%.2f%%</td><td>%.2f ms</td></tr>`, 100*p, scale(pvals[idx]))
	}
	response += `<tr><td class='sheader' colspan='2'>Rate per second</td></tr>`
	response += fmt.Sprintf(`<tr><td>Last 1-min rate</td><td>%.2f</td></tr>`, timer.Rate1())
	response += fmt.Sprintf(`<tr><td>Last 5-min rate</td><td>%.2f</td></tr>`, timer.Rate5())
	response += fmt.Sprintf(`<tr><td>Last 15-min rate</td><td>%.2f</td></tr>`, timer.Rate15())
	response += fmt.Sprintf(`<tr><td>Overall mean rate</td><td>%.2f</td></tr>`, timer.RateMean())
	response += `</table>`
	return response
}

/*WriteTimerStatistics - write the statistics of the given timer */
func WriteHistogramStatisticsWithoutChain(metric metrics.Histogram) string {
	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := metric.Percentiles(percentiles)
	response := fmt.Sprintf("<table width='100%%'>")
	response += "<tr><td class='sheader' colspan=2'>Metrics</td></tr>"
	response += fmt.Sprintf("<tr><td>Count</td><td>%v</td></tr>", metric.Count())
	response += "<tr><td class='sheader' colspan='2'>Metric Value</td></tr>"
	response += fmt.Sprintf("<tr><td>Min</td><td>%.2f</td></tr>", float64(metric.Min()))
	response += fmt.Sprintf("<tr><td>Mean</td><td>%.2f &plusmn;%.2f</td></tr>", metric.Mean(), metric.StdDev())
	response += fmt.Sprintf("<tr><td>Max</td><td>%.2f</td></tr>", float64(metric.Max()))
	for idx, p := range percentiles {
		response += fmt.Sprintf("<tr><td>%.2f%%</td><td>%.2f</td></tr>", 100*p, pvals[idx])
	}
	response += "</table>"
	return response
}

func PrintCSS() string {
	response := "<style>\n"
	response += ".number { text-align: right; }\n"
	response += ".menu li { list-style-type: none; }\n"
	response += "table, td, th { border: 1px solid black;  border-collapse: collapse;}\n"
	response += "tr.header { background-color: #E0E0E0;  }\n"
	response += ".inactive { background-color: #F44336; }\n"
	response += ".warning { background-color: #FFEB3B; }\n"
	response += ".optimal { color: #1B5E20; }\n"
	response += ".slow { font-style: italic; }\n"
	response += ".bold {font-weight:bold;}"
	response += "</style>"
	return response
}
