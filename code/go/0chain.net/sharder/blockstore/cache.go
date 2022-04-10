package blockstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/viper"
)

const (
	LRU                           = "lru"
	LFU                           = "lfu"
	WriteThrough                  = "write_through"
	WriteBack                     = "write_back"
	DefaultCacheReplacementPolicy = LRU
	DefaultCacheWritePolicy       = WriteBack
	DefaultCacheReplaceTime       = time.Hour * 2
)

type cacher interface {
	Write(hash string, data []byte, t *time.Time) error
	Read(hash string) (io.ReadCloser, error)
	Replace()
	UpadateMetaData(hash string, t *time.Time)
}

type diskCache struct {
	CachePath string

	AllowedBlockNumbers uint64
	AllowedBlockSize    uint64

	CurrentBlockNumbers uint64
	CurrentBlocksSize   uint64

	ReplaceInterval time.Duration
}

func (c *diskCache) UpadateMetaData(hash string, t *time.Time) {
	Logger.Info(fmt.Sprintf("Updating cache metadata for block %v", hash))
	lockCh := getLock(hash)
	lockHash(lockCh)
	defer unlockHash(lockCh)

	ca := NewCacheAccess(hash, t)
	ca.addOrUpdate()
}

func (c *diskCache) Write(hash string, data []byte, t *time.Time) error {
	// fmt.Println("Acquiring lock for cache write")
	lockCh := getLock(hash)
	lockHash(lockCh)
	// fmt.Println("Lock acquired")

	defer unlockHash(lockCh)

	Logger.Info(fmt.Sprintf("Writing %v to cache", hash))
	if c.CurrentBlockNumbers >= c.AllowedBlockNumbers || c.CurrentBlocksSize >= c.AllowedBlockSize {
		c.Replace()
	}

	bPath := filepath.Join(c.CachePath, hash)
	f, err := os.Create(bPath)
	if err != nil {
		return err
	}

	// Try writing twice first with possible cache replacement and if error occurs replace cache and write data.
	_, err = f.Write(data)
	if err != nil {
		f.Close()
		c.Replace()

		f, err := os.Create(bPath)
		if err != nil {
			return err
		}

		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	ca := NewCacheAccess(hash, t)
	ca.addOrUpdate()

	return nil
}

func (c *diskCache) Read(hash string) (io.ReadCloser, error) {
	lockCh := getLock(hash)
	lockHash(lockCh)
	defer unlockHash(lockCh)

	bPath := filepath.Join(c.CachePath, hash)
	f, err := os.Open(bPath)
	if err != nil {
		return nil, err
	}

	return f, nil
}

var replaceLock chan struct{}

func (c *diskCache) Replace() { // only lru implemented
	select {
	case replaceLock <- struct{}{}:
	default:
		return
	}

	defer func() {
		<-replaceLock
	}()

	limitCh := make(chan struct{}, 10)
	wg := sync.WaitGroup{}
	for ca := range GetHashKeysForReplacement() {
		limitCh <- struct{}{}
		wg.Add(1)
		go func(ca *cacheAccess) {
			defer func() {
				<-limitCh
				wg.Done()
			}()

			lockCh := getLock(ca.Hash)
			select {
			case lockCh <- struct{}{}:
				defer unlockHash(lockCh)
			default:
				return
			}

			os.Remove(filepath.Join(c.CachePath, ca.Hash))
			ca.Delete(common.GetRootContext())

		}(ca)
	}
	wg.Wait()
}

func cacheInit(cViper *viper.Viper) cacher {
	cachePath := cViper.GetString("path")
	if cachePath == "" {
		panic("Cache path is required")
	}

	if err := os.RemoveAll(cachePath); err != nil {
		panic(err)
	}

	if err := os.MkdirAll(cachePath, 0644); err != nil {
		panic(err)
	}

	availableSize, totalInodes, availableInodes, err := getAvailableSizeAndInodes(cachePath)
	if err != nil {
		panic(err)
	}

	var sizeToMaintain uint64
	sizeToMaintainI := cViper.Get("size_to_maintain")
	if sizeToMaintainI != nil {
		sizeToMaintain, err = getUint64ValueFromYamlConfig(sizeToMaintainI)
		if err != nil {
			panic(err)
		}

		sizeToMaintain *= GB
	}

	if availableSize <= sizeToMaintain {
		panic(ErrSizeLimit(cachePath, sizeToMaintain).Error())
	}

	var inodesToMaintain uint64
	inodesToMaintainI := cViper.Get("inodes_to_maintain")
	if inodesToMaintainI != nil {
		inodesToMaintain, err = getUint64ValueFromYamlConfig(inodesToMaintainI)
		if err != nil {
			panic(err)
		}
	}

	if float64(100*availableInodes)/float64(totalInodes) <= float64(inodesToMaintain) {
		panic(ErrInodesLimit(cachePath, inodesToMaintain).Error())
	}

	var allowedBlockNumbers uint64
	allowedBlockNumbersI := cViper.Get("allowed_block_numbers")
	if allowedBlockNumbersI != nil {
		allowedBlockNumbers, err = getUint64ValueFromYamlConfig(allowedBlockNumbersI)
		if err != nil {
			panic(err)
		}
	}

	var allowedBlockSize uint64
	allowedBlockSizeI := cViper.Get("allowed_block_size")
	if allowedBlockSizeI != nil {
		allowedBlockSize, err = getUint64ValueFromYamlConfig(allowedBlockSizeI)
		if err != nil {
			panic(err)
		}
	}

	cacheWritePolicy := cViper.GetString("write_policy")
	if cacheWritePolicy == "" {
		cacheWritePolicy = DefaultCacheWritePolicy
	}

	switch cacheWritePolicy {
	case WriteThrough, WriteBack:
	default:
		panic(fmt.Errorf("cache write policy %v is not supported", cacheWritePolicy))
	}

	cacheReplacementPolicy := cViper.GetString("replacement_policy")
	if cacheReplacementPolicy == "" {
		cacheReplacementPolicy = DefaultCacheReplacementPolicy
	}
	var replaceDuration time.Duration
	cacheReplacementInterval := cViper.GetInt("replacement_interval")
	if cacheReplacementInterval == 0 {
		replaceDuration = time.Duration(DefaultCacheReplaceTime)
	} else {
		replaceDuration = time.Minute * time.Duration(cacheReplacementInterval)
	}

	switch cacheReplacementPolicy { // When other policies are supported then it should be registered here and respectively called.
	case LRU:
	default:
		panic(fmt.Errorf("cache replacement policy %v is not supported", cacheReplacementPolicy))
	}

	return &diskCache{
		CachePath:           cachePath,
		AllowedBlockNumbers: allowedBlockNumbers,
		AllowedBlockSize:    allowedBlockSize,
		ReplaceInterval:     replaceDuration,
	}
}

var cacheBucketLock chan struct{}

func init() {
	cacheBucketLock = make(chan struct{}, 1)
	initHashLock()
}

var mutateLock chan struct{}
var hashLock map[string]chan struct{}

func initHashLock() {
	hashLock = make(map[string]chan struct{})
	mutateLock = make(chan struct{}, 1)
	go cleanHashLock()
}

func getLock(hash string) (lock chan struct{}) {
	// fmt.Println("Get lock for hash: ", hash)
	mutateLock <- struct{}{}
	var ok bool
	lock, ok = hashLock[hash]
	if !ok {
		lock = make(chan struct{}, 1)
		hashLock[hash] = lock
	}
	<-mutateLock
	return
}

func lockHash(ch chan struct{}) {
	ch <- struct{}{}
}

func unlockHash(ch chan struct{}) {
	<-ch
}

func cleanHashLock() {
	t := time.NewTicker(time.Second * 5)
	for range t.C {
		// fmt.Printf("\nBefore cleaning map: %v\nTotal elements: %v\n", len(hashLock), hashLock)
		Logger.Info("Cleaning hash lock map")
		mutateLock <- struct{}{}

		for hash, lock := range hashLock {
			select {
			case lock <- struct{}{}:
				// <-lock
				delete(hashLock, hash)
			default:

			}
		}
		// fmt.Printf("\nAfter cleaning map: %v\n\n", hashLock)
		<-mutateLock
	}
}
