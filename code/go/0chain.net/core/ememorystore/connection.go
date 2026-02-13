package ememorystore

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// CONNECTION_MAP is the key used to store connection map in context
type connMapKey struct{}

var CONNECTION_MAP = connMapKey{}

// ConnectionRegistry manages all active connections
type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string]*Connection
}

var registry = &ConnectionRegistry{
	connections: make(map[string]*Connection),
}

/*Connection - a struct that manages an underlying connection */
type Connection struct {
	ID                 string
	Conn               *grocksdb.Transaction
	Pool               *grocksdb.TransactionDB
	ReadOptions        *grocksdb.ReadOptions
	WriteOptions       *grocksdb.WriteOptions
	TransactionOptions *grocksdb.TransactionOptions
	CreatedAt          time.Time
	shouldRollback     bool
}

/*Commit - delegates the commit call to underlying connection */
func (c *Connection) Commit() error {
	err := c.Conn.Commit()
	c.shouldRollback = err != nil
	return err
}

func defaultDBOptions() *grocksdb.Options {
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(grocksdb.NewLRUCache(3 << 30))
	opts := grocksdb.NewDefaultOptions()
	opts.SetKeepLogFileNum(5)
	opts.SetMaxLogFileSize(100 * 1024 * 1024) // rotate info log at 100 MB
	opts.SetMaxTotalWalSize(32 * 1024 * 1024) // 32 MB logical WAL cap
	opts.SetWriteBufferSize(4 * 1024 * 1024)  // 4 MB — reduces WAL pre-allocation from 64 MB to ~4 MB per file
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	return opts
}

/*CreateDB - create a database */
func CreateDB(dataDir string) (*grocksdb.TransactionDB, error) {
	opts := defaultDBOptions()
	tdbopts := grocksdb.NewDefaultTransactionDBOptions()
	return grocksdb.OpenTransactionDb(opts, tdbopts, dataDir)
}

/* CreateDBWithMergeOperator - create a database with merge operator */
func CreateDBWithMergeOperator(dataDir string, mergeOperator grocksdb.MergeOperator) (*grocksdb.TransactionDB, error) {
	opts := defaultDBOptions()
	opts.SetMergeOperator(mergeOperator)
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

// Generate a unique connection ID using entity metadata and timestamp
func generateConnectionID(entityMetadata datastore.EntityMetadata) string {
	return fmt.Sprintf("%s-%d", entityMetadata.GetName(), time.Now().UnixNano())
}

// WithEntityConnection creates or retrieves a connection for the given entity metadata
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	dbpool := getdbpool(entityMetadata)
	if dbpool.Pool == DefaultPool {
		return WithConnection(ctx)
	}

	// Generate a unique connection ID
	connID := generateConnectionID(entityMetadata)

	// Create the transaction
	conn := GetTransaction(dbpool.Pool)
	conn.ID = connID

	// Register the connection
	registry.mu.Lock()
	registry.connections[connID] = conn
	registry.mu.Unlock()

	// Get or create the connection map from context
	var connMap map[string]string
	mapValue := ctx.Value(CONNECTION_MAP)
	if mapValue == nil {
		connMap = make(map[string]string)
	} else {
		connMap = mapValue.(map[string]string)
	}

	// Store the entity -> connection ID mapping
	connMap[entityMetadata.GetName()] = connID

	return context.WithValue(ctx, CONNECTION_MAP, connMap)
}

// GetEntityCon retrieves the connection for the given entity metadata
func GetEntityCon(ctx context.Context, entityMetadata datastore.EntityMetadata) *Connection {
	// Get the connection map
	mapValue := ctx.Value(CONNECTION_MAP)
	if mapValue == nil {
		// Use default if not found
		return GetEntityConnection(entityMetadata)
	}

	connMap := mapValue.(map[string]string)
	connID, exists := connMap[entityMetadata.GetName()]
	if !exists {
		// Use default if not mapped
		return GetEntityConnection(entityMetadata)
	}

	// Find the connection in registry
	registry.mu.RLock()
	conn, exists := registry.connections[connID]
	registry.mu.RUnlock()

	if !exists {
		return GetEntityConnection(entityMetadata)
	}

	return conn
}

// CloseConnection closes a specific connection by ID
func Close(ctx context.Context, entityMetadata datastore.EntityMetadata) {
	if ctx == nil {
		return
	}

	mapValue := ctx.Value(CONNECTION_MAP)
	if mapValue == nil {
		return // No connections in context
	}

	connMap := mapValue.(map[string]string)
	connID, exists := connMap[entityMetadata.GetName()]
	if !exists {
		return // No connection for this entity
	}

	// Remove from context map
	delete(connMap, entityMetadata.GetName())

	// Close and unregister the connection
	registry.mu.Lock()
	defer registry.mu.Unlock()

	conn, exists := registry.connections[connID]
	if !exists {
		return // Already closed
	}

	// Perform actual close operation
	conn.ReadOptions.Destroy()
	conn.WriteOptions.Destroy()
	conn.TransactionOptions.Destroy()
	if conn.shouldRollback {
		if err := conn.Conn.Rollback(); err != nil {
			logging.Logger.Error("rollback failed", zap.Error(err))
		} // commit is expected to be done by the caller
	}

	conn.Conn.Destroy()
	delete(registry.connections, connID)
	return
}

// CloseAll closes all connections in the given context
func CloseAll(ctx context.Context) {
	mapValue := ctx.Value(CONNECTION_MAP)
	if mapValue == nil {
		return // No connections
	}

	connMap := mapValue.(map[string]string)
	for entityName, connID := range connMap {
		registry.mu.Lock()
		conn, exists := registry.connections[connID]
		if exists {
			conn.ReadOptions.Destroy()
			conn.WriteOptions.Destroy()
			conn.TransactionOptions.Destroy()
			if conn.shouldRollback {
				if err := conn.Conn.Rollback(); err != nil {
					logging.Logger.Error("rollback failed", zap.Error(err))
				} // commit is expected to be done by the caller of the get connection
			}
			conn.Conn.Destroy()
			delete(registry.connections, connID)
		}
		registry.mu.Unlock()

		delete(connMap, entityName)
	}
}
