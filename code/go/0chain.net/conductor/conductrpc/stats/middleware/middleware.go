package middleware

import (
	"log"
	"net/http"
	"strconv"

	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
)

// BlockStatsMiddleware represents middleware for collecting nodes blocks servers stats.
func BlockStatsMiddleware(handler func(http.ResponseWriter, *http.Request), hashKey, path string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !crpc.Client().State().StatsCollectorEnabled {
			handler(w, r)
			return
		}

		round, err := strconv.Atoi(r.FormValue("round"))
		if err != nil {
			log.Panicf("Conductor: error while converting round from string: %v", err)
		}
		ss := &stats.BlockReport{
			NodeID: node.Self.ID,
			BlockInfo: stats.BlockInfo{
				Hash:  r.FormValue(hashKey),
				Round: round,
			},
			Handler: path,
		}
		if err := crpc.Client().AddBlockServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		handler(w, r)
	}
}
