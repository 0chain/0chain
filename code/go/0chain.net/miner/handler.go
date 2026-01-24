package miner

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/config"
	"0chain.net/smartcontract/dbs/event"
)

// LocalhostOnly wraps a handler to only allow requests from localhost
func LocalhostOnly(handler common.ReqRespHandlerf) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		remoteAddr := r.RemoteAddr

		// Check if request is from localhost
		isLocalhost := false
		for _, local := range []string{"localhost", "127.0.0.1", "[::1]", "::1"} {
			if strings.HasPrefix(host, local) || strings.HasPrefix(remoteAddr, local) ||
				strings.HasPrefix(remoteAddr, "127.0.0.1") || strings.HasPrefix(remoteAddr, "[::1]") {
				isLocalhost = true
				break
			}
		}

		if !isLocalhost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error": "This endpoint is only accessible from localhost"}`))
			return
		}

		handler(w, r)
	}
}

/*SetupHandlers - setup miner handlers */
func SetupHandlers() {
	http.HandleFunc("/v1/chain/get/stats", common.WithCORS(
		common.UserRateLimit(common.ToJSONResponse(ChainStatsHandler)),
	))
	http.HandleFunc("/_chain_stats", common.WithCORS(
		common.UserRateLimit(ChainStatsWriter),
	))
	http.HandleFunc("/v1/miner/get/stats", common.WithCORS(
		common.UserRateLimit(common.ToJSONResponse(MinerStatsHandler)),
	))
	http.HandleFunc("/_txn_stats", common.WithCORS(
		common.UserRateLimit(TxnStatsWriter),
	))
	// DKG backup/restore handlers
	http.HandleFunc("/_diagnostics/dkg/backups", common.WithCORS(
		common.UserRateLimit(common.ToJSONResponse(DKGListBackupsHandler)),
	))
	http.HandleFunc("/_diagnostics/dkg/restore", common.WithCORS(
		common.UserRateLimit(LocalhostOnly(common.ToJSONResponse(DKGRestoreHandler))),
	))
}

// swagger:route GET /v1/chain/get/stats miner GetChainStats
// Get chain stats.
// Retrieves the statistics related to the chain progress. No parameters needed.
//
// responses:
//  200: ChainStats
//  500:

func ChainStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetMinerChain().Chain
	return diagnostics.GetStatistics(c, chain.SteadyStateFinalizationTimer, 1000000.0), nil
}

// ChainStatsWriter - display the current chain stats
func ChainStatsWriter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := GetMinerChain().Chain
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Underlying()
	fmt.Fprintf(w, "<h2>%v - %v</h2>", self.GetPseudoName(), self.Description)
	fmt.Fprintf(w, "<br>")

	fmt.Fprintf(w, "<table>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Configuration <a href='v1/config/get'>...</a></h3>")
	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Current Status</h3>")
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Finalization Statistics (Steady state)</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Block Finalization Statistics (Start to Finish)</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td colspan='2'>")
	fmt.Fprintf(w, "<p>Steady state block finalization time = block generation + block processing + network time (1*large message + 2*small message)</p>")
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Txn Finalization Statistics (Start to Finish)</h3>")
	if config.Development() {
		diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTxnTimer, 1000000.0)
	} else {
		fmt.Fprintf(w, "Available only in development mode")
	}
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Finalization Lag Statistics</h3>")
	diagnostics.WriteHistogramStatistics(w, c, chain.FinalizationLagMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Kafka Event Push Latency Statistics (in milliseconds)</h3>")
	diagnostics.WriteHistogramStatistics(w, c, event.KafkaEventPushLatencyMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Generation Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, bgTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Round Block Generation Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, rbgTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Processing Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, bpTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Block Verification Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, btvTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Txns Statistics</h3>")
	diagnostics.WriteHistogramStatistics(w, c, bsHistogram)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Smart Contract Execution Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.SmartContractExecutionTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>State Save Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, block.StateSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>State Change Statistics</h3>")
	diagnostics.WriteHistogramStatistics(w, c, block.StateChangeSizeMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>State Prune Update Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneUpdateTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>State Prune Delete Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneDeleteTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>RRS Generation Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, vrfTimer, 1000000.0)
	if c.GetPruneStats() != nil {
		fmt.Fprintf(w, "</td><td valign='top'>")
		fmt.Fprintf(w, "<h3>Prune Stats</h3>")
		diagnostics.WritePruneStats(w, c.GetPruneStats())
	}
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "</table>")
}

// swagger:route GET /v1/miner/get/stats miner GetMinerStats
// Get Miner Stats.
// Retrieves the statistics related to the miner progress. No parameters needed.
//
// responses:
//
//	200: ExploreStats
func MinerStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetMinerChain().Chain
	var total int64
	ms := node.Self.Underlying().ProtocolStats.(*chain.MinerStats)
	for i := 0; i < c.GetGeneratorsNum(); i++ {
		total += ms.FinalizationCountByRank[i]
	}
	cr := c.GetRound(c.GetCurrentRound())
	rtoc := c.GetRoundTimeoutCount()
	if cr != nil {
		rtoc = int64(cr.GetTimeoutCount())
	}
	networkTimes := make(map[string]time.Duration)
	mb := c.GetCurrentMagicBlock()
	for k, v := range mb.Miners.CopyNodesMap() {
		networkTimes[k] = v.Info.MinersMedianNetworkTime
	}
	for k, v := range mb.Sharders.CopyNodesMap() {
		networkTimes[k] = v.Info.MinersMedianNetworkTime
	}

	return ExplorerStats{BlockFinality: chain.SteadyStateFinalizationTimer.Mean() / 1000000.0,
		LastFinalizedRound: c.GetLatestFinalizedBlock().Round,
		BlocksFinalized:    total,
		StateHealth:        node.Self.Underlying().Info.GetStateMissingNodes(),
		CurrentRound:       c.GetCurrentRound(),
		RoundTimeout:       rtoc,
		Timeouts:           c.RoundTimeoutsCount,
		AverageBlockSize:   node.Self.Underlying().Info.AvgBlockTxns,
		NetworkTime:        networkTimes,
	}, nil
}

// DKGBackupsResponse lists available DKG backups
type DKGBackupsResponse struct {
	BackupDir string   `json:"backup_dir"`
	Backups   []string `json:"backups"`
	Count     int      `json:"count"`
}

// DKGListBackupsHandler lists available DKG backup files
func DKGListBackupsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	backups, err := ListDKGBackups("")
	if err != nil {
		return nil, err
	}

	return DKGBackupsResponse{
		BackupDir: "data/dkg_backup",
		Backups:   backups,
		Count:     len(backups),
	}, nil
}

// DKGRestoreResponse contains the result of a restore operation
type DKGRestoreResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	BackupFile string `json:"backup_file"`
}

// DKGRestoreHandler restores DKG from a backup file
// Usage: /_diagnostics/dkg/restore?file=data/dkg_backup/dkg_summary_19_20260123_120000.json
func DKGRestoreHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	backupFile := r.URL.Query().Get("file")
	if backupFile == "" {
		return nil, common.NewError("restore", "missing 'file' parameter")
	}

	resp := DKGRestoreResponse{
		BackupFile: backupFile,
	}

	if err := RestoreDKGFromBackup(ctx, backupFile); err != nil {
		resp.Success = false
		resp.Message = fmt.Sprintf("restore failed: %v", err)
		return resp, nil
	}

	resp.Success = true
	resp.Message = "DKG restored successfully from backup"
	return resp, nil
}

// TxnStatsWriter - display the current txn stats
func TxnStatsWriter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := GetMinerChain().Chain
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Underlying()
	fmt.Fprintf(w, "<h2>%v - %v</h2>", self.GetPseudoName(), self.Description)
	fmt.Fprintf(w, "<br>")

	// find, missed := util.CacheStats()
	hits, miss := c.GetStateCache().Stats()
	fmt.Fprintf(w, "<h3>MPT cache hits/missed: %v/%v</h3>", hits, miss)
	fmt.Fprintf(w, "<br>")

	fmt.Fprintf(w, "<table>")

	count := 0

	for txnFunc, txnTimer := range chain.StartToFinalizeTxnTypeTimer {
		if count%3 == 0 {
			fmt.Fprintf(w, "<tr><td>")
		} else {
			fmt.Fprintf(w, "</td><td valign='top'>")
		}

		fmt.Fprintf(w, "<h3>%v</h3>", txnFunc)
		diagnostics.WriteTimerStatistics(w, c, txnTimer, 1000000.0)

		if count%3 == 2 {
			fmt.Fprintf(w, "</tr>")
		}

		count++
	}

	fmt.Fprintf(w, "</table>")
}
