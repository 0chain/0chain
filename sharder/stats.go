package sharder

import (
	"fmt"
	"net/http"
	"time"

	"0chain.net/chaincore/diagnostics"
	"github.com/rcrowley/go-metrics"
)

// BlockSyncTimer -
var BlockSyncTimer metrics.Timer

func init() {
	BlockSyncTimer = metrics.GetOrRegisterTimer("block_sync_timer", nil)
}

//Stats - a struct to store various runtime stats of the chain
type Stats struct {
	ShardedBlocksCount int64

	QOSRound int64
	// Repair block count as part of healthcheck
	RepairBlocksCount   int64
	RepairBlocksFailure int64
}

// WriteHealthCheckConfiguration -
func (sc *Chain) WriteHealthCheckConfiguration(w http.ResponseWriter, scan HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scan)
	bounds := &cc.bounds

	_ = cc

	// Get health check config
	cycleScan := sc.HCCycleScan()
	config := &cycleScan[scan]

	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Tunables</td></tr>")
	fmt.Fprintf(w, "<tr><td>Scan Enabled</td><td class='string'>%v</td></tr>",
		config.Enabled)
	fmt.Fprintf(w, "<tr><td>Repeat Interval (mins)</td><td class='string'>%v</td></tr>",
		config.RepeatInterval)
	fmt.Fprintf(w, "<tr><td>Batch Size</td><td class='string'>%v</td></tr>", config.BatchSize)

	var window string
	if config.Window == 0 {
		window = "Entire BlockChain"
	} else {
		window = fmt.Sprintf("%v", config.Window)
	}

	fmt.Fprintf(w, "<tr><td>Scan Window Size</td><td class='string'>%v</td></tr>", window)

	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Invocation History</td></tr>")
	fmt.Fprintf(w, "<tr><td>Inception</td><td class='string'>%v</td></tr>",
		cc.inception.Format(HealthCheckDateTimeFormat))
	fmt.Fprintf(w, "<tr><td>Repeat RepeatInterval (mins)</td><td class='string'>%v</td></tr>",
		config.RepeatInterval)

	fmt.Fprintf(w, "<tr><td>Cycle Count</td><td class='string'>%v</td></tr>", cc.CycleCount)

	fmt.Fprintf(w, "<tr><td class='sheader' colspan=2'>Cycle Bounds</td></tr>")

	fmt.Fprintf(w, "<tr><td>High Limit</td><td class='string'>%v</td></tr>", bounds.highRound)
	fmt.Fprintf(w, "<tr><td>Low Limit</td><td class='string'>%v</td></tr>", bounds.lowRound)
	fmt.Fprintf(w, "<tr><td>Current</td><td class='string'>%v</td></tr>", bounds.currentRound)
	var pendingCount int64
	if bounds.currentRound > bounds.lowRound {
		pendingCount = bounds.currentRound - bounds.lowRound
	}
	fmt.Fprintf(w, "<tr><td>Pending</td><td class='string'>%v</td></tr>", pendingCount)
	fmt.Fprintf(w, "</table>")

}

// WriteHealthCheckBlockSummary -
func (sc *Chain) WriteHealthCheckBlockSummary(w http.ResponseWriter, scan HealthCheckScan) {
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scan)
	current := &cc.counters.current
	previous := &cc.counters.previous
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr>"+
		"<td class='sheader' colspan=1'>Invocation History</td>"+
		"<td class='sheader' colspan=1'>Current</td>"+
		"<td class='sheader' colspan=1'>Previous</td>"+
		"</tr>")
	var previousStart, currentStart string
	var previousElapsed, currentElapsed string
	var previousStatus, currentStatus HealthCheckStatus
	roundUnit := time.Minute
	if scan == ProximityScan {
		roundUnit = time.Second
	}
	if previous.CycleStart.IsZero() {
		previousStart = "n/a"
		previousElapsed = "n/a"
		previousStatus = "n/a"
	} else {
		previousStart = previous.CycleStart.Format(HealthCheckDateTimeFormat)
		previousElapsed = previous.CycleDuration.Round(roundUnit).String()
		previousStatus = SyncDone
	}
	if current.CycleStart.IsZero() {
		currentStart = "n/a"
		currentElapsed = "n/a"
		currentStatus = "n/a"
	} else {
		currentStart = current.CycleStart.Format(HealthCheckDateTimeFormat)
		switch cc.Status {
		case SyncHiatus:
			currentElapsed = current.CycleDuration.Round(roundUnit).String()
		case SyncProgress:
			currentElapsed = time.Since(current.CycleStart).Round(roundUnit).String()
		}
		currentStatus = cc.Status
	}

	fmt.Fprintf(w, "<tr>"+
		"<td>Status</td>"+
		"<td class='string'>%v</td>"+
		"<td class='string'>%v</td></tr>",
		currentStatus,
		previousStatus)
	fmt.Fprintf(w, "<tr>"+
		"<td>Start</td>"+
		"<td class='string'>%v</td>"+
		"<td class='string'>%v</td></tr>",
		currentStart,
		previousStart)
	fmt.Fprintf(w, "<tr>"+
		"<td>Elapsed</td>"+
		"<td class='string'>%v</td>"+
		"<td class='string'>%v</td></tr>",
		currentElapsed,
		previousElapsed)
	fmt.Fprintf(w, "<tr>"+
		"<td class='sheader' colspan=3'>HealthCheck Invocation Status</td>"+
		"</tr>")
	fmt.Fprintf(w, "<tr><td>Invocation Count</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.HealthCheckInvocations, previous.HealthCheckInvocations)

	fmt.Fprintf(w, "<tr><td>Success</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.HealthCheckSuccess, previous.HealthCheckSuccess)

	fmt.Fprintf(w, "<tr><td>Failures</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.HealthCheckFailure, previous.HealthCheckFailure)

	fmt.Fprintf(w, "<tr></tr>")

	fmt.Fprintf(w, "<tr>"+
		"<td class='sheader' colspan=3'>Round Summary</td>"+
		"</tr>")
	fmt.Fprintf(w, "<tr><td>Missing</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.roundSummary.Missing, previous.roundSummary.Missing)
	fmt.Fprintf(w, "<tr><td>Repaired</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.roundSummary.RepairSuccess, previous.roundSummary.RepairSuccess)
	fmt.Fprintf(w, "<tr><td>Failed</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.roundSummary.RepairFailure, previous.roundSummary.RepairFailure)
	fmt.Fprintf(w, "<tr>"+
		"<td class='sheader' colspan=3'>Block Summary</td>"+
		"</tr>")
	fmt.Fprintf(w, "<tr><td>Missing</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.blockSummary.Missing, previous.blockSummary.Missing)
	fmt.Fprintf(w, "<tr><td>Repaired</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.blockSummary.RepairSuccess, previous.blockSummary.RepairSuccess)
	fmt.Fprintf(w, "<tr><td>Failed</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.blockSummary.RepairFailure, previous.blockSummary.RepairFailure)
	fmt.Fprintf(w, "<tr>"+
		"<td class='sheader' colspan=3'>Transaction Summary</td>"+
		"</tr>")
	fmt.Fprintf(w, "<tr><td>Missing</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.txnSummary.Missing, previous.txnSummary.Missing)
	fmt.Fprintf(w, "<tr><td>Repaired</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.txnSummary.RepairSuccess, previous.txnSummary.RepairSuccess)
	fmt.Fprintf(w, "<tr><td>Failed</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.txnSummary.RepairFailure, previous.txnSummary.RepairFailure)
	fmt.Fprintf(w, "<tr>"+
		"<td class='sheader' colspan=3'>Sharder Stored Blocks </td>"+
		"</tr>")
	fmt.Fprintf(w, "<tr><td>Missing</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.block.Missing, previous.block.Missing)
	fmt.Fprintf(w, "<tr><td>Repaired</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.block.RepairSuccess, previous.block.RepairSuccess)
	fmt.Fprintf(w, "<tr><td>Failed</td>"+
		"<td class='string'>%v</td><td class='string'>%v</td></tr>",
		current.block.RepairFailure, previous.block.RepairFailure)
	fmt.Fprintf(w, "</table>")
}

// WriteBlockSyncStatistics -
func (sc *Chain) WriteBlockSyncStatistics(w http.ResponseWriter, scan HealthCheckScan) {
	bss := sc.BlockSyncStats
	// Get cycle control
	cc := bss.getCycleControl(scan)
	diagnostics.WriteTimerStatistics(w, sc.Chain, cc.BlockSyncTimer, 1000000.0)
}

// swagger:model ExplorerStats
type ExplorerStats struct {
	LastFinalizedRound     int64   `json:"last_finalized_round"`
	StateHealth            int64   `json:"state_health"`
	AverageBlockSize       int     `json:"average_block_size"`
	PrevInvocationCount    uint64  `json:"pervious_invocation_count"`
	PrevInvocationScanTime string  `json:"previous_incovcation_scan_time"`
	MeanScanBlockStatsTime float64 `json:"mean_scan_block_stats_time"`
}

func (sc *Chain) WriteMinioStats(w http.ResponseWriter) {
	fmt.Fprintf(w, "<table width='100%%'>")
	fmt.Fprintf(w, "<tr><th class='sheader' colspan='2'>Minio Stats</th></tr>")
	fmt.Fprintf(w, "<tr><td>Total Rounds processed</td><td>%d</td></tr>", sc.TieringStats.TotalBlocksUploaded)
	fmt.Fprintf(w, "<tr><td>Last Round processed</td><td>%d</td></tr>", sc.TieringStats.LastRoundUploaded)
	fmt.Fprintf(w, "<tr><td>Last Upload time</td class='string'><td>%v</td></tr>", sc.TieringStats.LastUploadTime.Format(HealthCheckDateTimeFormat))
	fmt.Fprintf(w, "</table>")
}
