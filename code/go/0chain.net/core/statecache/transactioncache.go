package statecache

import "sync"

type TransactionCache struct {
	main  *BlockCache
	cache map[string]valueNode
	mu    sync.RWMutex
	round int64
}

func NewTransactionCache(main *BlockCache) *TransactionCache {
	return &TransactionCache{
		main:  main,
		cache: make(map[string]valueNode),
		round: main.round,
	}
}

func (tc *TransactionCache) Set(key string, e Value) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache[key] = valueNode{
		data:  e,
		round: tc.round,
	}
}

func (tc *TransactionCache) Get(key string) (Value, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	value, ok := tc.cache[key]
	if ok {
		return value.data, ok
	}

	return tc.main.Get(key)
}

func (tc *TransactionCache) Remove(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	value, ok := tc.cache[key]
	if ok {
		value.deleted = true
		tc.cache[key] = value
	}
}

func (tc *TransactionCache) Commit() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for key, value := range tc.cache {
		tc.main.setValue(key, value)
	}

	// Clear the transaction cache
	tc.cache = make(map[string]valueNode)
}
