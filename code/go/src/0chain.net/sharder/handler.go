package sharder

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	metrics "github.com/rcrowley/go-metrics"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/block/get", common.ToJSONResponse(BlockHandler))
	http.HandleFunc("/_block_stats", BlockStatsHandler)
}

//BlockHandler - a handler to respond to block queries */
func BlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("block")
	round := r.FormValue("round")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	var roundNumber int64 = -1
	var err error
	var b *block.Block
	if hash == "" {
		if round != "" {
			roundNumber, err = strconv.ParseInt(round, 10, 63)
			if err != nil {
				return nil, err
			}
			// TODO: Get the hash from the round
		} else {
			b = chain.GetServerChain().LatestFinalizedBlock
			if b != nil {
				return chain.GetBlockResponse(b, parts)
			}
		}
	}
	b, err = chain.GetServerChain().GetBlock(ctx, hash)
	if err == nil {
		return chain.GetBlockResponse(b, parts)
	}
	sc := GetSharderChain()
	if roundNumber == -1 {
		if round != "" {
			roundNumber, err = strconv.ParseInt(round, 10, 63)
			if err != nil {
				return nil, err
			}
		} else {
			// TODO: Get the round from the hash
		}
	}
	b, err = sc.GetBlockFromStore(hash, roundNumber)
	if err != nil {
		return nil, err
	}
	return chain.GetBlockResponse(b, parts)
}

/*BlockStatsHandler - a handler to provide block statistics */
func BlockStatsHandler(w http.ResponseWriter, r *http.Request) {
	scale := func(n float64) float64 {
		return (n / 1000000.0)
	}
	timer = metrics.GetOrRegisterTimer("block_time", nil)
	percentiles := []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	pvals := timer.Percentiles(percentiles)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Delta</td><td>%v</td></tr>", chain.DELTA)
	fmt.Fprintf(w, "<tr><td>Count</td><td>%v</td></tr>", timer.Count())
	fmt.Fprintf(w, "<tr><td>Min, Mean (Standard Dev), Max</td><td>%.2f, %.2f (%.2f), %.2f</td></tr>", scale(float64(timer.Min())), scale(timer.Mean()), scale(timer.StdDev()), scale(float64(timer.Max())))
	for idx, p := range percentiles {
		fmt.Fprintf(w, "<tr><td>%.2f%%</td><td>%.2f</td></tr>", 100*p, scale(pvals[idx]))
	}
	fmt.Fprintf(w, "<tr><td>1-min rate</td><td>%.2f</td></tr>", timer.Rate1())
	fmt.Fprintf(w, "<tr><td>5-min rate</td><td>%.2f</td></tr>", timer.Rate5())
	fmt.Fprintf(w, "<tr><td>15-min rate</td><td>%.2f</td></tr>", timer.Rate15())
	fmt.Fprintf(w, "<tr><td>mean rate</td><td>%.2f</td></tr>", timer.RateMean())
	fmt.Fprintf(w, "</table>")
}
