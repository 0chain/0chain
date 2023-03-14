package blockstore

import (
	"bytes"
	"container/list"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0chain/common/core/logging"

	"0chain.net/chaincore/block"
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
    size: "1GB"
`
	viper.GetViper().SetConfigType("yaml")
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

	m := map[string]int64{
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

	m := map[string]int64{
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
	v.Set("size", 500*MB)

	var c cacher
	require.NotPanics(t, func() {
		c = initCache(v)
	})

	hash1 := "hash1"
	b := new(block.Block)
	b.Hash = hash1
	err := c.Write(hash1, b)
	require.Nil(t, err)

	_, err = os.Stat(filepath.Join(p, hash1))
	require.Nil(t, err)

	hash2 := "hash2"
	b.Hash = hash2
	err = c.Write(hash2, b)
	require.Nil(t, err)

	hash3 := "hash3"
	b.Hash = hash3
	err = c.Write(hash3, b)
	require.Nil(t, err)

	time.Sleep(time.Second)
	_, err = os.Stat(filepath.Join(p, hash1))
	require.NotNil(t, err)

	_, err = os.Stat(filepath.Join(p, hash2))
	require.NotNil(t, err)

	_, err = os.Stat(filepath.Join(p, hash3))
	require.Nil(t, err)

	l := c.(*cache).lru.list
	e := l.Front()
	k := e.Value.(*listEntry).key
	require.Equal(t, k, hash3)
}

func TestCacheRead(t *testing.T) {
	p := "./cache"
	v := viper.New()
	v.Set("path", p)
	v.Set("size", 500*MB)

	var c cacher
	require.NotPanics(t, func() {
		c = initCache(v)
	})

	s := []string{
		"hash1",
		"hash2",
		"hash3",
	}

	var lastKey string
	for _, hash := range s {
		b := new(block.Block)
		b.Hash = hash
		err := c.Write(hash, b)
		require.Nil(t, err)

		_, err = os.Stat(filepath.Join(p, hash))
		require.Nil(t, err)

		lastKey = hash
	}

	time.Sleep(500 * time.Millisecond)
	e := c.(*cache).lru.list.Front()
	k := e.Value.(*listEntry).key
	require.Equal(t, k, lastKey)

	hash1 := "hash1"
	_, err := c.Read(hash1)
	require.Nil(t, err)

	time.Sleep(500 * time.Millisecond)
	e = c.(*cache).lru.list.Front()
	k = e.Value.(*listEntry).key
	require.Equal(t, k, hash1)
}
