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

// CacheTx is a transaction for pre commit cache, it
// would cache all the changed values and commit when block execution is completed
type CacheTx struct {
	mu            sync.RWMutex
	cache         map[string]Value
	main          *StateCache
	blockHash     string
	prevBlockHash string
}

func NewCacheTx(main *StateCache, prevBlockHash, blockHash string) *CacheTx {
	return &CacheTx{
		cache:         make(map[string]Value),
		main:          main,
		blockHash:     blockHash,
		prevBlockHash: prevBlockHash,
	}
}

// Set sets the value with the given key in the pre-commit cache
func (pcc *CacheTx) Set(key string, value Value) {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	pcc.cache[key] = value
}

// Get returns the value with the given key
func (pcc *CacheTx) Get(key string) (Value, bool) {
	pcc.mu.RLock()
	defer pcc.mu.RUnlock()

	// Check the pre-commit cache first
	value, ok := pcc.cache[key]
	if ok && !value.Deleted {
		return value, ok
	}

	// Should not return deleted value
	if ok && value.Deleted {
		return Value{}, false
	}

	return pcc.main.Get(key, pcc.prevBlockHash)
}

// Remove marks the value with the given key as deleted in the pre-commit cache
func (pcc *CacheTx) Remove(key string) {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	value, ok := pcc.cache[key]
	if ok {
		value.Deleted = true
		pcc.cache[key] = value
		return
	}

	// If the value is not in the pre-commit cache, check it in main cache,
	// and if found mark it as deleted in the current cache
	value, ok = pcc.main.Get(key, pcc.prevBlockHash)
	if ok {
		value.Deleted = true
		pcc.cache[key] = value
	}
}

// Commit moves the values from the pre-commit cache to the main cache
func (pcc *CacheTx) Commit() {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	pcc.main.mu.Lock()
	for key, value := range pcc.cache {
		if _, ok := pcc.main.cache[key]; !ok {
			pcc.main.cache[key] = make(map[string]Value)
		}
		pcc.main.cache[key][pcc.blockHash] = value
	}

	pcc.main.shift(pcc.prevBlockHash, pcc.blockHash)
	pcc.main.mu.Unlock()

	// Clear the pre-commit cache
	pcc.cache = make(map[string]Value)
}
