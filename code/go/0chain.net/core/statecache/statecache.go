package statecache

import (
	"sync"
)

type Value struct {
	// Define your value type here
	Data    []byte
	Deleted bool // indicates the value was removed
}

type StateCache struct {
	mu    sync.RWMutex
	cache map[string]map[string]Value
}

func NewStateCache() *StateCache {
	return &StateCache{
		cache: make(map[string]map[string]Value),
	}
}

// Get returns the value with the given key and block hash
func (sc *StateCache) Get(key, blockHash string) (Value, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	blockValues, ok := sc.cache[key]
	if !ok {
		return Value{}, false
	}

	v, ok := blockValues[blockHash]
	if ok && !v.Deleted {
		return v, true
	}
	return Value{}, false
}

// shift moves the value from previous block hash to current if current not exists
func (sc *StateCache) shift(prevHash, blockHash string) {
	for key, blockValues := range sc.cache {
		value, ok := blockValues[prevHash]
		if ok {
			if _, exists := blockValues[blockHash]; !exists {
				if sc.cache[key] == nil {
					sc.cache[key] = make(map[string]Value)
				}
				sc.cache[key][blockHash] = value
				// delete(sc.cache[key], prevHash)
			}
		}
	}
}

// Remove removes the values map with the given key
func (sc *StateCache) Remove(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	delete(sc.cache, key)
}
