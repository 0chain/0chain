package memorystore

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

/* Redis host environment variables


 */
/*DefaultPool - the default redis pool against a service (host) named redis */
var DefaultPool *redis.Pool

var connID atomic.Int64

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

type dbpool struct {
	ID     string
	CtxKey common.ContextKey
	Pool   *redis.Pool
}

// idStats collects redis connection ids and those not closed in specific time
type idStats struct {
	ids map[int64]time.Time
	sync.Mutex
}

func newIDStats() *idStats {
	return &idStats{
		ids: make(map[int64]time.Time),
	}
}

func (s *idStats) Add(id int64) {
	s.Lock()
	s.ids[id] = time.Now()
	s.Unlock()
}

func (s *idStats) Del(id int64) {
	s.Lock()
	delete(s.ids, id)
	Logger.Debug("idState removes id", zap.Int64("id", id))
	s.Unlock()
}

func (s *idStats) CheckExpiredIDs() {
	var expiredIDS []int64
	s.Lock()
	for id, t := range s.ids {
		if time.Since(t) > 3*time.Second {
			expiredIDS = append(expiredIDS, id)
		}
	}
	s.Unlock()
	sort.Slice(expiredIDS, func(i, j int) bool {
		return expiredIDS[i] < expiredIDS[j]
	})
	Logger.Debug("connections still in use", zap.Int64s("ids", expiredIDS))
}

var pools = make(map[string]*dbpool)
var idS *idStats

func init() {
	DefaultPool = NewPool(os.Getenv("REDIS_HOST"), 6379)
	pools[""] = &dbpool{ID: "", CtxKey: CONNECTION, Pool: DefaultPool}
	tkt := time.NewTicker(3 * time.Second)
	idS = newIDStats()
	go func() {
		for {
			select {
			case <-tkt.C:
				idS.CheckExpiredIDs()
			default:
			}
		}
	}()
}

func getConnectionCtxKey(dbid string) common.ContextKey {
	if dbid == "" {
		return CONNECTION
	}
	return common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, dbid))
}

/*AddPool - add a database pool to the repository of db pools */
func AddPool(dbid string, pool *redis.Pool) {
	dbpool := &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}
	pools[dbid] = dbpool
}

func GetConnectionCount(entityMetadata datastore.EntityMetadata) (int, int) {
	dbid := entityMetadata.GetDB()
	dbpool, ok := pools[dbid]
	if !ok {
		panic(fmt.Sprintf("Invalid entity metadata setup, unknown dbpool %v\n", dbid))
	}
	return dbpool.Pool.ActiveCount(), dbpool.Pool.IdleCount()
}

func getdbpool(entityMetadata datastore.EntityMetadata) *dbpool {
	dbid := entityMetadata.GetDB()
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
func GetConnection() *Conn {
	st := DefaultPool.Stats()
	id := connID.Add(1)
	idS.Add(id)
	Logger.Debug("GetConnection defualt redis pool stats",
		zap.Int("active", st.ActiveCount),
		zap.Int("idle", st.IdleCount),
		zap.Int64("id", id))

	return &Conn{Conn: DefaultPool.Get(), Tm: time.Now(), ID: id, Pool: DefaultPool}
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
			Logger.Info("Redis is not ready to take connections", zap.Any("retry", tries))
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
	dbpool := getdbpool(entityMetadata)
	id := connID.Add(1)
	idS.Add(id)
	st := dbpool.Pool.Stats()
	Logger.Debug("GetEntityConnection redis pool stats",
		zap.Int("active", st.ActiveCount),
		zap.Int("idle", st.IdleCount),
		zap.Int64("id", id),
	)
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

type connections map[common.ContextKey]*Conn

/*WithConnection takes a context and adds a connection value to it */
func WithConnection(ctx context.Context) context.Context {
	cons := ctx.Value(CONNECTION)
	if cons == nil {
		cMap := make(connections)
		cMap[CONNECTION] = GetConnection()
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := cons.(connections)
	if !ok {
		panic("invalid setup")
	}
	_, ok = cMap[CONNECTION]
	if !ok {
		cMap[CONNECTION] = GetConnection()
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
		cMap := make(connections)
		cMap[CONNECTION] = con
		return con
	}
	cMap, ok := cons.(connections)
	if !ok {
		panic("invalid setup")
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
		id := connID.Add(1)
		idS.Add(id)
		st := dbpool.Pool.Stats()
		Logger.Debug("WithEntityConnection redis pool stats",
			zap.Int("active", st.ActiveCount),
			zap.Int("idle", st.IdleCount),
			zap.Int64("id", id))
		cMap[dbpool.CtxKey] = &Conn{Conn: dbpool.Pool.Get(), Tm: time.Now(), ID: id, Pool: dbpool.Pool}
		return context.WithValue(ctx, CONNECTION, cMap)
	}
	cMap, ok := c.(connections)
	_, ok = cMap[dbpool.CtxKey]
	if !ok {
		id := connID.Add(1)
		idS.Add(id)
		st := dbpool.Pool.Stats()
		Logger.Debug("WithEntityConnection redis pool stats",
			zap.Int("active", st.ActiveCount),
			zap.Int("idle", st.IdleCount),
			zap.Int64("id", id))
		cMap[dbpool.CtxKey] = &Conn{Conn: dbpool.Pool.Get(), Tm: time.Now(), ID: id, Pool: dbpool.Pool}
	}
	return ctx

}

/*GetEntityCon returns a connection stored in the context which got created via WithEntityConnection */
func GetEntityCon(ctx context.Context, entityMetadata datastore.EntityMetadata) *Conn {
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
		Logger.Error("Connection is nil while closing")
		return
	}
	cMap := c.(connections)
	for ck, con := range cMap {
		err := con.Close()
		if err != nil {
			Logger.Error("Connection not closed", zap.Error(err))
		}

		st := con.Pool.Stats()
		idS.Del(con.ID)
		Logger.Debug("Close redis connections",
			zap.Any("context key", ck),
			zap.Any("connection duration", time.Since(con.Tm)),
			zap.Int64("id", con.ID),
			zap.Int("active", st.ActiveCount),
			zap.Int("idle", st.IdleCount))
	}
}
