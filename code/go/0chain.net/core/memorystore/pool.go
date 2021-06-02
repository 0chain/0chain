package memorystore

import (
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"os"
	"sync"
)

var DefaultPool *redis.Pool

/*NewPool - create a new redis pool accessible at the given address */
func NewPool(host string, port int) *redis.Pool {
	var address string
	if os.Getenv("DOCKER") != "" {
		address = fmt.Sprintf("%v:6379", host)
	} else {
		address = fmt.Sprintf("127.0.0.1:%v", port)
	}
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

func InitDefaultPool(host string, port int) {
	pools.initDefaultPool(host, port)
}

func AddPool(dbid string, pool *redis.Pool) {
	pools.addPool(dbid, pool)
}

func getConnectionCount(entityMetadata datastore.EntityMetadata) (int, int) {
	return pools.getConnectionCount(entityMetadata)
}

func getDbPool(entityMetadata datastore.EntityMetadata) *dbpool {
	return pools.getDbPool(entityMetadata)
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

func (p *poolList) initDefaultPool(host string, port int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	DefaultPool = NewPool(host, port)
	p.list[""] = &dbpool{ID: "", CtxKey: CONNECTION, Pool: DefaultPool}
}

// AddPool - add a database pool to the repository of db pools.
func (p *poolList) addPool(id string, pool *redis.Pool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.list[id] = &dbpool{ID: id, CtxKey: getConnectionCtxKey(id), Pool: pool}
}

func (p *poolList) getConnectionCount(entityMetadata datastore.EntityMetadata) (int, int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	id := entityMetadata.GetDB()
	pool, ok := p.list[id]
	if !ok {
		logging.Logger.Panic("Invalid entity metadata setup, unknown db pool: " + id)
	}

	return pool.Pool.ActiveCount(), pool.Pool.IdleCount()
}

func (p *poolList) getDbPool(entityMetadata datastore.EntityMetadata) *dbpool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	id := entityMetadata.GetDB()
	pool, ok := p.list[id]
	if !ok {
		logging.Logger.Panic("Invalid entity metadata setup, unknown db pool: " + id)
	}

	return pool
}
