package statecache

import (
	"fmt"
	"sort"
	"sync"
)

// NewBlockTxnCaches creates a new block cache and a transaction cache for the given block
func NewBlockTxnCaches(sc *StateCache, b Block) (*BlockCache, *TransactionCache) {
	bc := NewBlockCache(sc, b)
	tc := NewTransactionCache(bc)
	return bc, tc
}

// Value is an interface that all values in the state cache must implement
type Value interface {
	Clone() interface{}
}

type String string

func (se String) Clone() interface{} {
	return se
}

type valueNode struct {
	data    Value
	deleted bool  // indicates the value was removed
	round   int64 // round number when this value is updated
}

type StateCache struct {
	mu    sync.RWMutex
	cache map[string]map[string]valueNode
}

func NewStateCache() *StateCache {
	return &StateCache{
		cache: make(map[string]map[string]valueNode),
	}
}

// Get returns the value with the given key and block hash
func (sc *StateCache) Get(key, blockHash string) (Value, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	blockValues, ok := sc.cache[key]
	if !ok {
		return nil, false
	}

	v, ok := blockValues[blockHash]
	if ok && !v.deleted {
		return v.data, true
	}
	return nil, false
}

func (sc *StateCache) getValue(key, blockHash string) (valueNode, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	blockValues, ok := sc.cache[key]
	if !ok {
		return valueNode{}, false
	}

	v, ok := blockValues[blockHash]
	if ok && !v.deleted {
		return v, true
	}
	return valueNode{}, false
}

// shift copy the value from previous block to current
func (sc *StateCache) shift(prevHash, blockHash string) {
	for key, blockValues := range sc.cache {
		v, ok := blockValues[prevHash]
		if ok {
			if _, exists := blockValues[blockHash]; !exists {
				if sc.cache[key] == nil {
					sc.cache[key] = make(map[string]valueNode)
				}
				sc.cache[key][blockHash] = v
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

// PruneRoundBelow removes all values that are below the given round
func (sc *StateCache) PruneRoundBelow(round int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for key, blockValues := range sc.cache {
		for blockHash, value := range blockValues {
			if value.round < round {
				delete(blockValues, blockHash)
			}
		}

		// Delete the map if it becomes empty
		if len(blockValues) == 0 {
			delete(sc.cache, key)
		}
	}
}

// PrettyPrint prints the state cache in a pretty format
func (sc *StateCache) PrettyPrint() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Sort keys in alphabetical order
	var keys []string
	for key := range sc.cache {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Print values for each key
	for _, key := range keys {
		fmt.Printf("Key: %s\n", key)

		blockValues := sc.cache[key]

		// Sort block hashes by round number in descending order
		var rounds []int64
		for _, value := range blockValues {
			rounds = append(rounds, value.round)
		}
		sort.Slice(rounds, func(i, j int) bool {
			return rounds[i] > rounds[j]
		})

		// Print values for each round
		for _, round := range rounds {
			fmt.Printf("  Round: %d\n", round)

			// Sort block hashes for the same round
			var hashes []string
			for hash, value := range blockValues {
				if value.round == round {
					hashes = append(hashes, hash)
				}
			}
			sort.Strings(hashes)

			// Print values for each hash
			for _, hash := range hashes {
				value := blockValues[hash]
				fmt.Printf("    Hash: %s\n", hash)
				fmt.Printf("      Deleted: %v\n", value.deleted)
			}
		}
	}
}
