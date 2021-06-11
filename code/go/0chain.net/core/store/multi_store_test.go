package store

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func clearTestFSMultiStore() {
	err := os.RemoveAll(path.Join("tmp", "multi"))
	if err != nil {
		panic(err)
	}
}

func createTestFSMultiStore(strategy MultiStorePutStrategy, count int) (Store, []Store) {
	var stores []Store
	count = int(math.Min(float64(count), 10))
	for i := 0; i < count; i++ {
		store, err := NewFSStore(path.Join("tmp", "multi", strconv.Itoa(i)))
		if err != nil {
			panic(err)
		}
		stores = append(stores, store)
	}
	return NewMultiStore(stores, strategy), stores
}

func TestMultiStoreGet(t *testing.T) {
	clearTestFSMultiStore()

	store, _ := createTestFSMultiStore(MinSizeFirst, 10)

	n := 1000
	maxBlockSize := 64 * 1024

	v := make([]byte, maxBlockSize)

	samples := make(map[int][]byte)

	for _, i := range rand.Perm(n / 7) {
		samples[7*i] = nil
	}

	for i := 0; i < n; i++ {
		k := []byte(strconv.Itoa(i))
		v = v[:rand.Intn(maxBlockSize)]
		if _, err := rand.Read(v); err != nil {
			panic(err)
		}
		if _, ok := samples[i]; ok {
			samples[i] = make([]byte, len(v))
			copy(samples[i], v)
		}
		require.NoError(t, store.Put(k, v))
	}

	for i, v := range samples {
		value, err := store.Get([]byte(strconv.Itoa(i)))
		require.NoError(t, err, fmt.Sprintf("key=%v", i))
		require.EqualValues(t, v, value)
	}

}

func TestMultiStoreDelete(t *testing.T) {
	clearTestFSMultiStore()

	store, _ := createTestFSMultiStore(MinCountFirst, 10)

	n := 100
	maxBlockSize := 64 * 1024

	v := make([]byte, maxBlockSize)

	for i := 0; i < n; i++ {
		k := []byte(strconv.Itoa(i))
		v = v[:rand.Intn(maxBlockSize)]
		if _, err := rand.Read(v); err != nil {
			panic(err)
		}
		require.NoError(t, store.Put(k, v))
	}

	k := []byte(strconv.Itoa(1))
	require.NoError(t, store.Delete(k))
	require.EqualError(t, store.Delete(k), ErrKeyNotFound.Error())
}

func TestMultiStorePutStrategyMinSizeFirst(t *testing.T) {
	clearTestFSMultiStore()

	store, stores := createTestFSMultiStore(MinSizeFirst, 10)

	n := 1000
	maxBlockSize := 64 * 1024

	v := make([]byte, maxBlockSize)

	for i := 0; i < n; i++ {
		k := []byte(strconv.Itoa(i))
		v = v[:rand.Intn(maxBlockSize)]
		if _, err := rand.Read(v); err != nil {
			panic(err)
		}
		require.NoError(t, store.Put(k, v))
	}

	var totalSize, totalCount int64
	maxSize, minSize := stores[0].Size(), stores[0].Size()
	for _, store := range stores {
		size, count := store.Size(), store.Count()
		totalSize += size
		totalCount += count
		if size > maxSize {
			maxSize = size
		}
		if size < minSize {
			minSize = size
		}
	}

	require.Equal(t, store.Size(), totalSize)
	require.Equal(t, store.Count(), totalCount)
	require.Condition(t, func() (success bool) { return maxSize-minSize <= int64(maxBlockSize) })
}

func TestMultiStorePutStrategyMinCountFirst(t *testing.T) {
	clearTestFSMultiStore()

	store, stores := createTestFSMultiStore(MinCountFirst, 10)

	n := 100
	maxBlockSize := 64 * 1024

	v := make([]byte, maxBlockSize)

	for i := 0; i < n; i++ {
		k := []byte(strconv.Itoa(i))
		v = v[:rand.Intn(maxBlockSize)]
		if _, err := rand.Read(v); err != nil {
			panic(err)
		}
		require.NoError(t, store.Put(k, v))
	}

	var counts []int64
	var sumSize, sumCount int64
	for _, store := range stores {
		size, count := store.Size(), store.Count()
		sumSize += size
		sumCount += count
		counts = append(counts, count)

	}
	require.Equal(t, store.Size(), sumSize)
	require.Equal(t, store.Count(), sumCount)
	require.ElementsMatch(t, counts, []int64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10})
}

func TestMultiStorePutStrategyRoundRobin(t *testing.T) {
	clearTestFSMultiStore()

	store, stores := createTestFSMultiStore(RoundRobin, 10)

	n := 100
	maxBlockSize := 64 * 1024

	v := make([]byte, maxBlockSize)

	for i := 0; i < n; i++ {
		k := []byte(strconv.Itoa(i))
		v = v[:rand.Intn(maxBlockSize)]
		if _, err := rand.Read(v); err != nil {
			panic(err)
		}
		require.NoError(t, store.Put(k, v))
	}

	var counts []int64
	var sumSize, sumCount int64
	for _, store := range stores {
		size, count := store.Size(), store.Count()
		sumSize += size
		sumCount += count
		counts = append(counts, count)

	}
	require.Equal(t, store.Size(), sumSize)
	require.Equal(t, store.Count(), sumCount)
	require.ElementsMatch(t, counts, []int64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10})
}
