package memorystore

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	log "0chain.net/core/logging"
)

// Redis host environment variables.
var (
	// DefaultPool - the default redis pool against a service (host) named redis.
	DefaultPool *redis.Pool

	connID atomic.Int64
)

// NewPool - create a new redis pool accessible at the given address.
func NewPool(host string, port int) *redis.Pool {
	var address string
	if os.Getenv("DOCKER") != "" {
		address = host + ":6379"
	} else {
		address = "127.0.0.1:" + strconv.Itoa(port)
	}
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", address)
			if err != nil {
				log.Logger.Panic(err.Error())
			}
			return c, err
		},
	}
}

type (
	dbpool struct {
		ID     string
		CtxKey common.ContextKey
		Pool   *redis.Pool
	}

	poolList struct {
		list  map[string]*dbpool
		mutex sync.RWMutex
	}
)

var pools = poolList{list: make(map[string]*dbpool)}

func InitDefaultPool(host string, port int) {
	pools.mutex.Lock()
	defer pools.mutex.Unlock()

	DefaultPool = NewPool(host, port)
	pools.list[""] = &dbpool{ID: "", CtxKey: CONNECTION, Pool: DefaultPool}
}

func getConnectionCtxKey(id string) common.ContextKey {
	key := CONNECTION
	if id != "" {
		key += common.ContextKey(id)
	}

	return key
}

// AddPool - add a database pool to the repository of db pools.
func AddPool(id string, pool *redis.Pool) {
	pools.mutex.Lock()
	defer pools.mutex.Unlock()

	pools.list[id] = &dbpool{ID: id, CtxKey: getConnectionCtxKey(id), Pool: pool}
}

func GetConnectionCount(entityMetadata datastore.EntityMetadata) (int, int) {
	pools.mutex.Lock()
	defer pools.mutex.Unlock()

	id := entityMetadata.GetDB()
	pool, ok := pools.list[id]
	if !ok {
		log.Logger.Panic("Invalid entity metadata setup, unknown dbpool: " + id)
	}

	return pool.Pool.ActiveCount(), pool.Pool.IdleCount()
}

func getdbpool(entityMetadata datastore.EntityMetadata) *dbpool {
	pools.mutex.RLock()
	defer pools.mutex.RUnlock()

	id := entityMetadata.GetDB()
	pool, ok := pools.list[id]
	if !ok {
		log.Logger.Panic("Invalid entity metadata setup, unknown dbpool: " + id)
	}

	return pool
}

// GetConnection - returns a connection from the Pool should always use right
// after getting the connection to avoid leaks defer c.Close()
func GetConnection() *Conn {
	id := connID.Add(1)
	return &Conn{Conn: DefaultPool.Get(), Tm: time.Now(), ID: id, Pool: DefaultPool}
}

// GetInfo - returns a connection from the Pool
// and will do info persistence on Redis to see the status of redis.
func GetInfo() {
	conn := DefaultPool.Get()
	defer func(conn redis.Conn) { _ = conn.Close() }(conn)

	delay := 10 * time.Second
	re := regexp.MustCompile("loading:1")
	for tries := 0; true; tries++ {
		info, err := redis.String(conn.Do("INFO", "persistence"))
		if err != nil {
			log.Logger.Panic("invalid setup")
		}
		if re.MatchString(info) {
			log.Logger.Info("Redis is not ready to take connections", zap.Any("retry", tries))
			time.Sleep(delay)
		} else {
			break
		}
	}
}

// GetEntityConnection - returns a connection from the pool configured for the entity.
func GetEntityConnection(entityMetadata datastore.EntityMetadata) *Conn {
	dbid := entityMetadata.GetDB()
	if dbid == "" {
		return GetConnection()
	}
	dbpool := getdbpool(entityMetadata)
	id := connID.Add(1)
	return &Conn{Conn: dbpool.Pool.Get(), Tm: time.Now(), Pool: dbpool.Pool, ID: id}
}

// CONNECTION - key used to get the connection object from the context.
const CONNECTION common.ContextKey = "connection."

type Conn struct {
	redis.Conn
	Tm   time.Time
	ID   int64
	Pool *redis.Pool
}

type connections map[common.ContextKey]*Conn

// WithConnection takes a context and adds a connection value to it.
func WithConnection(ctx context.Context) context.Context {
	cons := ctx.Value(CONNECTION)
	if cons == nil {
		cMap := make(connections)
		cMap[CONNECTION] = GetConnection()
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := cons.(connections)
	if !ok {
		log.Logger.Panic("invalid setup")
	}
	_, ok = cMap[CONNECTION]
	if !ok {
		cMap[CONNECTION] = GetConnection()
	}
	return ctx

}

// GetCon returns a connection stored in the context
// which got created via WithConnection.
func GetCon(ctx context.Context) *Conn {
	if ctx == nil {
		return GetConnection()
	}
	cons := ctx.Value(CONNECTION)
	if cons == nil {
		con := GetConnection()
		cMap := make(connections)
		cMap[CONNECTION] = con
		return con
	}
	cMap, ok := cons.(connections)
	if !ok {
		log.Logger.Panic("invalid setup")
	}
	con, ok := cMap[CONNECTION]
	if !ok {
		con = GetConnection()
		cMap[CONNECTION] = con
	}
	return con
}

// WithEntityConnection - returns a connection
// as per the configuration of the entity.
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	db := getdbpool(entityMetadata)
	if db.Pool == DefaultPool {
		return WithConnection(ctx)
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		cMap := make(connections)
		id := connID.Add(1)
		cMap[db.CtxKey] = &Conn{Conn: db.Pool.Get(), Tm: time.Now(), ID: id, Pool: db.Pool}
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := c.(connections)
	_, ok = cMap[db.CtxKey]
	if !ok {
		id := connID.Add(1)
		cMap[db.CtxKey] = &Conn{Conn: db.Pool.Get(), Tm: time.Now(), ID: id, Pool: db.Pool}
	}
	return ctx

}

// GetEntityCon returns a connection stored in the context
// which got created via WithEntityConnection.
func GetEntityCon(ctx context.Context, entityMetadata datastore.EntityMetadata) *Conn {
	if ctx == nil {
		return GetEntityConnection(entityMetadata)
	}
	db := getdbpool(entityMetadata)
	if db.Pool == DefaultPool {
		return GetCon(ctx)
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		return nil
	}
	cMap, ok := c.(connections)

	con, ok := cMap[db.CtxKey]
	if !ok {
		con = GetEntityConnection(entityMetadata)
		cMap[db.CtxKey] = con
	}
	return con
}

// Close - Close takes care of maintaining the closing of connection(s)
// stored in the context.
func Close(ctx context.Context) {
	c := ctx.Value(CONNECTION)
	if c == nil {
		log.Logger.Error("Connection is nil while closing")
		return
	}
	cMap := c.(connections)
	for _, con := range cMap {
		err := con.Close()
		if err != nil {
			log.Logger.Error("Connection not closed", zap.Error(err))
		}
	}
}
