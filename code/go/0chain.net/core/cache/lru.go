package cache

import (
	"sync/atomic"

	"0chain.net/core/common"
	lru "github.com/hashicorp/golang-lru/v2"
)

var ErrKeyNotFound = common.NewError("missing key", "key not found")

//LRU - LRU cache
type LRU[K comparable, V any] struct {
	Cache *lru.Cache[K, V]
	hit   int64
	miss  int64
	//lock  sync.Mutex
}

//NewLRUCache - create a new LRU cache
func NewLRUCache[K comparable, V any](size int) *LRU[K, V] {
	c := &LRU[K, V]{}
	c.Cache, _ = lru.New[K, V](size)

	return c
}

//Add - add a key and a value
func (c *LRU[K, V]) Add(key K, value V) error {
	c.Cache.Add(key, value)
	return nil
}

//Get - get the value associated with the key
func (c *LRU[K, V]) Get(key K) (interface{}, error) {
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
func (c *LRU[K, V]) Remove(key K) {
	c.Cache.Remove(key)
}

func (c *LRU[K, V]) GetHit() int64 {
	return atomic.LoadInt64(&c.hit)
}

func (c *LRU[K, V]) GetMiss() int64 {
	return atomic.LoadInt64(&c.miss)
}
