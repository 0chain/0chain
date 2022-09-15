package blockstore

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"

	"0chain.net/core/viper"
)

/*BlockStore - an interface to read and write blocks to some storage */
type BlockStoreI interface {
	Write(b *block.Block) error
	Read(hash string, round int64) (*block.Block, error)
	ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error)
	Delete(hash string) error
}

var Store BlockStoreI

/*GetStore - get the block store that's is setup */
func GetStore() BlockStoreI {
	return Store
}

/*SetupStore - Setup a file system based block storage */
func SetupStore(store BlockStoreI) {
	Store = store
}

type Tiering uint8

const (
	// Cache = 1, Disk = 2, Cold = 4
	DiskOnly         Tiering = 2
	CacheAndDisk     Tiering = 3
	DiskAndCold      Tiering = 6
	CacheDiskAndCold Tiering = 7
)

const (
	DefaultBlockMovementInterval = 720 * time.Hour
)

type blockStore struct {
	cache    cacher
	diskTier *diskTier
	coldTier *coldTier
	// fields with registered functions as per the config files
	write func(b *block.Block) error
	read  func(hash string, round int64) (b *block.Block, err error)

	// blockMovementInterval interval to check for blocks to move to cold
	// storage. This value also determines if a block is cold enough to move
	// so it is better to choose duration of month
	blockMovementInterval time.Duration
}

func (sm *blockStore) Write(b *block.Block) error {
	if b == nil {
		return errors.New("cannot write nil block")
	}

	logging.Logger.Info("Writing block: " + b.Hash)
	err := sm.write(b)
	if err != nil {
		logging.Logger.Error(err.Error())
		panic(err)
	}

	return nil
}

func (sm *blockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	logging.Logger.Info(fmt.Sprintf("Reading block summary for block: %v", bs.Hash))
	return sm.read(bs.Hash, bs.Round)
}

func (sm *blockStore) Read(hash string, round int64) (b *block.Block, err error) {
	logging.Logger.Info("Reading block: " + hash)
	return sm.read(hash, round)
}

func (sm *blockStore) Delete(hash string) error {
	return nil // Not implemented
}

func Init(ctx context.Context, sViper *viper.Viper, workDir string) {
	logging.Logger.Info("Initializing storages")
	storageType := sViper.GetInt("storage_type")
	if storageType == 0 {
		panic(errors.New("Storage Type is a required field"))
	}

	/*
		Mode can be one of "start" and "restart".
		"start" mode will clean the paths before starting.
		"restart" mode will not clean, but start from where it left.
		"restart" mode might be required when sharder needs to modify config
		or sharder crashed in the middle.
	*/
	mode := sViper.GetString("mode")
	if mode == "" {
		mode = "start"
	}

	bwrCacheSize, err := getUint64ValueFromYamlConfig(sViper.Get("rocks.cache_size"))
	if err != nil {
		panic(err)
	}
	initBlockWhereRecord(bwrCacheSize, mode, workDir)

	store := new(blockStore)
	switch Tiering(storageType) {
	default:
		panic(fmt.Sprint("Unknown storage type: ", storageType))
	case CacheAndDisk:
		//
		store.cache = initCache(sViper.Sub("cache"))
		store.diskTier = initDisk(sViper.Sub("disk"), mode)
		store.write = func(b *block.Block) (err error) {
			data, err := getBlockData(b)
			if err != nil {
				return err
			}

			blockPath, err := store.diskTier.write(b, data)
			if err != nil {
				return err
			}

			bwr := blockWhereRecord{
				Hash:      b.Hash,
				Tiering:   DiskTier,
				BlockPath: blockPath,
			}
			err = bwr.save()
			if err != nil {
				os.Remove(blockPath)
				return err
			}

			go func() {
				if err := store.cache.Write(b.Hash, data); err != nil {
					logging.Logger.Error(err.Error())
				}
			}()

			return nil
		}

		store.read = func(hash string, round int64) (b *block.Block, err error) {
			b, err = store.readFromCache(hash)
			if err == nil && b != nil {
				return
			}
			bwr, err := getBWR(hash)
			if err != nil {
				return nil, err
			}
			b, err = store.diskTier.read(bwr.BlockPath)
			if err == nil && b != nil {
				go store.addToCache(b)
			}
			return
		}

	case DiskOnly:
		store.diskTier = initDisk(sViper.Sub("disk"), mode)
		store.write = func(b *block.Block) (err error) {
			data, err := getBlockData(b)
			if err != nil {
				return err
			}

			blockPath, err := store.diskTier.write(b, data)
			if err != nil {
				return err
			}

			bwr := blockWhereRecord{
				Hash:      b.Hash,
				Tiering:   DiskTier,
				BlockPath: blockPath,
			}
			err = bwr.save()
			if err != nil {
				os.Remove(blockPath)
				return err
			}

			return nil
		}

		store.read = func(hash string, round int64) (b *block.Block, err error) {
			bwr, err := getBWR(hash)
			if err != nil {
				return nil, err
			}
			return store.diskTier.read(bwr.BlockPath)
		}

	case CacheDiskAndCold:
		store.cache = initCache(sViper.Sub("cache"))
		store.diskTier = initDisk(sViper.Sub("disk"), mode)
		store.coldTier = initCold(sViper.Sub("cold"), mode)

		store.write = func(b *block.Block) error {
			data, err := getBlockData(b)
			if err != nil {
				return err
			}

			blockPath, err := store.diskTier.write(b, data)
			if err != nil {
				return err
			}

			bwr := blockWhereRecord{
				Hash:      b.Hash,
				Tiering:   DiskTier,
				BlockPath: blockPath,
			}
			err = bwr.save()
			if err != nil {
				os.Remove(blockPath)
				return err
			}

			go func() {
				if err := store.cache.Write(b.Hash, data); err != nil {
					logging.Logger.Error(err.Error())
				}
			}()

			go store.addToUBR(b)

			return nil
		}

		store.read = func(hash string, round int64) (b *block.Block, err error) {
			b, err = store.readFromCache(hash)
			if err == nil {
				return
			}

			bwr, err := getBWR(hash)
			if err != nil {
				return nil, err
			}

			b, err = store.diskTier.read(bwr.BlockPath)
			if err == nil && b != nil {
				go store.addToCache(b)
				return
			}

			b, err = store.readFromColdTier(hash, bwr.ColdPath)
			if err == nil && b != nil {
				go store.addToCache(b)
			}

			return
		}

		blockMovementInterval := sViper.GetDuration("block_movement_interval")
		if blockMovementInterval == 0 {
			blockMovementInterval = DefaultBlockMovementInterval
		}
		store.blockMovementInterval = blockMovementInterval
	case DiskAndCold:
		store.diskTier = initDisk(sViper.Sub("disk"), mode)
		store.coldTier = initCold(sViper.Sub("cold"), mode)
		store.write = func(b *block.Block) error {
			data, err := getBlockData(b)
			if err != nil {
				return err
			}

			blockPath, err := store.diskTier.write(b, data)
			if err != nil {
				return err
			}

			bwr := blockWhereRecord{
				Hash:      b.Hash,
				Tiering:   DiskTier,
				BlockPath: blockPath,
			}
			err = bwr.save()
			if err != nil {
				os.Remove(blockPath)
				return err
			}

			go store.addToUBR(b)

			return nil
		}

		store.read = func(hash string, round int64) (b *block.Block, err error) {
			bwr, err := getBWR(hash)
			if err != nil {
				return nil, err
			}
			switch bwr.Tiering {
			case DiskTier:
				return store.diskTier.read(bwr.BlockPath)
			case ColdTier:

			case DiskAndColdTier:
				b, err = store.diskTier.read(bwr.BlockPath)
				if err == nil && b != nil {
					return b, nil
				}

				b, err = store.readFromColdTier(hash, bwr.ColdPath)
				return
			}

			return
		}

		blockMovementInterval := sViper.GetDuration("block_movement_interval")
		if blockMovementInterval == 0 {
			blockMovementInterval = DefaultBlockMovementInterval
		}
		store.blockMovementInterval = blockMovementInterval
		//
	}

	SetupStore(store)

	switch Tiering(storageType) {
	case DiskAndCold, CacheDiskAndCold:
		logging.Logger.Info("Setting up cold storage worker")
		go setupColdWorker(ctx)
	}

	go setupVolumeRevivingWorker(ctx)
}

func getBlockData(b *block.Block) ([]byte, error) {
	return json.Marshal(b)
}

func (store *blockStore) readFromColdTier(hash, coldPath string) (b *block.Block, err error) {
	data, err := store.coldTier.read(coldPath, hash)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(data)
	zR, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	err = datastore.ReadJSON(zR, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (store *blockStore) readFromCache(hash string) (b *block.Block, err error) {
	data, err := store.cache.Read(hash)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(data)
	err = datastore.ReadJSON(r, b)
	if err != nil {
		return nil, err
	}
	return
}

func (store *blockStore) addToCache(b *block.Block) {
	data, _ := getBlockData(b)
	err := store.cache.Write(b.Hash, data)
	if err != nil {
		logging.Logger.Error(err.Error())
	}
}

func (store *blockStore) addToUBR(b *block.Block) {
	ubr := &unmovedBlockRecord{
		Hash:      b.Hash,
		CreatedAt: b.CreationDate,
	}

	if err := ubr.Add(); err != nil {
		logging.Logger.Error("Error while adding %s to ubr. " + err.Error())
	}
}
