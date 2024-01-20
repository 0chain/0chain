package statecache

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateCache_Get(t *testing.T) {
	sc := NewStateCache()

	// Test Get method when cache is empty
	_, ok := sc.Get("key1", "hash1")
	if ok {
		t.Error("Expected false, got ", ok)
	}

	// Add a value to the cache
	value := Value{Data: []byte("data1"), Deleted: false}
	sc.cache["key1"] = map[string]Value{"hash1": value}

	// Test Get method when cache has a value
	v, ok := sc.Get("key1", "hash1")
	if !ok {
		t.Error("Expected true, got ", ok)
	}
	if string(v.Data) != "data1" {
		t.Error("Expected data1, got ", v.Data)
	}

	// Test Get method when cache has a deleted value
	value.Deleted = true
	sc.cache["key1"]["hash1"] = value
	_, ok = sc.Get("key1", "hash1")
	if ok {
		t.Error("Expected false, got ", ok)
	}
}

func TestCacheTx(t *testing.T) {
	sc := NewStateCache()
	ct := NewBlockCache(sc, "prevHash", "hash1")

	// Test Get method when cache is empty
	_, ok := ct.Get("key1")
	if ok {
		t.Error("Expected false, got ", ok)
	}

	// Test Set method
	ct.Set("key1", Value{Data: []byte("value1")})
	value, ok := ct.Get("key1")
	require.True(t, ok)
	require.Equal(t, "value1", string(value.Data))

	// Test Commit method
	ct.Set("key2", Value{Data: []byte("value2")})

	ct.Commit()
	value, ok = sc.Get("key2", "hash1")
	require.True(t, ok)
	require.Equal(t, "value2", string(value.Data))

	// Add a new value to the cache for key1 in hash2 block
	value2 := Value{Data: []byte("data2")}
	ct2 := NewBlockCache(sc, "hash1", "hash2")
	ct2.Set("key1", value2)
	ct2.Commit()

	// Get cache in current block
	v2, ok := sc.Get("key1", "hash2")
	require.True(t, ok)
	require.Equal(t, "data2", string(v2.Data))

	// Get cache in prior block
	v1, ok := sc.Get("key1", "hash1")
	require.True(t, ok)
	require.Equal(t, "value1", string(v1.Data))
}

func TestCacheTx_NotCommitted(t *testing.T) {
	sc := NewStateCache()
	ct := NewBlockCache(sc, "", "hash1")

	// Test Get method when cache is empty
	_, ok := ct.Get("key1")
	if ok {
		t.Error("Expected false, got ", ok)
	}

	// Test Set method
	ct.Set("key1", Value{Data: []byte("value1")})
	value, ok := ct.Get("key1")
	require.True(t, ok)
	require.Equal(t, "value1", string(value.Data))

	// Test Get method in state cache before committing
	_, ok = sc.Get("key1", "hash1")
	if ok {
		t.Error("Expected false, got ", ok)
	}

	// Test Remove method
	ct.Remove("key1")
	_, ok = ct.Get("key1")
	if ok {
		t.Error("Expected false, got ", ok)
	}

	// Test Get method in state cache before committing
	_, ok = sc.Get("key1", "hash1")
	if ok {
		t.Error("Expected false, got ", ok)
	}
}

func TestCacheTx_SkipBlock(t *testing.T) {
	sc := NewStateCache()
	ct := NewBlockCache(sc, "", "hash1")

	// Add values to the cache in block "hash1"
	ct.Set("key1", Value{Data: []byte("value1")})

	// Commit the changes to the main cache
	ct.Commit()

	// Skip one block
	ct = NewBlockCache(sc, "hash2", "hash3")

	_, ok := ct.Get("key1")
	require.False(t, ok)

	// Add a new value to the cache in block "hash3"
	ct.Set("key1", Value{Data: []byte("value3")})

	_, ok = ct.Get("key1")
	require.True(t, ok)

	ct.Commit()
	v, ok := sc.Get("key1", "hash3")
	require.True(t, ok)
	require.Equal(t, "value3", string(v.Data))
}

func TestCacheTx_Shift(t *testing.T) {
	sc := NewStateCache()
	ct := NewBlockCache(sc, "", "hash1")

	// Add values to the cache in block "hash1"
	ct.Set("key1", Value{Data: []byte("value1_h1")})
	ct.Set("key2", Value{Data: []byte("value2_h1")})

	// Commit the changes to the main cache
	ct.Commit()

	// New block that update key1 only
	ct = NewBlockCache(sc, "hash1", "hash2")
	ct.Set("key1", Value{Data: []byte("value1_h2")})
	ct.Commit()

	// Commit should trigger shift of key2 from hash1 to hash2
	v, ok := sc.Get("key2", "hash2")
	require.True(t, ok)
	require.Equal(t, "value2_h1", string(v.Data))

	// New block to update both key1 and key2
	ct = NewBlockCache(sc, "hash2", "hash3")
	ct.Set("key1", Value{Data: []byte("value1_h3")})
	ct.Set("key2", Value{Data: []byte("value2_h3")})

	v1, ok := ct.Get("key1")
	require.True(t, ok)
	require.Equal(t, "value1_h3", string(v1.Data))

	v2, ok := ct.Get("key2")
	require.True(t, ok)
	require.Equal(t, "value2_h3", string(v2.Data))

	ct.Commit()

	v1, ok = sc.Get("key1", "hash3")
	require.True(t, ok)
	require.Equal(t, "value1_h3", string(v1.Data))

	v2, ok = sc.Get("key2", "hash3")
	require.True(t, ok)
	require.Equal(t, "value2_h3", string(v2.Data))
}

func TestConcurrentExecutionAndCommit(t *testing.T) {
	sc := NewStateCache()

	// Create two concurrent CacheTx instances for the same block
	ct1 := NewBlockCache(sc, "", "hash1")
	ct2 := NewBlockCache(sc, "", "hash1")

	// Set values in both CacheTx instances
	ct1.Set("key1", Value{Data: []byte("value1_h1")})
	ct2.Set("key1", Value{Data: []byte("value1_h1")})

	// Concurrently commit both CacheTx instances
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		ct1.Commit()
	}()

	go func() {
		defer wg.Done()
		ct2.Commit()
	}()

	wg.Wait()

	// Verify that the cache is the same after concurrent execution and commit
	v1, ok := sc.Get("key1", "hash1")
	require.True(t, ok)
	require.Equal(t, "value1_h1", string(v1.Data))
}

func TestAddRemoveAdd(t *testing.T) {
	sc := NewStateCache()

	// Create a CacheTx instance
	ct := NewBlockCache(sc, "", "hash1")

	// Add a value to the CacheTx
	ct.Set("key1", Value{Data: []byte("value1")})

	// Commit the CacheTx
	ct.Commit()

	// Verify that the value is added to the StateCache
	v1, ok := sc.Get("key1", "hash1")
	require.True(t, ok)
	require.Equal(t, "value1", string(v1.Data))

	// Create another CacheTx instance
	ct2 := NewBlockCache(sc, "hash1", "hash2")

	// Remove the value from the CacheTx
	ct2.Remove("key1")

	// Commit the CacheTx
	ct2.Commit()

	// Verify that the value is removed from the StateCache
	_, ok = sc.Get("key1", "hash2")
	require.False(t, ok)

	// Add the value again to the CacheTx
	ct2.Set("key1", Value{Data: []byte("value1")})

	// Commit the CacheTx
	ct2.Commit()

	// Verify that the value is added back to the StateCache
	v2, ok := sc.Get("key1", "hash2")
	require.True(t, ok)
	require.Equal(t, "value1", string(v2.Data))
}

func TestTransactionCache(t *testing.T) {
	sc := NewStateCache()

	for i := 0; i < 10; i++ {
		hash := fmt.Sprintf("hash%d", i)
		var preHash string
		if i > 0 {
			preHash = fmt.Sprintf("hash%d", i-1)
		}

		bc := NewBlockCache(sc, preHash, hash)
		tc := NewTransactionCache(bc)

		// Test Get method when cache is empty
		_, ok := tc.Get("key1")
		if ok && i == 0 {
			t.Error("Expected false, got ", ok)
		}

		// Test Set method
		value1 := fmt.Sprintf("value1_%s", hash)
		tc.Set("key1", Value{Data: []byte(fmt.Sprintf("value1_%s", hash))})
		value, ok := tc.Get("key1")
		require.True(t, ok)
		require.Equal(t, value1, string(value.Data))

		// Test Commit method
		value2 := fmt.Sprintf("value2_%s", hash)
		tc.Set("key2", Value{Data: []byte(value2)})
		tc.Commit()

		v1, ok := bc.Get("key1")
		require.True(t, ok)
		require.Equal(t, value1, string(v1.Data))

		v2, ok := bc.Get("key2")
		require.True(t, ok)
		require.Equal(t, value2, string(v2.Data))

		// sc should not have the values yet before commit
		_, ok = sc.Get("key1", hash)
		require.False(t, ok)

		_, ok = sc.Get("key2", hash)
		require.False(t, ok)

		// sc should see the values after commit
		bc.Commit()
		vv1, ok := sc.Get("key1", hash)
		require.True(t, ok)
		require.Equal(t, value1, string(vv1.Data))

		vv2, ok := sc.Get("key2", hash)
		require.True(t, ok)
		require.Equal(t, value2, string(vv2.Data))
	}
}
