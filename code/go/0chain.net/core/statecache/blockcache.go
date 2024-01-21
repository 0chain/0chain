package statecache

import "sync"

// BlockCache is a pre commit cache for all changes in a block.
// Call `Commit()` method to merge
// the changes to the StateCache when the block is executed.
type BlockCache struct {
	mu            sync.RWMutex
	cache         map[string]Value
	main          *StateCache
	blockHash     string
	prevBlockHash string
	Round         int64
}

type Block struct {
	Round    int64  // round number when this block cache is created
	Hash     string // block hash
	PrevHash string // previous hash of the block
}

func NewBlockCache(main *StateCache, blockHash Block) *BlockCache {
	return &BlockCache{
		cache:         make(map[string]Value),
		main:          main,
		blockHash:     blockHash.Hash,
		prevBlockHash: blockHash.PrevHash,
		Round:         blockHash.Round,
	}
}

// Set sets the value with the given key in the pre-commit cache
func (pcc *BlockCache) Set(key string, value Value) {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	pcc.cache[key] = value
}

// Get returns the value with the given key
func (pcc *BlockCache) Get(key string) (Value, bool) {
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
func (pcc *BlockCache) Remove(key string) {
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
func (pcc *BlockCache) Commit() {
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
