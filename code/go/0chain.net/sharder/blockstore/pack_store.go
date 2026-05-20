package blockstore

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// PackBlockStore implements BlockStoreI with pack file support.
// Read path: SSD cache → pack files → loose files.
// Write path: loose files (unchanged), background compaction packs them.
type PackBlockStore struct {
	basePath              string
	packsPath             string
	blockMetadataProvider datastore.EntityMetadata
	cache                 cacher
	manifest              *packManifest
	compactor             *compactor
}

// NewPackBlockStore creates a pack-aware block store.
// Pass db=nil to disable background compaction (use standalone compactor instead).
func NewPackBlockStore(basePath string, cache cacher, metaProvider datastore.EntityMetadata, db *sql.DB) *PackBlockStore {
	packsPath := filepath.Join(basePath, "packs")
	os.MkdirAll(packsPath, 0700)

	manifest := newPackManifest()
	if err := manifest.load(packsPath); err != nil {
		logging.Logger.Error("failed to load pack manifest", zap.Error(err))
	}

	comp := newCompactor(basePath, packsPath, manifest, db)

	return &PackBlockStore{
		basePath:              basePath,
		packsPath:             packsPath,
		blockMetadataProvider: metaProvider,
		cache:                 cache,
		manifest:              manifest,
		compactor:             comp,
	}
}

// StartCompaction begins background compaction. Call after Init.
func (ps *PackBlockStore) StartCompaction(ctx context.Context) {
	ps.compactor.start(ctx)
}

// Write stores a block as a loose file (unchanged from current behavior).
func (ps *PackBlockStore) Write(b *block.Block) error {
	err := ps.writeLoose(b.Hash, b)
	if err != nil {
		return err
	}

	if b.MagicBlock != nil && b.Round == b.MagicBlock.StartingRound {
		logging.Logger.Debug("save magic block",
			zap.Int64("round", b.Round),
			zap.String("mb hash", b.MagicBlock.Hash),
		)
		return ps.writeLoose(b.MagicBlock.Hash, b)
	}
	return nil
}

// Read looks up a block by hash. Search order: cache → packs → loose file.
func (ps *PackBlockStore) Read(hash string) (*block.Block, error) {
	b := ps.blockMetadataProvider.Instance().(*block.Block)

	// 1. Check SSD cache
	data, err := ps.cache.Read(hash)
	if data != nil && err == nil {
		r := bytes.NewReader(data)
		if err := datastore.ReadMsgpack(r, b); err == nil {
			return b, nil
		}
	}

	// 2. Check pack files
	rawData, err := ps.manifest.lookup(hash)
	if err != nil {
		logging.Logger.Warn("pack lookup error", zap.String("hash", hash), zap.Error(err))
	}
	if rawData != nil {
		b, err = ps.decodeBlock(rawData)
		if err == nil {
			ps.cacheBlock(hash, b)
			return b, nil
		}
		logging.Logger.Warn("pack decode error", zap.String("hash", hash), zap.Error(err))
	}

	// 3. Fall back to loose file
	b, err = ps.readLoose(hash)
	if err != nil {
		return nil, err
	}

	ps.cacheBlock(hash, b)
	return b, nil
}

// ReadWithBlockSummary reads a block given its summary.
func (ps *PackBlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	return ps.Read(bs.Hash)
}

// writeLoose writes a block as an individual file (current format).
func (ps *PackBlockStore) writeLoose(hash string, b *block.Block) error {
	bp, err := getBlockFilePath(hash)
	if err != nil {
		return err
	}
	bPath := filepath.Join(ps.basePath, bp)
	if err := os.MkdirAll(filepath.Dir(bPath), 0700); err != nil {
		return err
	}

	f, err := os.Create(bPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := zlib.NewWriterLevel(f, zlib.BestCompression)
	if err != nil {
		return err
	}
	if err := datastore.WriteMsgpack(w, b); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// Also write to cache
	go func() {
		ctx, cancel := context.WithTimeout(context.TODO(), CacheWriteTimeOut)
		defer cancel()
		if err := ps.cache.Write(ctx, hash, b); err != nil {
			logging.Logger.Error(err.Error())
		}
	}()

	return nil
}

// readLoose reads a block from a loose file (current format).
func (ps *PackBlockStore) readLoose(hash string) (*block.Block, error) {
	bp, err := getBlockFilePath(hash)
	if err != nil {
		return nil, err
	}
	bPath := filepath.Join(ps.basePath, bp)
	f, err := os.Open(bPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	b := ps.blockMetadataProvider.Instance().(*block.Block)
	if err := datastore.ReadMsgpack(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

// decodeBlock decodes a raw zlib-compressed msgpack block.
func (ps *PackBlockStore) decodeBlock(rawData []byte) (*block.Block, error) {
	r, err := zlib.NewReader(bytes.NewReader(rawData))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	b := ps.blockMetadataProvider.Instance().(*block.Block)
	if err := datastore.ReadMsgpack(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

// cacheBlock writes a block to the SSD cache in the background.
func (ps *PackBlockStore) cacheBlock(hash string, b *block.Block) {
	go func() {
		ctx, cancel := context.WithTimeout(context.TODO(), CacheWriteTimeOut)
		defer cancel()
		if err := ps.cache.Write(ctx, hash, b); err != nil {
			logging.Logger.Error(err.Error())
		}
	}()
}

// Compact runs a manual compaction pass. Returns number of blocks packed.
func (ps *PackBlockStore) Compact() (int, error) {
	return ps.compactor.compact()
}

// SetCompactorDB sets the database connection for round-ordered compaction.
// Call this after the sharder's postgres is initialized.
func (ps *PackBlockStore) SetCompactorDB(db *sql.DB) {
	ps.compactor.db = db
}

// SetCompactorStartRound sets the round from which the background compactor
// begins packing. Use this on first deploy after historical migration to avoid
// re-packing blocks that already exist in inode-sorted packs.
func (ps *PackBlockStore) SetCompactorStartRound(round int64) {
	ps.compactor.lastPackedRd = round
}

// PackCount returns the number of pack files.
func (ps *PackBlockStore) PackCount() int {
	return ps.manifest.packCount()
}
