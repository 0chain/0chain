package memorystore

import (
	"context"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
	"github.com/gomodule/redigo/redis"
)

/*NewPool - create a new redis pool accessible at the given address */
func NewPool(address string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", address)
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

type dbpool struct {
	ID     string
	CtxKey common.ContextKey
	Pool   *redis.Pool
}

var pools = make(map[string]*dbpool)
var DefaultPool = NewPool(":6379")

func init() {
	pools[""] = &dbpool{ID: "", CtxKey: CONNECTION, Pool: DefaultPool}
}

func getConnectionCtxKey(dbid string) common.ContextKey {
	return common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, dbid))
}

/*AddPool - add a database pool to the repository of db pools */
func AddPool(dbid string, pool *redis.Pool) {
	dbpool := &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}
	pools[dbid] = dbpool
}

func getdbpool(entityMetadata datastore.EntityMetadata) *dbpool {
	dbid := entityMetadata.GetMemoryDB()
	dbpool, ok := pools[dbid]
	if !ok {
		panic(fmt.Sprintf("Invalid entity metadata setup, unknown dbpool %v\n", dbid))
	}
	return dbpool
}

/*GetConnection - returns a connection from the Pool
* Should always use right after getting the connection to avoid leaks
* defer c.Close()
 */
func GetConnection() redis.Conn {
	return DefaultPool.Get()
}

/*GetEntityConnection - retuns a connection from the pool configured for the entity */
func GetEntityConnection(entityMetadata datastore.EntityMetadata) redis.Conn {
	dbid := entityMetadata.GetMemoryDB()
	if dbid == "" {
		return GetConnection()
	}
	dbpool := getdbpool(entityMetadata)
	return dbpool.Pool.Get()
}

/*CONNECTION - key used to get the connection object from the context */
const CONNECTION common.ContextKey = "connection."

/*WithConnection takes a context and adds a connection value to it */
func WithConnection(ctx context.Context) context.Context {
	return context.WithValue(ctx, CONNECTION, GetConnection())
}

/*GetCon returns a connection stored in the context which got created via WithConnection */
func GetCon(ctx context.Context) redis.Conn {
	if ctx == nil {
		return GetConnection()
	}
	return ctx.Value(CONNECTION).(redis.Conn)
}

/*WithEntityConnection - returns a connection as per the configuration of the entity */
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	dbpool := getdbpool(entityMetadata)
	return context.WithValue(ctx, dbpool.CtxKey, dbpool.Pool.Get())
}

/*GetEntityCon returns a connection stored in the context which got created via WithEntityConnection */
func GetEntityCon(ctx context.Context, entityMetadata datastore.EntityMetadata) redis.Conn {
	if ctx == nil {
		return GetEntityConnection(entityMetadata)
	}
	dbpool := getdbpool(entityMetadata)
	return ctx.Value(dbpool.CtxKey).(redis.Conn)
}

/*Close - Close takes care of maintaining the closing of connection(s) stored in the context */
func Close(ctx context.Context) {
	// TODO:
}
