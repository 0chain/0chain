package smartcontractstate

import (
	"bytes"
	"context"

	. "0chain.net/logging"
	"0chain.net/util"
	"github.com/0chain/gorocksdb"
	"go.uber.org/zap"
)

/*PSCDB - a sc db that is persisted */
type PSCDB struct {
	dataDir string
	db      *gorocksdb.DB
	ro      *gorocksdb.ReadOptions
	wo      *gorocksdb.WriteOptions
	to      *gorocksdb.TransactionOptions
	fo      *gorocksdb.FlushOptions
}

/*NewPSCDB - create a new PSCDB */
func NewPSCDB(dataDir string) (*PSCDB, error) {
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
	pnodedb := &PSCDB{db: db}
	pnodedb.dataDir = dataDir
	pnodedb.ro = gorocksdb.NewDefaultReadOptions()
	pnodedb.wo = gorocksdb.NewDefaultWriteOptions()
	pnodedb.wo.SetSync(true)
	pnodedb.to = gorocksdb.NewDefaultTransactionOptions()
	pnodedb.fo = gorocksdb.NewDefaultFlushOptions()
	return pnodedb, nil
}

/*GetNode - implement interface */
func (pndb *PSCDB) GetNode(key Key) (Node, error) {
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
func (pndb *PSCDB) PutNode(key Key, node Node) error {
	data := node
	err := pndb.db.Put(pndb.wo, key, data)
	return err
}

/*DeleteNode - implement interface */
func (pndb *PSCDB) DeleteNode(key Key) error {
	err := pndb.db.Delete(pndb.wo, key)
	return err
}

/*MultiPutNode - implement interface */
func (pndb *PSCDB) MultiPutNode(keys []Key, nodes []Node) error {
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for idx, key := range keys {
		wb.Put(key, nodes[idx])
	}
	return pndb.db.Write(pndb.wo, wb)
}

/*MultiDeleteNode - implement interface */
func (pndb *PSCDB) MultiDeleteNode(keys []Key) error {
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for _, key := range keys {
		wb.Delete(key)
	}
	return pndb.db.Write(pndb.wo, wb)
}

/*Iterate - implement interface */
func (pndb *PSCDB) Iterate(ctx context.Context, handler SCDBIteratorHandler) error {
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
func (pndb *PSCDB) Flush() {
	pndb.db.Flush(pndb.fo)
}

/*Size - count number of keys in the db */
func (pndb *PSCDB) Size(ctx context.Context) int64 {
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
