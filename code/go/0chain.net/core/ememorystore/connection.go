package ememorystore

import (
	"context"
	"fmt"
	"sync"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/linxGnu/grocksdb"
	"go.uber.org/zap"
)

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

type dbpool struct {
	ID     string
	CtxKey common.ContextKey
	Pool   *grocksdb.TransactionDB
}

/*Connection - a struct that manages an underlying connection */
type Connection struct {
	sync.Mutex
	Conn               *grocksdb.Transaction
	ReadOptions        *grocksdb.ReadOptions
	WriteOptions       *grocksdb.WriteOptions
	TransactionOptions *grocksdb.TransactionOptions
	shouldRollback     bool
}

/*Commit - delegates the commit call to underlying connection */
func (c *Connection) Commit() error {
	c.Lock()
	defer c.Unlock()
	err := c.Conn.Commit()
	c.shouldRollback = err != nil
	return err
}

/*CreateDB - create a database */
func CreateDB(dataDir string) (*grocksdb.TransactionDB, error) {
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(grocksdb.NewLRUCache(3 << 30))
	opts := grocksdb.NewDefaultOptions()
	opts.SetKeepLogFileNum(5)
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	tdbopts := grocksdb.NewDefaultTransactionDBOptions()
	return grocksdb.OpenTransactionDb(opts, tdbopts, dataDir)
}

// DefaultPool - default db pool
var DefaultPool *grocksdb.TransactionDB

var pools = make(map[string]*dbpool)

func getConnectionCtxKey(dbid string) common.ContextKey {
	if dbid == "" {
		return CONNECTION
	}
	return common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, dbid))
}

/*AddPool - add a database pool to the repository of db pools */
func AddPool(dbid string, db *grocksdb.TransactionDB) *dbpool {
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
func GetTransaction(db *grocksdb.TransactionDB) *Connection {
	ro := grocksdb.NewDefaultReadOptions()
	wo := grocksdb.NewDefaultWriteOptions()
	to := grocksdb.NewDefaultTransactionOptions()

	t := db.TransactionBegin(wo, to, nil)
	conn := &Connection{Conn: t, ReadOptions: ro, WriteOptions: wo, TransactionOptions: to, shouldRollback: true}
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
		if con.shouldRollback {
			if err := con.Conn.Rollback(); err != nil {
				logging.Logger.Error("rollback failed", zap.Error(err))
			} // commit is expected to be done by the caller of the get connection
		}

		con.Conn.Destroy()
	}
}
