package memorystore

import (
	"fmt"
	"os"
	"sync"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/gomodule/redigo/redis"
)

var DefaultPool *redis.Pool

// NewPool - create a new redis pool accessible at the given address
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

type (
	dbPool struct {
		ID     string
		CtxKey common.ContextKey
		Pool   *redis.Pool
	}

	poolList struct {
		list  map[string]*dbPool
		mutex sync.RWMutex
	}
)

var pools = poolList{list: make(map[string]*dbPool)}

func (p *poolList) initDefaultPool(host string, port int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	DefaultPool = NewPool(host, port)
	p.list[""] = &dbPool{ID: "", CtxKey: CONNECTION, Pool: DefaultPool}
}

func (p *poolList) addPool(id string, pool *redis.Pool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.list[id] = &dbPool{ID: id, CtxKey: getConnectionCtxKey(id), Pool: pool}
}

func (p *poolList) getDbPool(entityMetadata datastore.EntityMetadata) *dbPool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	id := entityMetadata.GetDB()
	pool, ok := p.list[id]
	if !ok {
		logging.Logger.Panic("Invalid entity metadata setup, unknown db pool: " + id)
	}

	return pool
}
