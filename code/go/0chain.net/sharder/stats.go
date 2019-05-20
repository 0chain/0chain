package sharder

import (
	"fmt"
	"net/http"

	metrics "github.com/rcrowley/go-metrics"
)

var BlockSyncTimer metrics.Timer

func init() {
	BlockSyncTimer = metrics.GetOrRegisterTimer("block_sync_timer", nil)
}

//Stats - a struct to store various runtime stats of the chain
type Stats struct {
	ShardedBlocksCount int64
	HealthyRoundNum    int64
	QOSRound           int64
}

const (
	Sync     = "syncing"
	SyncDone = "synced"
)

type SyncStats struct {
	Status          string
	SyncBeginR      int64
	SyncUntilR      int64
	CurrSyncR       int64
	SyncBlocksCount int64
}

func (sc *Chain) WriteBlockSyncStats(w http.ResponseWriter) {
	status := sc.BSyncStats.Status
	if status == Sync {
		fmt.Fprintf(w, "<tr><td>Synced Blocks</td><td class='number'>%v</td></tr>", sc.BSyncStats.SyncBlocksCount)
		fmt.Fprintf(w, "<tr><td>Sync begin</td><td class='number'>%v</td></tr>", sc.BSyncStats.SyncBeginR)
		fmt.Fprintf(w, "<tr><td>Sync until</td><td class='number'>%v</td></tr>", sc.BSyncStats.SyncUntilR)
		fmt.Fprintf(w, "<tr><td>Last Synced</td><td class='number'>%v</td></tr>", sc.BSyncStats.CurrSyncR)
		fmt.Fprintf(w, "<tr><td>Still Sync</td><td class='number'>%v</td></tr>", sc.BSyncStats.SyncUntilR-sc.BSyncStats.CurrSyncR)
	}
}
