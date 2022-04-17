package cache

import (
	"sync/atomic"

	"0chain.net/core/common"
	lru "github.com/hashicorp/golang-lru"
)

var ErrKeyNotFound = common.NewError("missing key", "key not found")

//LRU - LRU cache
type LRU struct {
	Cache *lru.Cache
	hit   int64
	miss  int64
	//lock  sync.Mutex
}

//NewLRUCache - create a new LRU cache
func NewLRUCache(size int) *LRU {
	c := &LRU{}
	c.Cache, _ = lru.New(size)

	return c
}

//Add - add a key and a value
func (c *LRU) Add(key string, value interface{}) error {
	c.Cache.Add(key, value)
	return nil
}

//Get - get the value associated with the key
func (c *LRU) Get(key string) (interface{}, error) {
	//c.lock.Lock()
	//defer c.lock.Unlock()
	value, ok := c.Cache.Get(key)
	if !ok {
		atomic.AddInt64(&c.miss, 1)
		return nil, ErrKeyNotFound
	}
	atomic.AddInt64(&c.hit, 1)
	return value, nil
}

// Remove removes the entity of given key from cache
func (c *LRU) Remove(key string) {
	c.Cache.Remove(key)
}

func (c *LRU) GetHit() int64 {
	return atomic.LoadInt64(&c.hit)
}

func (c *LRU) GetMiss() int64 {
	return atomic.LoadInt64(&c.miss)
}
