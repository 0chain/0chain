package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"sync"
	"time"

	"github.com/0chain/gorocksdb"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

/*PNodeDB - a node db that is persisted */
type PNodeDB struct {
	db *gorocksdb.DB

	ro      *gorocksdb.ReadOptions
	wo      *gorocksdb.WriteOptions
	to      *gorocksdb.TransactionOptions
	fo      *gorocksdb.FlushOptions
	mutex   sync.Mutex
	version int64

	deadNodesCFH *gorocksdb.ColumnFamilyHandle
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
func NewPNodeDB(stateDir, logDir string) (*PNodeDB, error) {
	opts := newStateDBOptions(logDir)
	defer opts.Destroy()

	deadNodesOpt := newDeadNodesOptions()
	defer deadNodesOpt.Destroy()

	var (
		cfs     = []string{"default", "dead_nodes"}
		cfsOpts = []*gorocksdb.Options{opts, deadNodesOpt}
		cfh     *gorocksdb.ColumnFamilyHandle
	)

	db, cfhs, err := gorocksdb.OpenDbColumnFamilies(opts, stateDir, cfs, cfsOpts)
	switch err {
	case nil:
		cfh = cfhs[1]
	default:
		if !strings.Contains(err.Error(), "Column family not found") {
			return nil, err
		}

		// open db and create family if not exist
		db, err = gorocksdb.OpenDb(opts, stateDir)
		if err != nil {
			return nil, err
		}

		cfh, err = db.CreateColumnFamily(deadNodesOpt, "dead_nodes")
		if err != nil {
			return nil, err
		}
	}

	wo := gorocksdb.NewDefaultWriteOptions()
	wo.SetSync(false)

	return &PNodeDB{
		db:           db,
		deadNodesCFH: cfh,
		ro:           gorocksdb.NewDefaultReadOptions(),
		wo:           wo,
		to:           gorocksdb.NewDefaultTransactionOptions(),
		fo:           gorocksdb.NewDefaultFlushOptions(),
	}, nil
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

func (pndb *PNodeDB) saveDeadNodes(dn *deadNodes, v int64) error {
	// save back the dead nodes
	d, err := dn.encode(v)
	if err != nil {
		return err
	}

	return pndb.db.PutCF(pndb.wo, pndb.deadNodesCFH, uint64ToBytes(uint64(v)), d)
}

// RecordDeadNodesWithVersion records dead nodes with current finalizing block number
func (pndb *PNodeDB) RecordDeadNodesWithVersion(nodes []Node, v int64) error {
	dn := deadNodes{make(map[string]bool, len(nodes))}
	for _, n := range nodes {
		dn.Nodes[n.GetHash()] = true
	}

	return pndb.saveDeadNodes(&dn, v)
}

func (pndb *PNodeDB) PruneBelowVersionV(ctx context.Context, version Sequence, v int64) error {
	// max prune rounds
	const (
		maxPruneRounds = 500
		maxPruneNodes  = 1000
	)

	type deadNodesRecord struct {
		round     uint64
		nodesKeys []Key
	}

	var (
		ps    = GetPruneStats(ctx)
		count int64

		keys        = make([]Key, 0, maxPruneNodes)
		pruneRounds = make([]uint64, 0, maxPruneRounds)

		deadNodesC = make(chan deadNodesRecord, 1)
	)

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		pndb.iteratorDeadNodes(cctx, func(key, value []byte) bool {
			roundNum := bytesToUint64(key)
			if roundNum >= uint64(version) {
				return false // break iteration
			}

			// decode node keys
			dn := deadNodes{}
			err := dn.decode(value, v)
			if err != nil {
				logging.Logger.Warn("prune state iterator - iterator decode node keys failed",
					zap.Error(err),
					zap.Uint64("round", roundNum))
				return true // continue
			}

			ns := make([]Key, 0, len(dn.Nodes))
			for k := range dn.Nodes {
				kk, err := fromHex(k)
				if err != nil {
					logging.Logger.Warn("prune state - iterator decode key failed",
						zap.Error(err),
						zap.Uint64("round", roundNum))
					return true // continue
				}
				ns = append(ns, kk)
				count++
			}

			deadNodesC <- deadNodesRecord{
				round:     roundNum,
				nodesKeys: ns,
			}
			return true
		})
		close(deadNodesC)
	}()

	for {
		select {
		case dn, ok := <-deadNodesC:
			if !ok {
				// all has been processed
				pndb.Flush()

				if ps != nil {
					ps.Deleted = count
				}

				return nil
			}

			pruneRounds = append(pruneRounds, dn.round)
			keys = append(keys, dn.nodesKeys...)
			if len(pruneRounds) >= maxPruneRounds || len(keys) >= maxPruneNodes {
				// delete nodes
				if err := pndb.MultiDeleteNode(keys); err != nil {
					return err
				}

				if err := pndb.multiDeleteDeadNodes(pruneRounds); err != nil {
					return err
				}

				pruneRounds = pruneRounds[:0]
				keys = keys[:0]
			}
		}
	}
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
		wb.DeleteCF(pndb.deadNodesCFH, uint64ToBytes(r))
	}
	return pndb.db.Write(pndb.wo, wb)
}

func (pndb *PNodeDB) iteratorDeadNodes(ctx context.Context, handler func(key, value []byte) bool) {
	ro := gorocksdb.NewDefaultReadOptions()
	defer ro.Destroy()
	ro.SetFillCache(false)
	it := pndb.db.NewIteratorCF(ro, pndb.deadNodesCFH)
	defer it.Close()
	for it.SeekToFirst(); it.Valid(); it.Next() {
		select {
		case <-ctx.Done():
			return
		default:
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
	pndb.db.Flush(pndb.fo)
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
		logging.Logger.Error("count", zap.Error(err))
		return -1
	}
	return count
}

// Close closes the rocksdb
func (pndb *PNodeDB) Close() {
	pndb.db.Close()
}
