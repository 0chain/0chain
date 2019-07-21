package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"context"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
	"runtime"
	"time"
)

var HealthCheckDateTimeFormat = "2006-01-02T15:04:05"

type BlockHealthCheckStatus int

const (
	HealthCheckSuccess = iota
	HealthCheckFailure
)

type HealthCheckScan int

const (
	DeepScan HealthCheckScan = iota
	ProximityScan
)

func (e HealthCheckScan) String() string {
	modeNames := []string{"Deep.....", "Proximity"}
	return modeNames[e]
}

type HealthCheckStatus string

const (
	SyncProgress HealthCheckStatus = "syncing"
	SyncHiatus                     = "hiatus"
	SyncDone                       = "synced"
)

type EntityCounters struct {
	Missing       uint64
	RepairSuccess uint64
	RepairFailure uint64
}

type BlockCounters struct {
	CycleIteration int64
	CycleStart     time.Time
	CycleEnd       time.Time
	CycleDuration  time.Duration

	// Sweep Rate for blocks
	SweepCount     int64
	ElapsedSeconds int64
	SweepRate      int64

	HealthCheckInvocations uint64
	HealthCheckSuccess     uint64
	HealthCheckFailure     uint64

	// Entity Counters.
	block        EntityCounters
	blockSummary EntityCounters
	roundSummary EntityCounters
	txnSummary   EntityCounters
}

func (bc *BlockCounters) init() {
	*bc = BlockCounters{}

	bc.CycleStart = time.Now().Truncate(time.Second)
	bc.CycleEnd = time.Time{}
}

type CycleCounters struct {
	ScanMode HealthCheckScan

	current  BlockCounters
	previous BlockCounters
}

func (cc *CycleCounters) transfer() {
	cc.previous = cc.current
}

type CycleBounds struct {
	window       int64
	lowRound     int64
	currentRound int64
	highRound    int64
}

type RangeBounds struct {
	roundLow int64
	roundHigh int64
	roundRange int64
}

func GetRangeBounds(roundEdge int64, roundRange int64) RangeBounds {
	var bounds RangeBounds
	if roundRange > 0 {
		bounds.roundLow = roundEdge
		bounds.roundHigh = roundEdge + roundRange
	} else {
		bounds.roundHigh = roundEdge
		bounds.roundLow = roundEdge + roundRange
	}
	if bounds.roundHigh <= 0 {
		bounds.roundHigh = 1
	}
	if bounds.roundLow <= 0 {
		bounds.roundLow = 1
	}
	bounds.roundRange = bounds.roundHigh - bounds.roundLow + 1
	return bounds
}

type CycleControl struct {
	ScanMode HealthCheckScan
	Status   HealthCheckStatus

	inception time.Time
	bounds    CycleBounds

	CycleCount  int64

	BlockSyncTimer metrics.Timer

	counters CycleCounters
}

func (bss *SyncStats) getCycleControl(scanMode HealthCheckScan) *CycleControl {
	return &bss.cycle[scanMode]
}

type SyncStats struct {
	cycle [2]CycleControl
}

func (sc *Chain) setCycleBounds(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats
	cb := &bss.cycle[scanMode].bounds

	// Clear old bounds
	*cb = CycleBounds{}
	config := &sc.HC_CycleScan[scanMode]
	cb.window = config.Window

	//roundEntity, err := sc.GetMostRecentRoundFromDB(ctx)
	round := sc.GetLatestFinalizedBlock().Round
	cb.highRound = round
	if round == 0 {
		cb.highRound = 1
	}

	// Start from the high round
	cb.currentRound = cb.highRound
	if cb.window == 0 || cb.window > cb.highRound {
		// Cover entire blockchain.
		cb.lowRound = 1
	} else {
		cb.lowRound = cb.highRound - cb.window + 1
	}
}

/*HealthCheckWorker - checks the health for each round*/
func (sc *Chain)HealthCheckSetup(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Update the scan mode.
	cc.ScanMode = scanMode

	cc.BlockSyncTimer = metrics.GetOrRegisterTimer(scanMode.String(), nil)

}

func (sc *Chain) HealthCheckWorker(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get the configuration
	config := &sc.HC_CycleScan[scanMode]

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Wait for the settling period.
	time.Sleep(config.Settle)

	// Setup inception

	cc.inception = time.Now()

	if config.Enabled == false {

		// Scan is disabled. Print event periodically.
		wakeToReport := config.ReportStatus
		for true {
			Logger.Info("HC-CycleHistory",
				zap.String("scan", scanMode.String()),
				zap.Bool("enabled", config.Enabled))
			time.Sleep(wakeToReport)
		}
	}

	// Set the cycle bounds
	sc.setCycleBounds(ctx, scanMode)
	cb := &cc.bounds

	Logger.Info("HC-Init",
		zap.String("mode", scanMode.String()),
		zap.Int64("high", cb.highRound),
		zap.Int64("low", cb.lowRound),
		zap.Int64("current", cb.currentRound),
		zap.Int64("window", cb.window))

	Logger.Info("HC-Init",
		zap.String("mode", scanMode.String()),
		zap.Int64("batch-size", config.BatchSize),
		zap.Int("interval", config.RepeatIntervalMins))

	// Initial setup
	Logger.Info("HC-Init",
		zap.String("mode", scanMode.String()),
		zap.Time("inception", cc.inception))

	// Initialize the health check statistics
	sc.initSyncStats(ctx, scanMode)

	for true {
		select {
		case <-ctx.Done():
			return
		default:
			cc.Status = SyncProgress
			for cb.currentRound = cb.highRound; cb.currentRound >= cb.lowRound; cb.currentRound-- {
				t := time.Now()
				sc.healthCheck(ctx, cb.currentRound, scanMode)

				// Update the duration
				duration := time.Since(t)

				// Update the statistics
				sc.updateSyncStats(ctx, cb.currentRound, duration, scanMode)

				// Schedule other tasks.
				runtime.Gosched()
			}

			// Wait for new work.
			sc.waitForWork(ctx, scanMode)
		}
	}
}

func (sc *Chain) initSyncStats(ctx context.Context, scanMode HealthCheckScan) {

	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Bounds for the round.
	bounds := cc.bounds

	// Update the cycle count.
	cc.CycleCount++

	// Copy current to previous cycle.
	cc.counters.transfer()

	// Clear current cycle counters
	cc.counters.current.init()

	// Log start of new round
	Logger.Info("HC-CycleHistory",
		zap.String("mode", cc.ScanMode.String()),
		zap.Int64("cycle", cc.CycleCount),
		zap.String("event", "start"),
		zap.String("bounds",
			fmt.Sprintf("[%v-%v]", bounds.highRound, bounds.lowRound)),
		zap.Time("start", cc.counters.current.CycleStart.Truncate(time.Second)))
}

func (sc *Chain) updateSyncStats(ctx context.Context, current int64, duration time.Duration, scanMode HealthCheckScan) {

	// var highRound int64
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Update the timer. Common for both scans.
	cc.BlockSyncTimer.Update(duration)

}

func (sc *Chain) waitForWork(ctx context.Context, scanMode HealthCheckScan) {
	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Get the current cycle
	bc := &cc.counters.current

	// Bounds for the round.
	bounds := cc.bounds

	for true {
		// Log end of the current cycle
		bc.CycleEnd = time.Now().Truncate(time.Second)
		bc.CycleDuration = time.Since(bc.CycleStart).Truncate(time.Second)
		bc.ElapsedSeconds = int64(bc.CycleDuration.Seconds())

		// Mark as cycle is in hiatus
		cc.Status = SyncHiatus

		Logger.Info("HC-CycleHistory",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("event", "end"),
			zap.String("bounds",
				fmt.Sprintf("[%v-%v]", bounds.highRound, bounds.lowRound)),
			zap.Time("start", bc.CycleStart.Truncate(time.Second)),
			zap.Time("end", bc.CycleEnd.Truncate(time.Second)),
			zap.Duration("duration", bc.CycleDuration))

		// Calculate the sweep rate
		bc.SweepCount = bounds.highRound - bounds.lowRound + 1

		if bc.ElapsedSeconds > 0 {
			bc.SweepRate = bc.SweepCount / bc.ElapsedSeconds
		}

		Logger.Info("HC-CycleHistory",
			zap.String("mode", cc.ScanMode.String()),
			zap.Int64("cycle", cc.CycleCount),
			zap.String("event", "sweep-rate"),
			zap.Int64("BlocksSweeped", bc.SweepCount),
			zap.Int64("ElapsedSeconds", bc.ElapsedSeconds),
			zap.Int64("SweepRate", bc.SweepRate))

		// End of the cycle. Sleep between cycles.
		config := &sc.HC_CycleScan[scanMode]

		sleepTime := config.RepeatInterval
		wakeToReport := config.ReportStatus
		if wakeToReport > sleepTime {
			wakeToReport = sleepTime
		}

		// Add time to sleep before waking up
		restartCycle := time.Now().Add(sleepTime)
		for ok := true; ok; ok = restartCycle.After(time.Now()) {
			Logger.Info("HC-CycleHistory",
				zap.String("mode", cc.ScanMode.String()),
				zap.Int64("cycle", cc.CycleCount),
				zap.String("event", "hiatus"),
				zap.Time("restart", restartCycle.Truncate(time.Second)))
			time.Sleep(wakeToReport)
		}

		// Time to start a new cycle
		sc.setCycleBounds(ctx, scanMode)
		sc.initSyncStats(ctx, scanMode)
		break;
	}
}

func (sc *Chain) hcUpdateBlockStatus(scanMode HealthCheckScan, status *BlockHealthCheckStatus) {

	bss := sc.BlockSyncStats

	// Get cycle control
	cc := bss.getCycleControl(scanMode)
	current := &cc.counters.current
	current.HealthCheckInvocations++

	switch *status {
	case HealthCheckSuccess:
		current.HealthCheckSuccess++
	case HealthCheckFailure:
		current.HealthCheckFailure++
	}
}

func (sc *Chain) healthCheck(ctx context.Context, rNum int64, scanMode HealthCheckScan) {

	var hcStatus BlockHealthCheckStatus = HealthCheckSuccess

	defer sc.hcUpdateBlockStatus(scanMode, &hcStatus)

	bss := sc.BlockSyncStats
	config := &sc.HC_CycleScan[scanMode]
	// Get cycle control
	cc := bss.getCycleControl(scanMode)

	// Get the current counters.
	current := &cc.counters.current

	var r *round.Round
	var bs *block.BlockSummary
	var b *block.Block

	self := node.GetSelfNode(ctx)

	r, foundRoundSummary := sc.hasRoundSummary(ctx, rNum)
	if foundRoundSummary == false {
		// Update missing round summary
		current.roundSummary.Missing++

		// No round found. Fetch the round summary and round information.
		r = sc.syncRoundSummary(ctx, rNum, -config.BatchSize, scanMode)
		if r == nil {
			current.roundSummary.RepairFailure++
		} else {
			current.roundSummary.RepairSuccess++
		}
	}

	if sc.isValidRound(r) == false {
		// Unable to get the round summary information.
		hcStatus = HealthCheckFailure
		return
	}

	// Obtained valid round. Retrieve blocks.
	bs, foundBlockSummary := sc.hasBlockSummary(ctx, r.BlockHash)
	if foundBlockSummary == false {
		current.blockSummary.Missing++

		// Missing block summary. Sync the blocks
		bs = sc.syncBlockSummary(ctx, r, -config.BatchSize, scanMode)
		if bs != nil {
			current.blockSummary.RepairSuccess++
		} else {
			current.blockSummary.RepairFailure++
		}
	}

	if bs == nil {
		// Unable to retrieve block summary.
		hcStatus = HealthCheckFailure
		return
	}

	// Check for block presence.
	canShard := sc.IsBlockSharderFromHash(bs.Hash, self.Node)

	needTxnSummary := false
	// Check if the sharder has txn_summary
	if bs.NumTxns > 0 {
		count, err := sc.getTxnCountForRound(ctx, bs.Round)
		if err != nil || count != bs.NumTxns {
			needTxnSummary = true
		}
	}
	if needTxnSummary {
		// Missing txn summary. Need to pull from remote sharder.
		current.txnSummary.Missing++
	}

	// The sharder needs txn_summary. Get the block
	b, foundBlock := sc.hasBlock(bs.Hash, r.Number)
	if foundBlock == false {
		if needTxnSummary || canShard {
			// The sharder doesn't have the block.
			// It needs a block either to fix txnsummary or missing block
			// that it should have sharded.
			current.block.Missing++

			b = sc.requestBlock(ctx, r)
			if b == nil {
				HCLogger.Info("HC-MissingObject",
					zap.String("mode", cc.ScanMode.String()),
					zap.Int64("cycle", cc.CycleCount),
					zap.String("object", "Block"),
					zap.Int64("round", r.Number),
					zap.String("hash", r.BlockHash))
				current.block.RepairFailure++
				hcStatus = HealthCheckFailure
				return
			}
		}

		if canShard {
			// The sharder has acquired the block and should save it.
			err := sc.storeBlock(ctx, b)
			if err != nil {
				Logger.Error("HC-DSWriteFailure",
					zap.String("mode", cc.ScanMode.String()),
					zap.Int64("cycle", cc.CycleCount),
					zap.String("object", "block"),
					zap.Int64("round", r.Number),
					zap.Error(err))
				current.block.RepairFailure++
				hcStatus = HealthCheckFailure
				return
			} else {
				current.block.RepairSuccess++
			}
		}
	}

	// Check if the sharder needs to store txn summary
	if needTxnSummary {
		if b == nil {
			Logger.Panic("HC-Assertion",
				zap.String("mode", cc.ScanMode.String()),
				zap.Int64("cycle", cc.CycleCount),
				zap.String("object", "block"),
				zap.Int64("round", r.Number),
				zap.String("Missing block", bs.Hash))
		}

		// The block has transactions and may need to be stored.
		err := sc.storeBlockTransactions(ctx, b)
		if err == nil {
			current.txnSummary.RepairSuccess++
		} else {
			Logger.Error("HC-DSWriteFailure",
				zap.String("mode", cc.ScanMode.String()),
				zap.Int64("cycle", cc.CycleCount),
				zap.String("object", "TransactionSummary"),
				zap.Int64("round", bs.Round),
				zap.Int("txn-count", bs.NumTxns),
				zap.String("block-hash", bs.Hash),
				zap.Error(err))
			current.txnSummary.RepairFailure++
			hcStatus = HealthCheckFailure
			return
		}
	}
	return
}
