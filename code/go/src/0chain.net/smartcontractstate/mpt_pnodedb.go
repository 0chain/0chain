package smartcontractstate

import (
	"bytes"
	"context"

	. "0chain.net/logging"
	"0chain.net/util"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
)

/*PNodeDB - a node db that is persisted */
type PNodeDB struct {
	dataDir string
	db      *gorocksdb.DB
	ro      *gorocksdb.ReadOptions
	wo      *gorocksdb.WriteOptions
	to      *gorocksdb.TransactionOptions
	fo      *gorocksdb.FlushOptions
}

/*NewPNodeDB - create a new PNodeDB */
func NewPNodeDB(dataDir string) (*PNodeDB, error) {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	/* SetHashSkipListRep + OptimizeForPointLookup seems to be corrupting when pruning
	opts.SetHashSkipListRep(1000000, 4, 4)
	opts.SetAllowConcurrentMemtableWrites(false)
	*/
	opts.OptimizeForPointLookup(64)
	opts.SetCompression(gorocksdb.LZ4Compression)
	db, err := gorocksdb.OpenDb(opts, dataDir)
	if err != nil {
		return nil, err
	}
	pnodedb := &PNodeDB{db: db}
	pnodedb.dataDir = dataDir
	pnodedb.ro = gorocksdb.NewDefaultReadOptions()
	pnodedb.wo = gorocksdb.NewDefaultWriteOptions()
	pnodedb.wo.SetSync(true)
	pnodedb.to = gorocksdb.NewDefaultTransactionOptions()
	pnodedb.fo = gorocksdb.NewDefaultFlushOptions()
	return pnodedb, nil
}

/*GetNode - implement interface */
func (pndb *PNodeDB) GetNode(key util.Key) (Node, error) {
	data, err := pndb.db.Get(pndb.ro, key)
	if err != nil {
		return nil, err
	}
	defer data.Free()
	buf := data.Data()
	if buf == nil || len(buf) == 0 {
		return nil, ErrNodeNotFound
	}
	return CreateNode(bytes.NewReader(buf))
}

/*PutNode - implement interface */
func (pndb *PNodeDB) PutNode(key util.Key, node Node) error {
	data := node.Encode()
	err := pndb.db.Put(pndb.wo, key, data)
	return err
}

/*DeleteNode - implement interface */
func (pndb *PNodeDB) DeleteNode(key util.Key) error {
	err := pndb.db.Delete(pndb.wo, key)
	return err
}

/*MultiPutNode - implement interface */
func (pndb *PNodeDB) MultiPutNode(keys []util.Key, nodes []Node) error {
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for idx, key := range keys {
		wb.Put(key, nodes[idx].Encode())
	}
	return pndb.db.Write(pndb.wo, wb)
}

/*MultiDeleteNode - implement interface */
func (pndb *PNodeDB) MultiDeleteNode(keys []util.Key) error {
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
			Logger.Error("iterate - create node", zap.String("key", util.ToHex(kdata)), zap.Error(err))
			continue
		}
		err = handler(ctx, kdata, node)
		if err != nil {
			key.Free()
			value.Free()
			Logger.Error("iterate - create node handler error", zap.String("key", util.ToHex(kdata)), zap.Any("data", vdata), zap.Error(err))
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
	handler := func(ctx context.Context, key util.Key, node Node) error {
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
