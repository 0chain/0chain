package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

const (
	// blockGapScanInterval - how often the repair worker scans for missing block rows.
	blockGapScanInterval = 30 * time.Second
	// blockGapScanSpan - how many recent rounds to scan. Must stay below
	// block.EventsRingSize so every gap found is still replayable from the ring.
	blockGapScanSpan = 900
	// blockGapHeadMargin - newest rounds are skipped; they may still be in flight.
	blockGapHeadMargin = 30
)

// blocksGapRepairWorker periodically re-inserts blocks rows that the events
// pipeline dropped (a block-finalization timeout or a transient Work error
// rolls back the whole events tx and there is no retry). The merged events of
// every finalized round are persisted to the block-events ring (see
// chain.storeEventsFunc) BEFORE the lossy channel handoff, so a dropped round
// can be replayed locally via getBlockEvents while it is still in the ring.
func (edb *EventDb) blocksGapRepairWorker(ctx context.Context,
	getBlockEvents func(round int64) (int64, []Event, error)) {
	ticker := time.NewTicker(blockGapScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			edb.repairMissingBlockRows(getBlockEvents)
		}
	}
}

func (edb *EventDb) repairMissingBlockRows(getBlockEvents func(round int64) (int64, []Event, error)) {
	var gaps []int64
	err := edb.Store.Get().Raw(
		"SELECT s.i FROM generate_series((SELECT max(round) FROM blocks) - ?, (SELECT max(round) FROM blocks) - ?) s(i) "+
			"LEFT JOIN blocks ON blocks.round = s.i WHERE blocks.round IS NULL ORDER BY s.i",
		blockGapScanSpan, blockGapHeadMargin).Scan(&gaps).Error
	if err != nil {
		logging.Logger.Warn("block gap repair - scan failed", zap.Error(err))
		return
	}

	for _, r := range gaps {
		if err := edb.repairBlockRow(r, getBlockEvents); err != nil {
			logging.Logger.Warn("block gap repair - replay failed (external backfill will cover)",
				zap.Int64("round", r), zap.Error(err))
			continue
		}
		logging.Logger.Info("block gap repair - re-inserted dropped block row",
			zap.Int64("round", r))
	}
}

func (edb *EventDb) repairBlockRow(round int64,
	getBlockEvents func(round int64) (int64, []Event, error)) error {
	rd, events, err := getBlockEvents(round)
	if err != nil {
		return err
	}
	if rd != round {
		return fmt.Errorf("round %d evicted from events ring (slot holds %d)", round, rd)
	}

	for i := range events {
		if events[i].Tag != TagFinalizeBlock {
			continue
		}
		// Ring events are JSON round-tripped, so Data is a generic map - re-decode.
		bs, err := json.Marshal(events[i].Data)
		if err != nil {
			return err
		}
		var b Block
		if err := json.Unmarshal(bs, &b); err != nil {
			return err
		}
		if err := edb.addOrUpdateBlock(b); err != nil {
			return err
		}
		return edb.updateMinerBlocksFinalised(b.MinerID)
	}
	return fmt.Errorf("no finalize-block event in ring for round %d", round)
}
