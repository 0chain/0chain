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
	"math"
	"os"
	"path/filepath"
	"sync"

	"0chain.net/core/logging"
	"0chain.net/core/viper"
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

	sizeMu *sync.Mutex
	// size current total blocks size in the path
	size int

	lru *lru
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

	c.sizeMu.Lock()
	c.size += n
	c.sizeMu.Unlock()

	if c.size >= c.sizeLimit {
		go c.replace(n)
	}

	go c.lru.Add(hash, n)
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
		isNew := c.lru.Add(hash, n)
		if isNew {
			if err := c.Write(hash, data); err != nil {
				c.lru.Remove(hash)
				return
			}
			c.sizeMu.Lock()
			c.size += n
			c.sizeMu.Unlock()
		}
	}()

	return
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

		hash := e.Value.(*entry).key
		delSize := e.Value.(*entry).size // delete size
		bPath := filepath.Join(c.path, hash)

		go func(bPath string) {
			os.Remove(bPath)
			c.lru.list.Remove(e)
		}(bPath)

		sum += delSize
		if sum > size {
			break
		}
	}

	c.sizeMu.Lock()
	c.size -= sum
	c.sizeMu.Unlock()
}

/*
// Comment now as its too much of lock contention
func (c *cache) replaceHalfCaches() {
	limitCh := make(chan struct{}, 10)
	var sum int

	for ent := range c.lru.getKeysAndCleanList(50) {
		bPath := filepath.Join(c.path, ent.key)
		sum += ent.size

		limitCh <- struct{}{}
		go func(bPath string) {
			os.Remove(bPath)
			<-limitCh
		}(bPath)
	}

	c.sizeMu.Lock()
	c.size -= sum
	c.sizeMu.Unlock()
}
*/
type entry struct {
	key  string
	size int
}

// lru A combination of map and doubly linked list that provides simple mechanism
// of implementing lru replacement policy.
// For a key, space use by it will be: 2*len(key) + size_of(int).
// Basically it is 2*64+64 = 192
type lru struct {
	lock  *sync.Mutex
	list  *list.List
	items map[string]*list.Element
}

// Add add/update entry to the list and map
func (l *lru) Add(key string, size int) (isNew bool) {
	l.lock.Lock()
	if ent, ok := l.items[key]; ok {
		l.list.MoveToFront(ent)
		l.lock.Unlock()
		return
	}
	e := &list.Element{Value: &entry{key: key, size: size}}
	l.items[key] = e
	l.list.PushFront(e)

	l.lock.Unlock()
	return true
}

// Remove will remove key from the items and element from list if exists
func (l *lru) Remove(key string) {
	l.lock.Lock()

	var elem *list.Element
	var ok bool
	if elem, ok = l.items[key]; !ok {
		return
	}

	l.list.Remove(elem)
	delete(l.items, key)

	l.lock.Unlock()
}

// getKeysAndCleanList will take percent as argument.
// so if percent is 50 then 50% of the elements in the list will be deleted
// and those deleted items will be sent to the channel for further processing.
func (l *lru) getKeysAndCleanList(percent int) <-chan *entry {
	ch := make(chan *entry)
	switch {
	case percent > 100:
		percent = 100
	case percent < 1:
		close(ch)
		return ch
	}

	listLength := int(math.Floor(float64(percent/100) * float64(l.list.Len())))
	go func() {
		l.lock.Lock()
		for i := 0; i < listLength; i++ {
			e := l.list.Back()
			if e == nil {
				break
			}

			ent := e.Value.(*entry)
			l.list.Remove(e)
			delete(l.items, ent.key)
			ch <- ent
		}

		close(ch)
		l.lock.Unlock()
	}()

	return ch
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

	size, err := getintValueFromYamlConfig(viper.GetString("size"))
	if err != nil {
		panic(err)
	}

	return &cache{
		path:      cPath,
		sizeLimit: size,
	}
}
