package memorystore

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "github.com/0chain/common/core/logging"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

var connID atomic.Int64

/*GetConnection - returns a connection from the Pool
* Should always use right after getting the connection to avoid leaks
* defer c.Close()
 */
func GetConnection() *Conn {
	id := connID.Add(1)
	return &Conn{Conn: DefaultPool.Get(), Tm: time.Now(), ID: id, Pool: DefaultPool}
}

func getConnectionCtxKey(dbid string) common.ContextKey {
	if dbid == "" {
		return CONNECTION
	}
	return common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, dbid))
}

/*GetInfo - returns a connection from the Pool and will do info persistence on Redis to see the status of redis
 */
func GetInfo() {
	conn := DefaultPool.Get()
	defer conn.Close()
	delay := 10 * time.Second
	re := regexp.MustCompile("loading:1")
	for tries := 0; true; tries++ {
		info, err := redis.String(conn.Do("INFO", "persistence"))
		if err != nil {
			panic("invalid setup")
		}
		if re.MatchString(info) {
			Logger.Info("Redis is not ready to take connections", zap.Int("retry", tries))
			time.Sleep(delay)
		} else {
			break
		}
	}
}

/*GetEntityConnection - returns a connection from the pool configured for the entity */
func GetEntityConnection(entityMetadata datastore.EntityMetadata) *Conn {
	dbid := entityMetadata.GetDB()
	if dbid == "" {
		return GetConnection()
	}
	dbpool := pools.getDbPool(entityMetadata)
	id := connID.Add(1)
	return &Conn{Conn: dbpool.Pool.Get(), Tm: time.Now(), Pool: dbpool.Pool, ID: id}
}

/*CONNECTION - key used to get the connection object from the context */
const CONNECTION common.ContextKey = "connection."

type Conn struct {
	redis.Conn
	Tm   time.Time
	ID   int64
	Pool *redis.Pool
}

type connections struct {
	cons  map[common.ContextKey]*Conn
	mutex *sync.RWMutex
}

func newConnections() connections {
	return connections{
		cons:  make(map[common.ContextKey]*Conn),
		mutex: &sync.RWMutex{},
	}
}

func (c *connections) set(key common.ContextKey, conn *Conn) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cons[key] = conn
}

func (c *connections) get(key common.ContextKey) (*Conn, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	conn, ok := c.cons[key]
	return conn, ok
}

func (c *connections) iterateCons(f func(*Conn)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, conn := range c.cons {
		f(conn)
	}
}

/*WithConnection takes a context and adds a connection value to it */
func WithConnection(ctx context.Context) context.Context {
	cons := ctx.Value(CONNECTION)
	if cons == nil {
		cMap := newConnections()
		cMap.set(CONNECTION, GetConnection())
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := cons.(connections)
	if !ok {
		panic("invalid setup")
	}
	_, ok = cMap.get(CONNECTION)
	if !ok {
		cMap.set(CONNECTION, GetConnection())
	}
	return ctx

}

/*GetCon returns a connection stored in the context which got created via WithConnection */
func GetCon(ctx context.Context) *Conn {
	if ctx == nil {
		return GetConnection()
	}
	cons := ctx.Value(CONNECTION)
	if cons == nil {
		con := GetConnection()
		cMap := newConnections()
		cMap.set(CONNECTION, con)
		return con
	}
	cMap, ok := cons.(connections)
	if !ok {
		panic("invalid setup")
	}
	con, ok := cMap.get(CONNECTION)
	if !ok {
		con = GetConnection()
		cMap.set(CONNECTION, con)
	}
	return con
}

/*WithEntityConnection - returns a connection as per the configuration of the entity */
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	dbpool := pools.getDbPool(entityMetadata)
	if dbpool.Pool == DefaultPool {
		return WithConnection(ctx)
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		cMap := newConnections()
		id := connID.Add(1)
		cMap.set(dbpool.CtxKey, &Conn{Conn: dbpool.Pool.Get(), Tm: time.Now(), ID: id, Pool: dbpool.Pool})
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := c.(connections)
	if ok {
		_, ok = cMap.get(dbpool.CtxKey)
		if !ok {
			id := connID.Add(1)
			cMap.set(dbpool.CtxKey, &Conn{Conn: dbpool.Pool.Get(), Tm: time.Now(), ID: id, Pool: dbpool.Pool})
		}
	}
	return ctx

}

/*GetEntityCon returns a connection stored in the context which got created via WithEntityConnection */
func GetEntityCon(ctx context.Context, entityMetadata datastore.EntityMetadata) *Conn {
	if ctx == nil {
		return GetEntityConnection(entityMetadata)
	}
	dbpool := pools.getDbPool(entityMetadata)
	if dbpool.Pool == DefaultPool {
		return GetCon(ctx)
	}
	c := ctx.Value(CONNECTION)
	if c == nil {
		return nil
	}
	cMap, ok := c.(connections)
	if !ok {
		return nil
	}

	con, ok := cMap.get(dbpool.CtxKey)
	if !ok {
		con = GetEntityConnection(entityMetadata)
		cMap.set(dbpool.CtxKey, con)
	}
	return con
}

/*Close - Close takes care of maintaining the closing of connection(s) stored in the context */
func Close(ctx context.Context) {
	c := ctx.Value(CONNECTION)
	if c == nil {
		Logger.Error("Connection is nil while closing")
		return
	}
	cMap := c.(connections)
	cMap.iterateCons(func(con *Conn) {
		err := con.Close()
		if err != nil {
			Logger.Error("Connection not closed", zap.Error(err))
		}
	})
}
