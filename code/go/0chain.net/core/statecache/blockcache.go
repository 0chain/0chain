package statecache

import (
	"sync"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type BlockCacher interface {
	Get(key string) (Value, bool)
	Round() int64
	Commit()
	setValue(key string, v valueNode)
}

// BlockCache is a pre commit cache for all changes in a block.
// This is mainly for caching values in current block when executing blocks.
//
// Querying from this BlockCache will only return value from current block, and previous block if not found
// in current block. That means if there are no changes happen in current block yet,
// querying the value from current block hash will return nothing even the StateCache does has the value.
// So please remember to use QueryBlockCache to get values from current block.
//
// Call `Commit()` method to merge
// the changes to the StateCache when the block is executed.
type BlockCache struct {
	mu            sync.RWMutex
	cache         map[string]valueNode
	main          *StateCache
	blockHash     string
	prevBlockHash string
	round         int64
}

type Block struct {
	Round    int64  // round number when this block cache is created
	Hash     string // block hash
	PrevHash string // previous hash of the block
}

func NewBlockCache(main *StateCache, b Block) *BlockCache {
	return &BlockCache{
		cache:         make(map[string]valueNode),
		main:          main,
		blockHash:     b.Hash,
		prevBlockHash: b.PrevHash,
		round:         b.Round,
	}
}

// Set sets the value with the given key in the pre-commit cache
func (pcc *BlockCache) Set(key string, e Value) {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	pcc.cache[key] = valueNode{
		data:  e.Clone(),
		round: pcc.round,
	}
}

func (pcc *BlockCache) Round() int64 {
	return pcc.round
}

func (pcc *BlockCache) setValue(key string, v valueNode) {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	v.data = v.data.Clone()
	pcc.cache[key] = v
}

// Get returns the value with the given key
func (pcc *BlockCache) Get(key string) (Value, bool) {
	pcc.mu.RLock()
	defer pcc.mu.RUnlock()

	// Check the pre-commit cache first
	value, ok := pcc.cache[key]
	if ok && !value.deleted {
		logging.Logger.Debug("block cache get", zap.String("key", key))
		return value.data.Clone(), ok
	}

	// Should not return deleted value
	if ok && value.deleted {
		logging.Logger.Debug("block cache get - deleted", zap.String("key", key))
		return nil, false
	}

	return pcc.main.Get(key, pcc.prevBlockHash)
}

// Remove marks the value with the given key as deleted in the pre-commit cache
func (pcc *BlockCache) remove(key string) {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	value, ok := pcc.cache[key]
	if ok {
		value.deleted = true
		pcc.cache[key] = value
		return
	} else {
		pcc.cache[key] = valueNode{
			deleted: true,
			round:   pcc.round,
		}
	}
}

// Commit moves the values from the pre-commit cache to the main cache
func (pcc *BlockCache) Commit() {
	pcc.mu.Lock()
	defer pcc.mu.Unlock()

	pcc.main.mu.Lock()
	for key, v := range pcc.cache {
		if _, ok := pcc.main.cache[key]; !ok {
			pcc.main.cache[key] = make(map[string]valueNode)
		}
		v.data = v.data.Clone()
		pcc.main.cache[key][pcc.blockHash] = v
	}

	pcc.main.shift(pcc.prevBlockHash, pcc.blockHash)
	pcc.main.mu.Unlock()

	// Clear the pre-commit cache
	pcc.cache = make(map[string]valueNode)
}
