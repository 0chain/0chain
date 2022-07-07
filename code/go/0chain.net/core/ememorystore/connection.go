package ememorystore

import (
	"context"
	"fmt"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"github.com/0chain/gorocksdb"
	"go.uber.org/zap"
)

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

type dbpool struct {
	ID     string
	CtxKey common.ContextKey
	Pool   *gorocksdb.TransactionDB
}

/*Connection - a struct that manages an underlying connection */
type Connection struct {
	Conn               *gorocksdb.Transaction
	ReadOptions        *gorocksdb.ReadOptions
	WriteOptions       *gorocksdb.WriteOptions
	TransactionOptions *gorocksdb.TransactionOptions
}

/*Commit - delegates the commit call to underlying connection */
func (c *Connection) Commit() error {
	return c.Conn.Commit()
}

/*CreateDB - create a database */
func CreateDB(dataDir string) (*gorocksdb.TransactionDB, error) {
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetKeepLogFileNum(5)
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	tdbopts := gorocksdb.NewDefaultTransactionDBOptions()
	return gorocksdb.OpenTransactionDb(opts, tdbopts, dataDir)
}

func OpenDBWithColumnFamilies(
	dir string,
	cfs []string, // columnFamilies
	cfsOpts []*gorocksdb.Options, // columnFamilies options
	cacheSize uint64,
	isCreate bool,
) (*gorocksdb.DB, gorocksdb.ColumnFamilyHandles, error) {

	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(cacheSize))

	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	if isCreate {
		opts.SetCreateIfMissing(true)
		db, err := gorocksdb.OpenDb(opts, dir)
		if err != nil {
			return nil, nil, err
		}
		return db, nil, nil
	}

	db, cfHs, err := gorocksdb.OpenDbColumnFamilies(opts, dir, cfs, cfsOpts)
	if err != nil {
		return nil, nil, err
	}
	return db, cfHs, nil
}

//DefaultPool - default db pool
// FIXME This DefaultPool is always nil.
var DefaultPool *gorocksdb.TransactionDB

var pools = make(map[string]*dbpool)

func getConnectionCtxKey(dbid string) common.ContextKey {
	if dbid == "" {
		return CONNECTION
	}
	return common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, dbid))
}

/*AddPool - add a database pool to the repository of db pools */
func AddPool(dbid string, db *gorocksdb.TransactionDB) *dbpool {
	dbpool := &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: db}
	pools[dbid] = dbpool
	return dbpool
}

func getdbpool(entityMetadata datastore.EntityMetadata) *dbpool {
	dbid := entityMetadata.GetDB()
	dbpool, ok := pools[dbid]
	if !ok {
		panicf("Invalid entity metadata setup, unknown dbpool %v\n", dbid)
	}
	return dbpool
}

/*GetConnection - returns a connection from the Pool
* Should always use right after getting the connection to avoid leaks
* defer c.Close()
 */
func GetConnection() *Connection {
	return GetTransaction(DefaultPool)
}

/*GetTransaction - get the transaction object associated with this db */
func GetTransaction(db *gorocksdb.TransactionDB) *Connection {
	ro := gorocksdb.NewDefaultReadOptions()
	wo := gorocksdb.NewDefaultWriteOptions()
	to := gorocksdb.NewDefaultTransactionOptions()

	t := db.TransactionBegin(wo, to, nil)
	conn := &Connection{Conn: t, ReadOptions: ro, WriteOptions: wo, TransactionOptions: to}
	return conn
}

/*GetEntityConnection - returns a connection from the pool configured for the entity */
func GetEntityConnection(entityMetadata datastore.EntityMetadata) *Connection {
	dbid := entityMetadata.GetDB()
	if dbid == "" {
		return GetConnection()
	}
	dbpool := getdbpool(entityMetadata)
	return GetTransaction(dbpool.Pool)
}

/*CONNECTION - key used to get the connection object from the context */
const CONNECTION common.ContextKey = "econnection."

type connections map[common.ContextKey]*Connection

/*WithConnection takes a context and adds a connection value to it */
func WithConnection(ctx context.Context) context.Context {
	c := ctx.Value(CONNECTION)
	if c == nil {
		cMap := make(connections)
		cMap[CONNECTION] = GetConnection()
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := c.(connections)
	if !ok {
		panicf("invalid setup, type of connection is %T", c)
	}
	_, ok = cMap[CONNECTION]
	if !ok {
		cMap[CONNECTION] = GetConnection()
	}
	return ctx

}

/*GetCon returns a connection stored in the context which got created via WithConnection */
func GetCon(ctx context.Context) *Connection {
	if ctx == nil {
		return GetConnection()
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		con := GetConnection()
		cMap := make(connections)
		cMap[CONNECTION] = con
		return con
	}
	cMap, ok := c.(connections)
	if !ok {
		panicf("invalid setup, type of connection is %T", c)
	}
	con, ok := cMap[CONNECTION]
	if !ok {
		con = GetConnection()
		cMap[CONNECTION] = con
	}
	return con
}

/*WithEntityConnection - returns a connection as per the configuration of the entity */
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	dbpool := getdbpool(entityMetadata)
	if dbpool.Pool == DefaultPool {
		return WithConnection(ctx)
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		cMap := make(connections)
		cMap[dbpool.CtxKey] = GetTransaction(dbpool.Pool)
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := c.(connections)
	if !ok {
		panicf("invalid setup, type of connection is %T", c)
	}
	_, ok = cMap[dbpool.CtxKey]
	if !ok {
		cMap[dbpool.CtxKey] = GetTransaction(dbpool.Pool)
	}
	return ctx

}

/*GetEntityCon returns a connection stored in the context which got created via WithEntityConnection */
func GetEntityCon(ctx context.Context, entityMetadata datastore.EntityMetadata) *Connection {
	if ctx == nil {
		return GetEntityConnection(entityMetadata)
	}
	dbpool := getdbpool(entityMetadata)
	if dbpool.Pool == DefaultPool {
		return GetCon(ctx)
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		return nil
	}
	cMap, ok := c.(connections)
	if !ok {
		panicf("invalid setup, type of connection is %T", c)
	}
	con, ok := cMap[dbpool.CtxKey]
	if !ok {
		con = GetEntityConnection(entityMetadata)
		cMap[dbpool.CtxKey] = con
	}
	return con
}

/*Close - Close takes care of maintaining the closing of connection(s) stored in the context */
func Close(ctx context.Context) {
	c := ctx.Value(CONNECTION)
	if c == nil {
		return
	}
	cMap, ok := c.(connections)
	if !ok {
		panicf("invalid setup, type of connection is %T", c)
	}
	for _, con := range cMap {
		con.ReadOptions.Destroy()
		con.WriteOptions.Destroy()
		con.TransactionOptions.Destroy()
		if err := con.Conn.Rollback(); err != nil {
			logging.Logger.Warn("rollback failed", zap.Error(err))
		} // commit is expected to be done by the caller of the get connection
		con.Conn.Destroy()
	}
}
