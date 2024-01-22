package statecache

// QueryBlockCache is a read-only cache for querying values from the current block.
type QueryBlockCache struct {
	sc        *StateCache
	blockHash string
}

// NewQueryBlockCache creates a new QueryBlockCache instance.
func NewQueryBlockCache(sc *StateCache, blockHash string) *QueryBlockCache {
	return &QueryBlockCache{
		sc:        sc,
		blockHash: blockHash,
	}
}

// Get returns the value with the given key from the current block cache.
func (qbc *QueryBlockCache) Get(key string) (Value, bool) {
	return qbc.sc.Get(key, qbc.blockHash)
}

func (qbc *QueryBlockCache) Round() int64 {
	panic("Round should not be called on QueryBlockCache")
}

func (qbc *QueryBlockCache) Commit() {
	panic("Commit should not be called on QueryBlockCache")
}

// setValue implements BlockCacher.
func (*QueryBlockCache) setValue(key string, v valueNode) {
	panic("setValue should not be called on QueryBlockCache")
}
