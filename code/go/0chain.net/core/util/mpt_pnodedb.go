package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"
	"time"

	"github.com/0chain/gorocksdb"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

/*PNodeDB - a node db that is persisted */
type PNodeDB struct {
	// state db
	stateDB *gorocksdb.DB
	// dead nodes db
	deadNodesDB *gorocksdb.DB

	ro      *gorocksdb.ReadOptions
	wo      *gorocksdb.WriteOptions
	to      *gorocksdb.TransactionOptions
	fo      *gorocksdb.FlushOptions
	mutex   sync.Mutex
	version int64
}

const (
	SSTTypeBlockBasedTable = 0
	SSTTypePlainTable      = 1
)

var (
	PNodeDBCompression = gorocksdb.LZ4Compression
	deadNodesKey       = []byte("dead_nodes")
)

var sstType = SSTTypeBlockBasedTable

func newStateDBOptions(logDir string) *gorocksdb.Options {
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

	return opts
}

func newDeadNodesOptions() *gorocksdb.Options {
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetKeepLogFileNum(5)
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	return opts
}

// NewPNodeDB - create a new PNodeDB
func NewPNodeDB(stateDir, deadNodesDir, logDir string) (*PNodeDB, error) {
	opts := newStateDBOptions(logDir)
	db, err := gorocksdb.OpenDb(opts, stateDir)
	if err != nil {
		return nil, err
	}

	dnOpts := newDeadNodesOptions()
	dnDB, err := gorocksdb.OpenDb(dnOpts, deadNodesDir)
	if err != nil {
		return nil, err
	}

	wo := gorocksdb.NewDefaultWriteOptions()
	wo.SetSync(false)

	return &PNodeDB{
		stateDB:     db,
		deadNodesDB: dnDB,
		ro:          gorocksdb.NewDefaultReadOptions(),
		wo:          wo,
		to:          gorocksdb.NewDefaultTransactionOptions(),
		fo:          gorocksdb.NewDefaultFlushOptions(),
	}, nil
}

/*GetNode - implement interface */
func (pndb *PNodeDB) GetNode(key Key) (Node, error) {
	data, err := pndb.stateDB.Get(pndb.ro, key)
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
	err := pndb.stateDB.Put(pndb.wo, key, data)
	if DebugMPTNode {
		logging.Logger.Debug("node put to PersistDB",
			zap.String("key", ToHex(key)), zap.Error(err),
			zap.Int64("Origin", int64(node.GetOrigin())),
			zap.Int64("Version", int64(node.GetVersion())))
	}
	return err
}

//func (pndb *PNodeDB) getDeadNodes(v int64) (*deadNodes, error) {
//	data, err := pndb.stateDB.Get(pndb.ro, deadNodesKey)
//	if err != nil {
//		return nil, err
//	}
//
//	defer data.Free()
//	buf := data.Data()
//
//	dn := deadNodes{Nodes: make(map[string]int64)}
//	if len(buf) > 0 {
//		if err := dn.decode(buf, v); err != nil {
//			return nil, err
//		}
//	}
//	return &dn, nil
//}

func (pndb *PNodeDB) saveDeadNodes(dn *deadNodes, v int64) error {
	// save back the dead nodes
	d, err := dn.encode(v)
	if err != nil {
		return err
	}

	return pndb.deadNodesDB.Put(pndb.wo, uint64ToBytes(uint64(v)), d)
}

func (pndb *PNodeDB) RecordDeadNodesWithVersion(nodes []Node, v int64) error {
	dn := deadNodes{make(map[string]bool, len(nodes))}
	for _, n := range nodes {
		dn.Nodes[n.GetHash()] = true
	}

	return pndb.saveDeadNodes(&dn, v)
}

func (pndb *PNodeDB) PruneBelowVersionV(ctx context.Context, version Sequence, v int64) error {
	var (
		ps    = GetPruneStats(ctx)
		count int64

		keys        []Key
		pruneRounds []uint64
	)

	pndb.iteratorDeadNodes(func(key, value []byte) bool {
		roundNum := bytesToUint64(key)
		if roundNum >= uint64(version) {
			return false // break iteration
		}

		pruneRounds = append(pruneRounds, roundNum)

		// decode node keys
		dn := deadNodes{}
		err := dn.decode(value, v)
		if err != nil {
			logging.Logger.Warn("prune state iterator - iterator decode node keys failed",
				zap.Error(err),
				zap.Uint64("round", roundNum))
			return true // continue
		}

		for k := range dn.Nodes {
			kk, err := fromHex(k)
			if err != nil {
				logging.Logger.Warn("prune state - iterator decode key failed",
					zap.Error(err),
					zap.Uint64("round", roundNum))
				return true // continue
			}
			keys = append(keys, kk)
			count++
		}

		return true
	})

	// delete nodes
	if err := pndb.MultiDeleteNode(keys); err != nil {
		return err
	}

	if err := pndb.multiDeleteDeadNodes(pruneRounds); err != nil {
		return err
	}

	pndb.Flush()

	if ps != nil {
		ps.Deleted = count
	}

	return nil
}

func uint64ToBytes(r uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, r)
	return b
}

func bytesToUint64(data []byte) uint64 {
	return binary.BigEndian.Uint64(data)
}

/*MultiDeleteNode - implement interface */
func (pndb *PNodeDB) multiDeleteDeadNodes(rounds []uint64) error {
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for _, r := range rounds {
		wb.Delete(uint64ToBytes(r))
	}
	return pndb.deadNodesDB.Write(pndb.wo, wb)
}

func (pndb *PNodeDB) iteratorDeadNodes(handler func(key, value []byte) bool) {
	ro := gorocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	ro.SetFillCache(false)
	it := pndb.deadNodesDB.NewIterator(ro)
	defer it.Close()
	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()

		keyData := key.Data()
		valueData := value.Data()
		if !handler(keyData, valueData) {
			key.Free()
			value.Free()
			return
		}

		key.Free()
		value.Free()
	}
}

/*DeleteNode - implement interface */
func (pndb *PNodeDB) DeleteNode(key Key) error {
	err := pndb.stateDB.Delete(pndb.wo, key)
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
	err := pndb.stateDB.Write(pndb.wo, wb)
	if err != nil {
		logging.Logger.Error("pnode save nodes failed",
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
	return pndb.stateDB.Write(pndb.wo, wb)
}

/*Iterate - implement interface */
func (pndb *PNodeDB) Iterate(ctx context.Context, handler NodeDBIteratorHandler) error {
	ro := gorocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	ro.SetFillCache(false)
	it := pndb.stateDB.NewIterator(ro)
	defer it.Close()
	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()
		kdata := key.Data()
		if bytes.Equal(kdata, deadNodesKey) {
			continue
		}
		vdata := value.Data()
		node, err := CreateNode(bytes.NewReader(vdata))
		if err != nil {
			key.Free()
			value.Free()
			logging.Logger.Error("iterate - create node", zap.String("key", ToHex(kdata)), zap.Error(err))
			continue
		}
		err = handler(ctx, kdata, node)
		if err != nil {
			key.Free()
			value.Free()
			logging.Logger.Error("iterate - create node handler error", zap.String("key", ToHex(kdata)), zap.Any("data", vdata), zap.Error(err))
			return err
		}
		key.Free()
		value.Free()
	}
	return nil
}

/*Flush - flush the db */
func (pndb *PNodeDB) Flush() {
	pndb.stateDB.Flush(pndb.fo)
	pndb.deadNodesDB.Flush(pndb.fo)
}

/*PruneBelowVersion - prune the state below the given origin */
//func (pndb *PNodeDB) PruneBelowVersion(ctx context.Context, version Sequence) error {
//	ps := GetPruneStats(ctx)
//	var total int64
//	var count int64
//	var leaves int64
//	batch := make([]Key, 0, BatchSize)
//	keys := make([]string, 0, BatchSize)
//	handler := func(ctx context.Context, key Key, node Node) error {
//		total++
//		if node.GetVersion() >= version {
//			if _, ok := node.(*LeafNode); ok {
//				leaves++
//			}
//			return nil
//		}
//		count++
//		tkey := make([]byte, len(key))
//		copy(tkey, key)
//		batch = append(batch, tkey)
//		keys = append(keys, ToHex(tkey))
//		if len(batch) == BatchSize {
//			logging.Logger.Debug("prune batch keys", zap.Strings("keys", keys))
//			err := pndb.MultiDeleteNode(batch)
//			batch = batch[:0]
//			keys = keys[:0]
//			if err != nil {
//				Logger.Error("prune below origin - error deleting node",
//					zap.String("key", ToHex(key)),
//					zap.Any("old_version", node.GetVersion()),
//					zap.Any("new_version", version),
//					zap.Error(err))
//				return err
//			}
//		}
//		return nil
//	}
//	err := pndb.Iterate(ctx, handler)
//	if err != nil {
//		return err
//	}
//	if len(batch) > 0 {
//		logging.Logger.Debug("prune batch keys", zap.Strings("keys", keys))
//		err := pndb.MultiDeleteNode(batch)
//		if err != nil {
//			Logger.Error("prune below origin - error deleting node", zap.Any("new_version", version), zap.Error(err))
//			return err
//		}
//	}
//	pndb.Flush()
//	if ps != nil {
//		ps.Total = total
//		ps.Leaves = leaves
//		ps.Deleted = count
//	}
//	return err
//}

/*Size - count number of keys in the db */
func (pndb *PNodeDB) Size(ctx context.Context) int64 {
	var count int64
	handler := func(ctx context.Context, key Key, node Node) error {
		count++
		return nil
	}
	err := pndb.Iterate(ctx, handler)
	if err != nil {
		logging.Logger.Error("count", zap.Error(err))
		return -1
	}
	return count
}

// Close close the rocksdb
func (pndb *PNodeDB) Close() {
	pndb.stateDB.Close()
	pndb.deadNodesDB.Close()
}
