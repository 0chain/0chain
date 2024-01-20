package statecache

import "sync"

type TransactionCache struct {
	main  *BlockCache
	cache map[string]Value
	mu    sync.RWMutex
	round int64
}

func NewTransactionCache(main *BlockCache) *TransactionCache {
	return &TransactionCache{
		main:  main,
		cache: make(map[string]Value),
		round: main.Round,
	}
}

func (tc *TransactionCache) Set(key string, value Value) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	value.Round = tc.round

	tc.cache[key] = value
}

func (tc *TransactionCache) Get(key string) (Value, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	value, ok := tc.cache[key]
	if ok {
		return value, ok
	}

	return tc.main.Get(key)
}

func (tc *TransactionCache) Remove(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	value, ok := tc.cache[key]
	if ok {
		value.Deleted = true
		tc.cache[key] = value
	}
}

func (tc *TransactionCache) Commit() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for key, value := range tc.cache {
		tc.main.Set(key, value)
	}

	// Clear the transaction cache
	tc.cache = make(map[string]Value)
}
