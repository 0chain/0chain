package blockstore

// Cache is a simple implementation using simple lru for storing block's minimal data.
// Sharder should provide config for cache which includes basically path, and its size.
// Each read cache will be stored uncompressed into the cache path.
// For cache is full, and size of latest read block is n then blocks are removed from blocks such
// that total removed size is >= n

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/0chain/common/core/logging"
	simpleLru "github.com/hashicorp/golang-lru/v2"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/viper"
)

const (
	DefaultCacheBufferSize = 100
	KB                     = 1024
	MB                     = 1024 * KB
	GB                     = 1024 * MB
)

type cacher interface {
	Write(ctx context.Context, hash string, b *block.Block) error
	Read(hash string) ([]byte, error)
}

type noOpCache struct{}

func (noOpCache) Write(ctx context.Context, hash string, b *block.Block) error {
	return nil
}

func (noOpCache) Read(hash string) ([]byte, error) {
	return nil, nil
}

// cache manages blocks cache
type cache struct {
	// path to store uncompressed blocks
	path string

	lru      *simpleLru.Cache[string, interface{}]
	bufferCh chan *cacheEntry
}

// write will write block to the path and then run go routine to add its entry into
// lru cache. If the size reaches cache limit, it will try to delete old block caches.
func (c *cache) Write(ctx context.Context, hash string, b *block.Block) error {
	logging.Logger.Info(fmt.Sprintf("Writing %v to cache", hash))

	bPath := filepath.Join(c.path, hash)
	f, err := os.Create(bPath)
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := new(bytes.Buffer)
	err = datastore.WriteMsgpack(buffer, b)
	if err != nil {
		return err
	}

	data := buffer.Bytes()
	_, err = f.Write(data)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return errors.New("context timeout")
	case c.bufferCh <- &cacheEntry{key: hash}:
		return nil
	}
}

// read read from cache and update the metadata cache.
// If the read is done but actual file was deleted by other process then
// `isNew` value is checked and data is re-written.
func (c *cache) Read(hash string) (data []byte, err error) {
	bPath := filepath.Join(c.path, hash)
	f, err := os.Open(bPath)
	if err != nil {
		return
	}
	defer f.Close()

	data, err = io.ReadAll(f)
	if err != nil {
		return
	}

	go func() {
		c.bufferCh <- &cacheEntry{
			key:  hash,
			data: data,
		}
	}()

	return
}

func (c *cache) Add() {
	for cEntry := range c.bufferCh {
		if cEntry.data == nil || c.lru.Contains(cEntry.key) { // write
			c.lru.Add(cEntry.key, nil)
			continue
		}

		oldestKey, _, _ := c.lru.GetOldest()
		evicted := c.lru.Add(cEntry.key, nil)
		if evicted && oldestKey != "" {
			bPath := filepath.Join(c.path, oldestKey)
			os.Remove(bPath)

			bPath = filepath.Join(c.path, cEntry.key)
			f, err := os.Create(bPath)
			if err != nil {
				c.lru.Remove(cEntry.key)
				continue
			}
			_, err = f.Write(cEntry.data)
			if err != nil {
				c.lru.Remove(cEntry.key)
				f.Close()
				continue
			}
			f.Close()
		}

	}
}

type cacheEntry struct {
	key  string
	data []byte
}

/**********************************Initialization************************************/

func initCache(viper *viper.Viper) cacher {
	cPath := viper.GetString("path")
	err := os.RemoveAll(cPath)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(cPath, 0700)
	if err != nil {
		panic(err)
	}

	totalBlocks := viper.GetInt("total_blocks")
	if totalBlocks == 0 {
		panic("cannot initialize cache to store zero blocks")
	}

	lru, _ := simpleLru.New[string, interface{}](totalBlocks)
	c := &cache{
		path:     cPath,
		lru:      lru,
		bufferCh: make(chan *cacheEntry, DefaultCacheBufferSize),
	}

	go c.Add()

	return c
}
