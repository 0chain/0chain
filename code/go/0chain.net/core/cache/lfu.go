package cache

import (
	"sync"

	"github.com/koding/cache"
)

type LFU struct {
	Cache cache.Cache
	Hit   int64
	Miss  int64
	lock  sync.Mutex
}

// NewLFUCache - create a new LFU cache object
func NewLFUCache(size int) *LFU {
	c := &LFU{}
	c.Cache = cache.NewLFU(size)
	return c
}

// Add - add a given key and value
func (c *LFU) Add(key string, value interface{}) error {
	return c.Cache.Set(key, value)
}

// Get - get the value associated with the key
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
