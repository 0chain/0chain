package blockstore

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
    total_blocks: "1000"
`
	viper.GetViper().SetConfigType("yaml")
	err := viper.ReadConfig(bytes.NewReader([]byte(config)))
	require.NoError(t, err)

	require.NotPanics(t, func() {
		initCache(viper.Sub("cache"))
	})

}

func TestCacheWrite(t *testing.T) {
	p := "./cache"
	defer os.RemoveAll(p)
	v := viper.New()
	v.Set("path", p)
	v.Set("total_blocks", 10)

	var c cacher
	require.NotPanics(t, func() {
		c = initCache(v)
	})

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("hash#%d", i)
		b := new(block.Block)
		b.Hash = key
		ctx, ctxCncl := context.WithTimeout(context.TODO(), CacheWriteTimeOut)
		err := c.Write(ctx, key, b)
		require.NoError(t, err)
		ctxCncl()
	}
	lruCache := c.(*cache).lru
	require.EqualValues(t, 10, lruCache.Len())
	lruKeys := lruCache.Keys()
	require.Equal(t, fmt.Sprintf("hash#%d", 19), lruKeys[len(lruKeys)-1]) // newest
	require.Equal(t, fmt.Sprintf("hash#%d", 10), lruKeys[0])              //oldest

	dirents, err := os.ReadDir(p)
	require.NoError(t, err)
	require.Len(t, dirents, 10)
	for i := 10; i < 20; i++ {
		bPath := filepath.Join(p, fmt.Sprintf("hash#%d", i))
		_, err := os.Stat(bPath)
		require.NoError(t, err)
	}
}

func TestCacheRead(t *testing.T) {
	p := "./cache"
	v := viper.New()
	v.Set("path", p)
	v.Set("total_blocks", 10)

	var c cacher
	require.NotPanics(t, func() {
		c = initCache(v)
	})

	s := []string{
		"hash1",
		"hash2",
		"hash3",
	}

	for _, hash := range s {
		b := new(block.Block)
		b.Hash = hash
		ctx, ctxCncl := context.WithTimeout(context.TODO(), CacheWriteTimeOut)
		err := c.Write(ctx, hash, b)
		ctxCncl()
		require.Nil(t, err)

		_, err = os.Stat(filepath.Join(p, hash))
		require.Nil(t, err)
	}

	lruCache := c.(*cache).lru
	keys := lruCache.Keys()
	require.Equal(t, s[0], keys[0])
	require.Equal(t, s[2], keys[len(keys)-1])
	_, err := c.Read(s[0])
	require.NoError(t, err)
	keys = lruCache.Keys()
	require.Equal(t, s[0], keys[len(keys)-1])

	_, err = c.Read(s[1])
	require.NoError(t, err)
	keys = lruCache.Keys()
	require.Equal(t, s[1], keys[len(keys)-1])
}
