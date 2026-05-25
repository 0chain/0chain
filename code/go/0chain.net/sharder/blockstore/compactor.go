package blockstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

const (
	compactInterval  = 10 * time.Minute
	compactBatchSize = blocksPerPack // 10,000
)

// compactor packs loose block files into pack files ordered by round.
// It queries the sharder's postgres events_db to get the round→hash mapping,
// then packs blocks in sequential round order.
type compactor struct {
	loosePath    string // basePath where loose files live (hash-tree dirs)
	packsPath    string // basePath/packs/
	manifest     *packManifest
	lastPackedRd int64 // highest round already packed
	running      int32
	db           *sql.DB // connection to events_db
}

func newCompactor(loosePath, packsPath string, manifest *packManifest, db *sql.DB) *compactor {
	c := &compactor{
		loosePath: loosePath,
		packsPath: packsPath,
		manifest:  manifest,
		db:        db,
	}
	// Determine last packed round from existing pack file names
	entries, _ := os.ReadDir(packsPath)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".pack") {
			var lo, hi int64
			if _, err := fmt.Sscanf(e.Name(), "rounds_%d_%d.pack", &lo, &hi); err == nil {
				if hi > c.lastPackedRd {
					c.lastPackedRd = hi
				}
			}
		}
	}
	return c
}

// start begins the background compaction loop.
func (c *compactor) start(ctx context.Context) {
	if c.db == nil {
		logging.Logger.Warn("compactor: no db connection, background compaction disabled")
		return
	}

	// If bulk inode-sorted packs exist, skip ahead to near the latest round
	// to avoid scanning millions of already-packed blocks.
	if c.manifest.packCount() > 0 {
		var maxRound int64
		err := c.db.QueryRow("SELECT COALESCE(MAX(round), 0) FROM blocks").Scan(&maxRound)
		if err == nil && maxRound > 0 && c.lastPackedRd < maxRound-20000 {
			c.lastPackedRd = maxRound - 20000
			logging.Logger.Info("compactor: skipped to recent rounds (bulk packs exist)",
				zap.Int64("last_packed_round", c.lastPackedRd),
				zap.Int64("max_round", maxRound))
		}
	}

	logging.Logger.Info("compactor started",
		zap.Int64("last_packed_round", c.lastPackedRd),
		zap.Int("existing_packs", c.manifest.packCount()))

	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				n, err := c.compact()
				if err != nil {
					logging.Logger.Error("compaction error", zap.Error(err))
				} else if n > 0 {
					logging.Logger.Info("compaction complete",
						zap.Int("blocks_packed", n),
						zap.Int64("last_packed_round", c.lastPackedRd),
						zap.Int("total_packs", c.manifest.packCount()))
				}
				timer.Reset(compactInterval)
			}
		}
	}()
}

// compact runs one compaction pass. Returns number of blocks packed.
func (c *compactor) compact() (int, error) {
	if !atomic.CompareAndSwapInt32(&c.running, 0, 1) {
		return 0, nil
	}
	defer atomic.StoreInt32(&c.running, 0)

	totalPacked := 0
	for {
		n, err := c.compactNextBatch()
		if err != nil {
			return totalPacked, err
		}
		if n == 0 {
			break
		}
		totalPacked += n
	}
	return totalPacked, nil
}

// blockRound is a round→hash pair from the database.
type blockRound struct {
	round int64
	hash  string
}

// compactNextBatch queries the next batch of blocks by round from postgres
// and packs them into a single pack file.
func (c *compactor) compactNextBatch() (int, error) {
	if c.db == nil {
		return 0, nil
	}

	// Query next N blocks after lastPackedRd, ordered by round
	rows, err := c.db.Query(
		"SELECT round, hash FROM blocks WHERE round > $1 ORDER BY round ASC LIMIT $2",
		c.lastPackedRd, compactBatchSize)
	if err != nil {
		return 0, fmt.Errorf("query blocks: %w", err)
	}
	defer rows.Close()

	var blocks []blockRound
	for rows.Next() {
		var br blockRound
		if err := rows.Scan(&br.round, &br.hash); err != nil {
			return 0, fmt.Errorf("scan block: %w", err)
		}
		blocks = append(blocks, br)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate blocks: %w", err)
	}

	if len(blocks) < compactBatchSize {
		// Not enough blocks for a full pack — wait for more
		return 0, nil
	}

	loRound := blocks[0].round
	hiRound := blocks[len(blocks)-1].round
	packPath := filepath.Join(c.packsPath, fmt.Sprintf("rounds_%09d_%09d.pack", loRound, hiRound))

	pw, err := newPackWriter(packPath)
	if err != nil {
		return 0, fmt.Errorf("create pack writer: %w", err)
	}

	packed := 0
	for _, br := range blocks {
		bp, err := getBlockFilePath(br.hash)
		if err != nil {
			continue
		}
		loosePath := filepath.Join(c.loosePath, bp)
		data, err := readRawBlock(loosePath)
		if err != nil {
			// Block might already be in a pack or missing from disk
			logging.Logger.Debug("skip missing loose block",
				zap.Int64("round", br.round), zap.String("hash", br.hash))
			continue
		}
		if err := pw.addBlock(br.hash, data); err != nil {
			pw.abort()
			return packed, fmt.Errorf("add block to pack: %w", err)
		}
		packed++
	}

	if packed == 0 {
		pw.abort()
		// Still advance lastPackedRd so we don't query the same range
		c.lastPackedRd = hiRound
		return 0, nil
	}

	if err := pw.finish(packPath); err != nil {
		return 0, fmt.Errorf("finish pack: %w", err)
	}

	pr, err := openPackIndex(packPath)
	if err != nil {
		return 0, fmt.Errorf("verify pack: %w", err)
	}

	c.manifest.addPack(packPath, pr)
	c.lastPackedRd = hiRound

	logging.Logger.Info("packed blocks by round",
		zap.Int64("round_lo", loRound),
		zap.Int64("round_hi", hiRound),
		zap.Int("block_count", packed))

	return packed, nil
}

// extractHashFromPath reconstructs the block hash from its file path.
// Path format: basePath/<h1>/<h2>/<h3>/<h4>/<h5>/<rest>.dat.zlib
// Hash = h1 + h2 + h3 + h4 + h5 + rest
func extractHashFromPath(basePath, fullPath string) string {
	rel, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return ""
	}

	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) < subDirs+1 {
		return ""
	}

	hash := ""
	for i := 0; i < subDirs; i++ {
		if len(parts[i]) != 1 {
			return ""
		}
		hash += parts[i]
	}

	filename := parts[subDirs]
	rest := strings.TrimSuffix(filename, "."+extension)
	hash += rest

	return hash
}

// cleanEmptyParents removes empty directories up to the base path.
func cleanEmptyParents(filePath, basePath string) {
	dir := filepath.Dir(filePath)
	for dir != basePath && dir != "." && dir != "/" {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
