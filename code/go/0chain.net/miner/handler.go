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
	// DKG diagnostics and recovery handlers (read-only - public access)
	http.HandleFunc("/_diagnostics/dkg/status", common.WithCORS(
		common.UserRateLimit(common.ToJSONResponse(DKGStatusHandler)),
	))
	http.HandleFunc("/_diagnostics/dkg/test_recovery", common.WithCORS(
		common.UserRateLimit(common.ToJSONResponse(DKGTestRecoveryHandler)),
	))
	http.HandleFunc("/_diagnostics/dkg/backups", common.WithCORS(
		common.UserRateLimit(common.ToJSONResponse(DKGListBackupsHandler)),
	))
	// DKG state-modifying handlers (localhost only for security)
	http.HandleFunc("/_diagnostics/dkg/force_recovery", common.WithCORS(
		common.UserRateLimit(LocalhostOnly(common.ToJSONResponse(DKGForceRecoveryHandler))),
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

// DKGStatusResponse contains the current DKG status for diagnostics
type DKGStatusResponse struct {
	CurrentRound    int64             `json:"current_round"`
	LatestMBNumber  int64             `json:"latest_mb_number"`
	LatestMBSR      int64             `json:"latest_mb_starting_round"`
	PrevMBNumber    int64             `json:"prev_mb_number,omitempty"`
	PrevMBSR        int64             `json:"prev_mb_starting_round,omitempty"`
	DKGStatus       map[string]string `json:"dkg_status"`
	SelfNodeKey     string            `json:"self_node_key"`
	ViewChangeRound int64             `json:"view_change_round,omitempty"`
}

// DKGStatusHandler returns the current DKG status
func DKGStatusHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	mc := GetMinerChain()
	c := mc.Chain

	resp := DKGStatusResponse{
		CurrentRound: c.GetCurrentRound(),
		DKGStatus:    make(map[string]string),
		SelfNodeKey:  node.Self.Underlying().GetKey(),
	}

	// Get latest MB
	lfmb := mc.GetLatestFinalizedMagicBlock(ctx)
	if lfmb != nil && lfmb.MagicBlock != nil {
		mb := lfmb.MagicBlock
		resp.LatestMBNumber = mb.MagicBlockNumber
		resp.LatestMBSR = mb.StartingRound

		// Check DKG for current MB
		dkg := c.GetDKGByStartingRound(mb.StartingRound)
		if dkg != nil {
			resp.DKGStatus[fmt.Sprintf("MB%d", mb.MagicBlockNumber)] = fmt.Sprintf(
				"OK (T=%d, shares=%d, SR=%d)", dkg.T, dkg.GetSecretSharesSize(), dkg.StartingRound)
		} else {
			// Try to load from store
			summary, err := LoadDKGSummary(ctx, fmt.Sprintf("%d", mb.MagicBlockNumber))
			if err != nil {
				resp.DKGStatus[fmt.Sprintf("MB%d", mb.MagicBlockNumber)] = fmt.Sprintf("MISSING: %v", err)
			} else if summary.SecretShares == nil || len(summary.SecretShares) < mb.T {
				resp.DKGStatus[fmt.Sprintf("MB%d", mb.MagicBlockNumber)] = fmt.Sprintf(
					"INCOMPLETE: have %d shares, need %d", len(summary.SecretShares), mb.T)
			} else {
				resp.DKGStatus[fmt.Sprintf("MB%d", mb.MagicBlockNumber)] = fmt.Sprintf(
					"IN_STORE (shares=%d, need %d)", len(summary.SecretShares), mb.T)
			}
		}

		// Check previous MB
		prevMB := mc.GetPrevMagicBlockFromMB(mb)
		if prevMB != nil && prevMB.MagicBlockNumber > 0 {
			resp.PrevMBNumber = prevMB.MagicBlockNumber
			resp.PrevMBSR = prevMB.StartingRound

			dkg := c.GetDKGByStartingRound(prevMB.StartingRound)
			if dkg != nil {
				resp.DKGStatus[fmt.Sprintf("MB%d", prevMB.MagicBlockNumber)] = fmt.Sprintf(
					"OK (T=%d, shares=%d, SR=%d)", dkg.T, dkg.GetSecretSharesSize(), dkg.StartingRound)
			} else {
				summary, err := LoadDKGSummary(ctx, fmt.Sprintf("%d", prevMB.MagicBlockNumber))
				if err != nil {
					resp.DKGStatus[fmt.Sprintf("MB%d", prevMB.MagicBlockNumber)] = fmt.Sprintf("MISSING: %v", err)
				} else if summary.SecretShares == nil || len(summary.SecretShares) < prevMB.T {
					resp.DKGStatus[fmt.Sprintf("MB%d", prevMB.MagicBlockNumber)] = fmt.Sprintf(
						"INCOMPLETE: have %d shares, need %d", len(summary.SecretShares), prevMB.T)
				} else {
					resp.DKGStatus[fmt.Sprintf("MB%d", prevMB.MagicBlockNumber)] = fmt.Sprintf(
						"IN_STORE (shares=%d, need %d)", len(summary.SecretShares), prevMB.T)
				}
			}
		}
	}

	return resp, nil
}

// DKGRecoveryTestResponse contains the result of a DKG recovery test
type DKGRecoveryTestResponse struct {
	MBNumber        int64    `json:"mb_number"`
	StartingRound   int64    `json:"starting_round"`
	SharesRecovered int      `json:"shares_recovered"`
	Threshold       int      `json:"threshold"`
	TotalMiners     int      `json:"total_miners"`
	CanRecover      bool     `json:"can_recover"`
	VerifyResult    string   `json:"verify_result"`
	ShareSources    []string `json:"share_sources,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// DKGTestRecoveryHandler tests DKG recovery without applying changes (dry run)
func DKGTestRecoveryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	mc := GetMinerChain()
	results := make([]DKGRecoveryTestResponse, 0)

	lfmb := mc.GetLatestFinalizedMagicBlock(ctx)
	if lfmb == nil || lfmb.MagicBlock == nil {
		return nil, common.NewError("test_recovery", "no latest finalized magic block")
	}

	// Test recovery for current MB
	currentMB := lfmb.MagicBlock
	results = append(results, testRecoveryForMB(ctx, currentMB))

	// Test recovery for previous MB
	prevMB := mc.GetPrevMagicBlockFromMB(currentMB)
	if prevMB != nil && prevMB.MagicBlockNumber > 0 {
		results = append(results, testRecoveryForMB(ctx, prevMB))
	}

	return results, nil
}

func testRecoveryForMB(ctx context.Context, mb *block.MagicBlock) DKGRecoveryTestResponse {
	resp := DKGRecoveryTestResponse{
		MBNumber:      mb.MagicBlockNumber,
		StartingRound: mb.StartingRound,
		Threshold:     mb.T,
		TotalMiners:   mb.N,
	}

	if mb.ShareOrSigns == nil {
		resp.Error = "magic block has no ShareOrSigns"
		return resp
	}

	// Try to recover
	summary, err := RecoverDKGSummaryFromMagicBlock(ctx, mb)
	if err != nil {
		resp.Error = err.Error()
		return resp
	}

	resp.SharesRecovered = len(summary.SecretShares)
	resp.CanRecover = resp.SharesRecovered >= mb.T

	// Collect share sources
	selfKey := node.Self.Underlying().GetKey()
	shares := mb.ShareOrSigns.GetShares()
	for senderKey, sos := range shares {
		if sos != nil && sos.ShareOrSigns != nil {
			if dkgShare, ok := sos.ShareOrSigns[selfKey]; ok && dkgShare != nil && dkgShare.Share != "" {
				resp.ShareSources = append(resp.ShareSources, senderKey[:16]+"...")
			}
		}
	}

	// Verify if possible
	if mb.Mpks != nil {
		if err := VerifyDKGSummary(summary, mb); err != nil {
			resp.VerifyResult = fmt.Sprintf("FAILED: %v", err)
		} else {
			resp.VerifyResult = "PASSED"
		}
	} else {
		resp.VerifyResult = "SKIPPED (no MPKs)"
	}

	return resp
}

// DKGForceRecoveryResponse contains the result of a forced DKG recovery
type DKGForceRecoveryResponse struct {
	Success    bool                      `json:"success"`
	Message    string                    `json:"message"`
	Recovered  []DKGRecoveryTestResponse `json:"recovered"`
	BackupDir  string                    `json:"backup_dir"`
	BackupFile string                    `json:"backup_file,omitempty"`
}

// DKGForceRecoveryHandler forces DKG recovery with backup
func DKGForceRecoveryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	mc := GetMinerChain()

	resp := DKGForceRecoveryResponse{
		BackupDir: "data/dkg_backup",
		Recovered: make([]DKGRecoveryTestResponse, 0),
	}

	if err := mc.ForceRecoverDKG(ctx); err != nil {
		resp.Success = false
		resp.Message = fmt.Sprintf("recovery failed: %v", err)
		return resp, nil
	}

	resp.Success = true
	resp.Message = "DKG recovery completed successfully"

	// Get status after recovery
	lfmb := mc.GetLatestFinalizedMagicBlock(ctx)
	if lfmb != nil && lfmb.MagicBlock != nil {
		resp.Recovered = append(resp.Recovered, testRecoveryForMB(ctx, lfmb.MagicBlock))
		prevMB := mc.GetPrevMagicBlockFromMB(lfmb.MagicBlock)
		if prevMB != nil && prevMB.MagicBlockNumber > 0 {
			resp.Recovered = append(resp.Recovered, testRecoveryForMB(ctx, prevMB))
		}
	}

	return resp, nil
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
