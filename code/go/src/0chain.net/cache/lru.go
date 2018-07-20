package cache

import (
	"0chain.net/common"
	"github.com/hashicorp/golang-lru"
	"sync"
)

func GetLRUCacheProvider() *LRU {
	return &LRU{}
}

type LRU struct {
	Cache *lru.Cache
	Hit   int64
	Miss  int64
	lock  sync.Mutex
}

func (c *LRU) New(size int) {
	c.Cache, _ = lru.New(size)
	c.Hit = 0
	c.Miss = 0
}

func (c *LRU) Add(key string, value interface{}) error {
	c.Cache.Add(key, value)
	return nil
}

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
