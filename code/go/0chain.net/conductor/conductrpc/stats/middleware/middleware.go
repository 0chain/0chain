package middleware

import (
	"log"
	"net/http"
	"strconv"

	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
)

type (
	// BlockStatsConfigurator contains needed for the BlockStats middleware information.
	BlockStatsConfigurator struct {
		HashKey      string
		Handler      string
		SenderHeader string
	}
)

// BlockStats represents middleware for collecting nodes blocks servers stats.
func BlockStats(handler func(http.ResponseWriter, *http.Request), cfg BlockStatsConfigurator) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !crpc.Client().State().StatsCollectorEnabled {
			handler(w, r)
			return
		}

		roundStr := r.FormValue("round")
		round := 0
		if roundStr != "" {
			var err error
			round, err = strconv.Atoi(roundStr)
			if err != nil {
				log.Panicf("Conductor: error while converting round from string: %v", err)
			}
		}
		ss := &stats.BlockRequest{
			NodeID:   node.Self.ID,
			Hash:     r.FormValue(cfg.HashKey),
			Round:    round,
			Handler:  cfg.Handler,
			SenderID: r.Header.Get(cfg.SenderHeader),
		}
		if err := crpc.Client().AddBlockServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		handler(w, r)
	}
}
