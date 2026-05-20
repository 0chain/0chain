# Block Pack Compaction: Architecture & Operations

## Problem

Sharder block storage uses one file per block, keyed by SHA256 hash across a 5-level directory tree (`basePath/<h1>/<h2>/<h3>/<h4>/<h5>/<rest>.dat.zlib`). With 150M+ blocks, this means 150M+ small files (3-13KB each, ~1.35TB total) scattered across ~1M directories.

**Impact:**
- rclone sync between servers: days-weeks (per-file overhead × millions of files)
- HDD random reads: ~100 IOPS = ~2-3 blocks/sec for round-ordered access
- inode exhaustion risk on some filesystems

## Solution: Pack Files

Pack 10,000 blocks into a single file (~74-90MB). 150M blocks = ~15,000 pack files.

### Pack File Format

```
[data section: concatenated length-prefixed block blobs]
[index section: sorted entries of {hash[64], offset[8], length[4]}]
[trailer: indexOffset[8] + indexCount[4] + magic("BPAK")[4]]
```

- Trailer at end → file written sequentially (append blocks, then index, then trailer)
- Index sorted by hash → binary search for O(log N) lookup
- Block data stored as-is (original zlib-compressed msgpack bytes)
- Index entry: 76 bytes × 10K = 760KB per pack

### Sharder Read Path (Production)

```
Read(hash):
  1. SSD cache         → hit? return (unchanged from current)
  2. Pack file lookup  → binary search in-memory manifest → binary search pack index → pread
  3. Loose file        → fallback to old hash-tree (for unpacked blocks)
```

**In-memory manifest**: sorted list of `{packPath, minHash, maxHash}` loaded at startup. 15K packs × 68 bytes = ~1MB. Binary search finds candidate pack(s), then the pack's index (760KB) is loaded on demand and cached.

### Write Path (Production)

Unchanged. New blocks written as loose files. Background compactor packs them periodically.

---

## Two Compaction Modes

### 1. Historical Blocks (One-Time Migration)

For the existing 150M+ blocks already on disk. Uses the standalone `blockcompact` tool with cache warming for maximum throughput.

**Why it's different**: Historical blocks are scattered across millions of directories on HDD. Reading them in any order requires random seeks (~100 IOPS). The cache-warm technique loads the entire directory tree into Linux page cache (64GB RAM), turning random HDD reads into RAM reads.

**Architecture:**

```
Phase 1: Warm Cache (one-time, ~5-10 min)
  - Run parallel `find` across all hash prefixes
  - Loads directory metadata + small file data into Linux page cache
  - 64GB RAM holds the entire directory structure

Phase 2: Pack (sequential per prefix)
  - blockcompact scans by inode (physical disk order)
  - With warm cache: find=0s, read 10K files=1s, write pack=<1s
  - ~9.4M files per prefix = ~940 packs per prefix
  - 16 prefixes total, 8 per server (split across 2 servers)

Phase 3: Transfer
  - rclone/scp pack files to new servers
  - Single sequential files transfer efficiently
```

**Split across 2 servers** (each has full block data):
- Sharder4 (198.154.93.98): compacts prefixes 0-7
- Sharder5 (209.127.228.230): compacts prefixes 8-f
- Each server processes ~75M blocks
- Both run in parallel on separate HDDs

**Commands:**

```bash
# Step 1: Warm cache (run on each server for its assigned prefixes)
# Sharder4:
for p in 0 1 2 3 4 5 6 7; do
  find /var/0chain/sharder/hdd/docker.local/sharder1/data/blocks/$p \
    -name "*.dat.zlib" -printf "%i\n" > /dev/null &
done

# Sharder5:
for p in 8 9 a b c d e f; do
  find /var/0chain/sharder/hdd/docker.local/sharder1/data/blocks/$p \
    -name "*.dat.zlib" -printf "%i\n" > /dev/null &
done

# Step 2: Compact sequentially (start after cache is warm)
# Sharder4:
nohup bash -c 'for p in 0 1 2 3 4 5 6 7; do
  blockcompact -blocks /var/0chain/sharder/hdd/docker.local/sharder1/data/blocks \
    -prefix $p -n 1000 -batch 10000
done' > /tmp/compact_all.log 2>&1 &

# Sharder5:
nohup bash -c 'for p in 8 9 a b c d e f; do
  blockcompact -blocks /var/0chain/sharder/hdd/docker.local/sharder1/data/blocks \
    -prefix $p -n 1000 -batch 10000
done' > /tmp/compact_all.log 2>&1 &

# Step 3: Transfer packs to new servers
rclone sync oldserver:/path/to/blocks/packs/ newserver:/path/to/blocks/packs/
```

**Timeline**: ~2-3 hours per server with warm cache.

### 2. Current/Future Blocks (Ongoing Production)

For new blocks produced by the live chain. Handled automatically by the sharder's built-in `PackBlockStore` compactor — no manual intervention needed.

**How it works:**

1. Chain produces blocks at ~400ms/round (~2.5 blocks/sec)
2. Sharder writes each block as a loose file (unchanged write path)
3. Background goroutine checks every 10 minutes
4. Queries postgres events_db: `SELECT round, hash FROM blocks WHERE round > $lastPackedRound ORDER BY round LIMIT 10000`
5. When 10,000 new blocks accumulate (~67 min at current chain rate), packs them
6. New pack file written to `basePath/packs/rounds_NNNN_NNNN.pack`
7. In-memory manifest updated — new blocks immediately servable from pack

**Why it's fast**: New blocks are already in the SSD cache (the sharder just wrote them). The compactor reads from cache and writes sequentially to HDD. No random seeks, no cache warming needed.

**Cadence**: One new pack every ~70 minutes. ~20 packs/day. Negligible resource usage.

**Configuration**: Enabled by default when sharder is built with `PackBlockStore`. The `StartCompaction(ctx)` call in `sharder.go` launches the background worker. Set `compactor.db` to the sharder's postgres connection for round-ordered packing.

---

## Performance Data (Measured May 2026)

### Compaction Speed

| Method | Scan | Pack 10K | Rate | Total 150M blocks |
|--------|------|----------|------|--------------------|
| Round-ordered (cold HDD) | instant | 90 min | 2 blk/s | 937 days |
| Inode-sorted (cold HDD) | 1s | 5 min | 33 blk/s | 52 days |
| Inode-sorted (warm cache) | 0s | 1s | 10K blk/s | ~3 hours |

### Transfer Speed

| Method | 10K blocks | Time |
|--------|-----------|------|
| rclone loose files | 10K individual files | hours |
| scp single pack | 1 × 74-91MB file | 45 seconds |

### Key Insight: Why Cache Warming Works

The parallel `find` runs load directory metadata (inode tables, dentry cache) into Linux page cache. With 64GB RAM on these servers, the entire directory structure for ~9.4M files per prefix fits comfortably. When the sequential compactor runs afterward:

- `find -printf '%i %p' | head -10000` reads from RAM → 0 seconds
- `open()` + `read()` for each block file hits page cache → 1 second for 10K files
- Pack write is sequential HDD I/O → full throughput

**Critical**: Do NOT run rclone or other HDD-heavy processes during compaction. They evict the page cache and collapse throughput from 10K blk/s to 2-3 blk/s.

---

## Cleanup Tool: `blockcleaner`

Deletes loose files that have been packed. Processes oldest rounds first.

```bash
blockcleaner -blocks /path/to/blocks -n 50000 [-dry-run]
```

- Reads all pack indices, builds hash set of packed blocks
- Loads packs sorted by round range (oldest first)
- For each packed hash, deletes the corresponding loose file
- `-n` limits deletions per run
- `-dry-run` for preview
- Cleans empty parent directories after deletion

**Run after compaction is complete and packs are verified on all servers.**

---

## File Locations

| File | Purpose |
|------|---------|
| `sharder/blockstore/packfile.go` | Pack file format: read/write, binary search |
| `sharder/blockstore/pack_index.go` | In-memory manifest, candidate lookup |
| `sharder/blockstore/pack_store.go` | `PackBlockStore` implementing `BlockStoreI` |
| `sharder/blockstore/compactor.go` | Background compaction for ongoing production |
| `sharder/blockstore/cmd/blockcompact/` | Standalone tool for historical compaction |
| `sharder/blockstore/cmd/blockcleaner/` | Standalone tool for loose file cleanup |
| `sharder/blockstore/pack_store_test.go` | Unit tests |

---

## Status (May 2026)

- **Sharder4** (198.154.93.98): compacting prefixes 0-7, ~740+ packs done at 10K blk/s
- **Sharder5** (209.127.228.230): compacting prefixes 8-f, cache warming in progress
- Pack store code integrated into sharder (`fs_store.go`, `sharder.go`), Docker build passes
- New blobber servers (sharder1/4 hosts) have 1.2T already synced via rclone
- `blockcompact` and `blockcleaner` binaries cross-compiled for Linux amd64

### Server Mapping

| Old Server | IP | Compacting | New Host | New IP |
|------------|-----|-----------|----------|--------|
| sharder4 | 198.154.93.98 | prefixes 0-7 | blobber1 | 147.124.220.135 |
| sharder5 | 209.127.228.230 | prefixes 8-f | blobber7 | 66.51.159.40 |
| — | — | — | blobber2 (sharder1) | 147.124.220.133 |
