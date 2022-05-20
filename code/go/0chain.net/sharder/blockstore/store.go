package blockstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/logging"
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
	write  func(b *block.Block) (string, error)
	read   func(hash string, round int64) (b *block.Block, err error)
	delete func(hash string) error

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
	blockPath, err := sm.write(b)
	if err != nil {
		logging.Logger.Error(err.Error())
		panic(err)
	}

	logging.Logger.Info(fmt.Sprintf("Block %v written to %v successfully", b.Hash, blockPath))

	return nil
}

func (sm *blockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	logging.Logger.Info(fmt.Sprintf("Reading block summary for block: %v", bs.Hash))
	return sm.read(bs.Hash, bs.Round)
}

func (sm *blockStore) Read(hash string, round int64) (b *block.Block, err error) {
	logging.Logger.Info("Reading block: " + b.Hash)
	return sm.read(hash, round)
}

func (sm *blockStore) Delete(hash string) error {
	return nil // Not implemented
}

func InitializeStore(sViper *viper.Viper, ctx context.Context) {
	logging.Logger.Info("Initializing storages")
	storageType := sViper.GetInt("storage_type")
	if storageType == 0 {
		panic(errors.New("Storage Type is a required field"))
	}

	mode := sViper.GetString("mode")
	if mode == "" {
		mode = "start"
	}
	/*
		setup bwr
	*/

	store := new(blockStore)

	switch Tiering(storageType) {
	default:
		panic(fmt.Sprint("Unknown storage type: ", storageType))
	case CacheAndDisk:
		//
		fallthrough
	case DiskOnly:
		//
	case CacheDiskAndCold:
		//
		fallthrough
	case DiskAndCold:

		blockMovementInterval := sViper.GetDuration("block_movement_interval")
		if blockMovementInterval == 0 {
			blockMovementInterval = DefaultBlockMovementInterval
		}
		store.blockMovementInterval = blockMovementInterval
		//
	}

	/*
		setup workers for block movement and cache replacement
	*/
}

func getBlockData(b *block.Block) ([]byte, error) {
	return json.Marshal(b)
}
