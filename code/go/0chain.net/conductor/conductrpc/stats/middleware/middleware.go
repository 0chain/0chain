package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/core/datastore"
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
		roundNum := 0
		if roundStr != "" {
			var err error
			roundNum, err = strconv.Atoi(roundStr)
			if err != nil {
				log.Panicf("Conductor: error while converting round from string: %v", err)
			}
		}
		ss := &stats.BlockRequest{
			NodeID:   node.Self.ID,
			Hash:     r.FormValue(cfg.HashKey),
			Round:    roundNum,
			Handler:  cfg.Handler,
			SenderID: r.Header.Get(cfg.SenderHeader),
		}
		if err := crpc.Client().AddBlockServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		handler(w, r)
	}
}

// VRFSStats represents middleware for datastore.JSONEntityReqResponderF handlers.
// Collects vrfs requests stats.
func VRFSStats(handler datastore.JSONEntityReqResponderF) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		if !crpc.Client().State().StatsCollectorEnabled {
			return handler(ctx, entity)
		}

		vrfs, ok := entity.(*round.VRFShare)
		if !ok {
			log.Panicf("Conductor: unexpected entity type is provided")
		}

		ss := &stats.VRFSRequest{
			NodeID:   node.Self.ID,
			Round:    vrfs.Round,
			SenderID: node.GetSender(ctx).GetKey(),
		}
		if err := crpc.Client().AddVRFSServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		return handler(ctx, entity)
	}
}
