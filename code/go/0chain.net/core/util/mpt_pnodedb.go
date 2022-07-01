package util

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/0chain/gorocksdb"

	"0chain.net/core/logging"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*PNodeDB - a node db that is persisted */
type PNodeDB struct {
	dataDir  string
	db       *gorocksdb.DB
	ro       *gorocksdb.ReadOptions
	wo       *gorocksdb.WriteOptions
	to       *gorocksdb.TransactionOptions
	fo       *gorocksdb.FlushOptions
	mutex    sync.Mutex
	version  int64
	versions []int64
}

const (
	SSTTypeBlockBasedTable = 0
	SSTTypePlainTable      = 1
)

var (
	PNodeDBCompression = gorocksdb.LZ4Compression
)

var sstType = SSTTypeBlockBasedTable

/*NewPNodeDB - create a new PNodeDB */
func NewPNodeDB(dataDir, logDir string) (*PNodeDB, error) {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCompression(PNodeDBCompression)
	if sstType == SSTTypePlainTable {
		opts.SetAllowMmapReads(true)
		opts.SetPrefixExtractor(gorocksdb.NewFixedPrefixTransform(6))
		opts.SetPlainTableFactory(32, 10, 0.75, 16)
	} else {
		opts.OptimizeForPointLookup(64)
		opts.SetAllowMmapReads(true)
		opts.SetPrefixExtractor(gorocksdb.NewFixedPrefixTransform(6))
	}
	opts.IncreaseParallelism(2)          // pruning and saving happen in parallel
	opts.SetSkipLogErrorOnRecovery(true) // do sync if necessary
	opts.SetDbLogDir(logDir)
	opts.EnableStatistics()
	opts.OptimizeUniversalStyleCompaction(64 * 1024 * 1024)
	db, err := gorocksdb.OpenDb(opts, dataDir)
	if err != nil {
		return nil, err
	}
	pnodedb := &PNodeDB{db: db}
	pnodedb.dataDir = dataDir
	pnodedb.ro = gorocksdb.NewDefaultReadOptions()
	pnodedb.wo = gorocksdb.NewDefaultWriteOptions()
	pnodedb.wo.SetSync(false)
	pnodedb.to = gorocksdb.NewDefaultTransactionOptions()
	pnodedb.fo = gorocksdb.NewDefaultFlushOptions()
	return pnodedb, nil
}

/*GetNode - implement interface */
func (pndb *PNodeDB) GetNode(key Key) (Node, error) {
	data, err := pndb.db.Get(pndb.ro, key)
	if err != nil {
		return nil, err
	}
	defer data.Free()
	buf := data.Data()
	if len(buf) == 0 {
		return nil, ErrNodeNotFound
	}
	return CreateNode(bytes.NewReader(buf))
}

/*PutNode - implement interface */
func (pndb *PNodeDB) PutNode(key Key, node Node) error {
	data := node.Encode()
	err := pndb.db.Put(pndb.wo, key, data)
	if DebugMPTNode {
		logging.Logger.Debug("node put to PersistDB",
			zap.String("key", ToHex(key)), zap.Error(err),
			zap.Int64("Origin", int64(node.GetOrigin())),
			zap.Int64("Version", int64(node.GetVersion())))
	}
	return err
}

/*DeleteNode - implement interface */
func (pndb *PNodeDB) DeleteNode(key Key) error {
	err := pndb.db.Delete(pndb.wo, key)
	return err
}

/*MultiGetNode - get multiple nodes */
func (pndb *PNodeDB) MultiGetNode(keys []Key) ([]Node, error) {
	var nodes []Node
	var err error
	for _, key := range keys {
		node, nerr := pndb.GetNode(key)
		if nerr != nil {
			err = nerr
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, err
}

/*MultiPutNode - implement interface */
func (pndb *PNodeDB) MultiPutNode(keys []Key, nodes []Node) error {
	ts := time.Now()
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for idx, key := range keys {
		wb.Put(key, nodes[idx].Encode())
		if DebugMPTNode {
			logging.Logger.Debug("multi node put to PersistDB",
				zap.String("key", ToHex(key)),
				zap.Int64("Origin", int64(nodes[idx].GetOrigin())),
				zap.Int64("Version", int64(nodes[idx].GetVersion())))
		}
	}
	err := pndb.db.Write(pndb.wo, wb)
	if err != nil {
		logging.Logger.Debug("pnode save nodes failed",
			zap.Int64("round", pndb.version),
			zap.Any("duration", ts),
			zap.Error(err))
	}
	return err
}

/*MultiDeleteNode - implement interface */
func (pndb *PNodeDB) MultiDeleteNode(keys []Key) error {
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for _, key := range keys {
		wb.Delete(key)
	}
	return pndb.db.Write(pndb.wo, wb)
}

/*Iterate - implement interface */
func (pndb *PNodeDB) Iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	ro := gorocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	ro.SetFillCache(false)
	it := pndb.db.NewIterator(ro)
	defer it.Close()
	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()
		kdata := key.Data()
		vdata := value.Data()
		node, err := CreateNode(bytes.NewReader(vdata))
		if err != nil {
			key.Free()
			value.Free()
			Logger.Error("iterate - create node", zap.String("key", ToHex(kdata)), zap.Error(err))
			continue
		}
		err = handler(ctx, kdata, node)
		if err != nil {
			key.Free()
			value.Free()
			Logger.Error("iterate - create node handler error", zap.String("key", ToHex(kdata)), zap.Any("data", vdata), zap.Error(err))
			return err
		}
		key.Free()
		value.Free()
	}
	return nil
}

/*Flush - flush the db */
func (pndb *PNodeDB) Flush() {
	pndb.db.Flush(pndb.fo)
}

/*PruneBelowVersion - prune the state below the given origin */
func (pndb *PNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
	ps := GetPruneStats(ctx)
	var total int64
	var count int64
	var leaves int64
	batch := make([]Key, 0, BatchSize)
	handler := func(ctx context.Context, key Key, node Node) error {
		total++
		if node.GetVersion() >= version {
			if _, ok := node.(*LeafNode); ok {
				leaves++
			}
			return nil
		}
		count++
		tkey := make([]byte, len(key))
		copy(tkey, key)
		batch = append(batch, tkey)
		if len(batch) == BatchSize {
			err := pndb.MultiDeleteNode(batch)
			batch = batch[:0]
			if err != nil {
				Logger.Error("prune below origin - error deleting node",
					zap.String("key", ToHex(key)),
					zap.Any("old_version", node.GetVersion()),
					zap.Any("new_version", version),
					zap.Error(err))
				return err
			}
		}
		return nil
	}
	err := pndb.Iterate(ctx, handler)
	if err != nil {
		return err
	}
	if len(batch) > 0 {
		err := pndb.MultiDeleteNode(batch)
		if err != nil {
			Logger.Error("prune below origin - error deleting node", zap.Any("new_version", version), zap.Error(err))
			return err
		}
	}
	pndb.Flush()
	if ps != nil {
		ps.Total = total
		ps.Leaves = leaves
		ps.Deleted = count
	}
	return err
}

/*Size - count number of keys in the db */
func (pndb *PNodeDB) Size(ctx context.Context) int64 {
	var count int64
	handler := func(ctx context.Context, key Key, node Node) error {
		count++
		return nil
	}
	err := pndb.Iterate(ctx, handler)
	if err != nil {
		Logger.Error("count", zap.Error(err))
		return -1
	}
	return count
}

// Close close the rocksdb
func (pndb *PNodeDB) Close() {
	pndb.db.Close()
}

// GetDBVersions retusn all tracked db versions
func (pndb *PNodeDB) GetDBVersions() []int64 {
	pndb.mutex.Lock()
	defer pndb.mutex.Unlock()
	vs := make([]int64, len(pndb.versions))
	copy(vs, pndb.versions)

	return vs
}

// TrackDBVersion appends the db version to tracked records
func (pndb *PNodeDB) TrackDBVersion(v int64) {
	pndb.mutex.Lock()
	defer pndb.mutex.Unlock()
	pndb.versions = append(pndb.versions, v)
}
