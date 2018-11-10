package cache

import (
	"sync"

	"0chain.net/common"
	"github.com/hashicorp/golang-lru"
)

//LRU - LRU cache
type LRU struct {
	Cache *lru.Cache
	Hit   int64
	Miss  int64
	lock  sync.Mutex
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
	c.lock.Lock()
	defer c.lock.Unlock()
	value, ok := c.Cache.Get(key)
	if !ok {
		c.Miss++
		return nil, common.NewError("missing key", "key not found")
	}
	c.Hit++
	return value, nil
}

func (c *LRU) GetHit() int64 {
	return c.Hit
}

func (c *LRU) GetMiss() int64 {
	return c.Miss
}
