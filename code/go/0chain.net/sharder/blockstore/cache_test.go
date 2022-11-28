package blockstore

import (
	"bytes"
	"container/list"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0chain/common/core/logging"

	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("debug", ".")
}

func TestInitCache(t *testing.T) {
	config := `
cache:
	path: "/path/to/cache"
	size: 1024*1024*1024
`
	err := viper.ReadConfig(bytes.NewReader([]byte(config)))
	require.NoError(t, err)

	require.NotPanics(t, func() {
		initCache(viper.Sub("cache"))
	})

}

func TestLRUAdd(t *testing.T) {
	l := lru{
		list:  list.New(),
		items: make(map[string]*list.Element),
	}

	m := map[string]int{
		"k1": 1,
		"k2": 2,
		"k3": 3,
		"k4": 4,
		"k5": 5,
	}
	for k, v := range m {
		l.Add(k, v)
		e := l.list.Front()
		key := e.Value.(*listEntry).key
		require.Equal(t, k, key)
	}
}

func TestLRURemove(t *testing.T) {
	l := lru{
		list:  list.New(),
		items: make(map[string]*list.Element),
	}

	m := map[string]int{
		"k1": 1,
		"k2": 2,
		"k3": 3,
		"k4": 4,
		"k5": 5,
	}
	for k, v := range m {
		l.Add(k, v)
	}

	for k := range m {
		l.Remove(k)
		_, ok := l.items[k]
		require.False(t, ok)
	}
}

func TestCacheWrite(t *testing.T) {
	p := "./cache"
	v := viper.New()
	v.Set("path", p)
	v.Set("size", "1024*2")

	var c cacher
	require.NotPanics(t, func() {
		c = initCache(v)
	})

	size := 1024
	hash1 := "hash1"
	b := generateRandomBytes(t, size)
	err := c.Write(hash1, b)
	require.Nil(t, err)

	finfo, err := os.Stat(filepath.Join(p, hash1))
	require.Nil(t, err)

	require.EqualValues(t, size, finfo.Size())

	size = 1025
	hash2 := "hash2"
	b = generateRandomBytes(t, size)
	err = c.Write(hash2, b)
	require.Nil(t, err)

	size = 1025
	hash3 := "hash3"
	b = generateRandomBytes(t, size)
	err = c.Write(hash3, b)
	require.Nil(t, err)

	time.Sleep(time.Second)
	finfo, err = os.Stat(filepath.Join(p, hash1))
	require.NotNil(t, err)

	finfo, err = os.Stat(filepath.Join(p, hash2))
	require.NotNil(t, err)

	finfo, err = os.Stat(filepath.Join(p, hash3))
	require.Nil(t, err)
	require.EqualValues(t, size, finfo.Size())

	l := c.(*cache).lru.list
	e := l.Front()
	k := e.Value.(*listEntry).key
	require.Equal(t, k, hash3)
}

func TestCacheRead(t *testing.T) {
	p := "./cache"
	v := viper.New()
	v.Set("path", p)
	v.Set("size", "1024*3")

	var c cacher
	require.NotPanics(t, func() {
		c = initCache(v)
	})

	m := map[string]int{
		"hash1": 1024,
		"hash2": 1024,
		"hash3": 1024,
	}

	var lastKey string
	for k, v := range m {
		b := generateRandomBytes(t, v)
		err := c.Write(k, b)
		require.Nil(t, err)

		finfo, err := os.Stat(filepath.Join(p, k))
		require.Nil(t, err)

		require.EqualValues(t, v, finfo.Size())
		lastKey = k
	}

	time.Sleep(500 * time.Millisecond)
	e := c.(*cache).lru.list.Front()
	k := e.Value.(*listEntry).key
	require.Equal(t, k, lastKey)

	hash1 := "hash1"
	b, err := c.Read(hash1)
	require.Nil(t, err)

	require.Equal(t, m[hash1], len(b))

	time.Sleep(500 * time.Millisecond)
	e = c.(*cache).lru.list.Front()
	k = e.Value.(*listEntry).key
	require.Equal(t, k, hash1)
}

func generateRandomBytes(t *testing.T, size int) []byte {
	b := make([]byte, size)
	n, err := rand.Read(b)
	require.Nil(t, err)
	require.Equal(t, n, size)

	return b

}
