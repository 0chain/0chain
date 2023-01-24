package blockstore

// Cache is a simple implementation using simple lru for storing block's minimal data.
// Sharder should provide config for cache which includes basically path, and its size.
// Each read cache will be stored uncompressed into the cache path.
// For cache is full, and size of latest read block is n then blocks are removed from blocks such
// that total removed size is >= n

import (
	"container/list"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0chain/common/core/logging"

	"0chain.net/core/viper"
)

const (
	DefaultCacheBufferSize = 100
	KB                     = 1024
	MB                     = 1024 * KB
	GB                     = 1024 * MB
)

type cacher interface {
	Write(hash string, data []byte) error
	Read(hash string) ([]byte, error)
}

// cache manages blocks cache
type cache struct {
	// path to store uncompressed blocks
	path string
	// sizeLimit limit of total blocks size in the path
	sizeLimit int64

	// size current total blocks size in the path
	size int64

	lru      *lru
	bufferCh chan *cacheEntry
}

// write will write block to the path and then run go routine to add its entry into
// lru cache. If the size reaches cache limit, it will try to delete old block caches.
func (c *cache) Write(hash string, data []byte) error {
	logging.Logger.Info(fmt.Sprintf("Writing %v to cache", hash))

	bPath := filepath.Join(c.path, hash)
	f, err := os.Create(bPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.Write(data)
	if err != nil {
		return err
	}

	c.bufferCh <- &cacheEntry{
		listEntry: listEntry{
			key:  hash,
			size: int64(n),
		},
	}

	return nil
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
		n := len(data)
		c.bufferCh <- &cacheEntry{
			listEntry: listEntry{
				key:  hash,
				size: int64(n),
			},
			data: data,
		}
	}()

	return
}

func (c *cache) Add() {
	for cEntry := range c.bufferCh {
		if c.size >= c.sizeLimit {
			c.replace(cEntry.size)
		}

		switch {
		case cEntry.data == nil: // write
			c.size += cEntry.size
			c.lru.Add(cEntry.key, cEntry.size)
		default: // read
			isNew := c.lru.Add(cEntry.key, cEntry.size)
			if isNew {
				bPath := filepath.Join(c.path, cEntry.key)
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

				c.size += cEntry.size
			}
		}

	}
}

// replace will take size as an argument.
// It will remove files that will atleast make up to the size.
// It will also remove the entry from the list
func (c *cache) replace(size int64) {
	var sum int64
	for {
		e := c.lru.list.Back()
		if e == nil {
			break
		}

		hash := e.Value.(*listEntry).key
		delSize := e.Value.(*listEntry).size
		bPath := filepath.Join(c.path, hash)

		os.Remove(bPath)
		c.lru.Remove(hash)

		sum += delSize
		if sum > size {
			break
		}
	}

	c.size -= sum
}

type listEntry struct {
	key  string
	size int64
}

type cacheEntry struct {
	listEntry
	data []byte
}

// lru A combination of map and doubly linked list that provides simple mechanism
// of implementing lru replacement policy.
// For a key, space used by it will be: 2*len(key) + size_of(int).
// Basically it is 2*64+64 = 192
type lru struct {
	list  *list.List
	items map[string]*list.Element
}

// Add add/update entry to the list and map
func (l *lru) Add(key string, size int64) (isNew bool) {
	if ent, ok := l.items[key]; ok {
		l.list.MoveToFront(ent)
		return
	}
	e := &listEntry{key: key, size: size}
	listElem := l.list.PushFront(e)
	l.items[key] = listElem

	return true
}

// Remove will remove key from the items and element from list if exists
func (l *lru) Remove(key string) {
	var elem *list.Element
	var ok bool
	if elem, ok = l.items[key]; !ok {
		return
	}

	l.list.Remove(elem)
	delete(l.items, key)

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

	size, err := parseCacheSize(viper.Get("size"))
	if err != nil {
		panic(err)
	}

	if size < 500*MB {
		panic("cache size cannot be lesser than 500MB")
	}

	c := &cache{
		path:      cPath,
		sizeLimit: size,
		lru: &lru{
			list:  list.New(),
			items: make(map[string]*list.Element),
		},
		bufferCh: make(chan *cacheEntry, DefaultCacheBufferSize),
	}

	go c.Add()

	return c
}

func parseCacheSize(sizeI interface{}) (int64, error) {
	switch sizeI := sizeI.(type) {
	case int:
		return int64(sizeI), nil
	case float64:
		return int64(sizeI), nil
	case string:
		s := sizeI
		s = strings.ToLower(s)
		multiplier := 1
		var sep string
		if strings.Contains(s, "kb") {
			sep = "kb"
			multiplier = KB
		} else if strings.Contains(s, "mb") {
			sep = "mb"
			multiplier = MB
		} else if strings.Contains(s, "gb") {
			sep = "gb"
			multiplier = GB
		}

		sizeStr := s
		if sep != "" {
			sizeStr = strings.Split(s, sep)[0]
		}

		if strings.Contains(sizeStr, ".") {
			size, err := strconv.ParseFloat(sizeStr, 64)
			if err != nil {
				return 0, err
			}
			size *= float64(multiplier)
			return int64(size), nil
		}

		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return 0, err
		}

		size *= int64(multiplier)
		return size, nil
	}

	return 0, fmt.Errorf("invalid size value: %v", sizeI)
}
