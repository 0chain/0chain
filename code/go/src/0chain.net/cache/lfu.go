package cache

import (
	"github.com/koding/cache"
	"sync"
)

func GetLFUCacheProvider() *LFU {
	return &LFU{}
}

type LFU struct {
	Cache cache.Cache
	Hit   int64
	Miss  int64
	lock  sync.Mutex
}

func (c *LFU) New(size int) {
	c.Cache = cache.NewLFU(size)
	c.Hit = 0
	c.Miss = 0
}

func (c *LFU) Add(key string, value interface{}) error {
	return c.Cache.Set(key, value)
}

func (c *LFU) Get(key string) (interface{}, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	value, err := c.Cache.Get(key)
	if err != nil {
		c.Miss++
		return nil, err
	}
	c.Hit++
	return value, err
}

func (c *LFU) GetHit() int64 {
	return c.Hit
}

func (c *LFU) GetMiss() int64 {
	return c.Miss
}
