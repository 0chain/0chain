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

	"github.com/0chain/common/core/logging"

	"0chain.net/core/viper"
)

const (
	DefaultCacheBufferSize = 100
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
	sizeLimit int

	// size current total blocks size in the path
	size int

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
			size: n,
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
				size: n,
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
func (c *cache) replace(size int) {
	var sum int
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
	size int
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
func (l *lru) Add(key string, size int) (isNew bool) {
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
	if viper == nil {
		panic(ErrCacheStorageConfNotProvided)
	}

	cPath := viper.GetString("path")
	err := os.RemoveAll(cPath)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(cPath, 0777)
	if err != nil {
		panic(err)
	}

	size, err := getintValueFromYamlConfig(viper.Get("size"))
	if err != nil {
		panic(err)
	}

	if size < 500*MB {
		panic("cache size cannot be lesser than 500MB")
	}

	var bufferSize int
	bufferSizeStr := viper.GetString("buffer_size")
	if bufferSizeStr == "" {
		bufferSize = DefaultCacheBufferSize
	} else {
		bufferSize, err = getintValueFromYamlConfig(viper.GetString("buffer_size"))
		if err != nil {
			panic(err)
		}
	}

	c := &cache{
		path:      cPath,
		sizeLimit: size,
		lru: &lru{
			list:  list.New(),
			items: make(map[string]*list.Element),
		},
		bufferCh: make(chan *cacheEntry, bufferSize),
	}

	go c.Add()

	return c
}
