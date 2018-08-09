package util

import (
	"bytes"
	"context"

	. "0chain.net/logging"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
)

/*
type NodeDB interface {
	GetNode(key Key) (Node, error)
	PutNode(key Key, node Node) error
	DeleteNode(key Key) error
}

*/

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
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
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
func (pndb *PNodeDB) GetNode(key Key) (Node, error) {
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
func (pndb *PNodeDB) PutNode(key Key, node Node) error {
	data := node.Encode()
	err := pndb.db.Put(pndb.wo, key, data)
	return err
}

/*DeleteNode - implement interface */
func (pndb *PNodeDB) DeleteNode(key Key) error {
	err := pndb.db.Delete(pndb.wo, key)
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
	ro.SetFillCache(false)
	it := pndb.db.NewIterator(ro)
	defer it.Close()
	for it.SeekToFirst(); it.Valid(); it.Next() {
		key := it.Key()
		value := it.Value()
		node, err := CreateNode(bytes.NewReader(value.Data()))
		if err != nil {
			Logger.Error("iterate - create node", zap.String("key", ToHex(key.Data())), zap.Error(err))
			continue
		}
		err = handler(ctx, key.Data(), node)
		if err != nil {
			Logger.Error("iterate - create node handler error", zap.String("key", ToHex(key.Data())), zap.Error(err))
			break
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
