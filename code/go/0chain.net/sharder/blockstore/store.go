package blockstore

import (
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/viper"
)

type Tiering uint8

const (
	// Cache = 1, Warm = 2, Hot = 4 and Cold = 8
	WarmOnly         Tiering = 2
	HotOnly          Tiering = 4
	CacheAndWarm     Tiering = 3
	CacheAndCold     Tiering = 9
	HotAndCold       Tiering = 12
	WarmAndCold      Tiering = 10
	CacheHotAndCold  Tiering = 13
	CacheWarmAndCold Tiering = 11
)

const (
	HOT   = "hot"
	WARM  = "warm"
	CACHE = "cache"
	COLD  = "cold"
)

var Store BlockStore
var workerMap map[string]interface{}

type BlockStore struct {
	Tiering  Tiering
	Cache    cacher
	HotTier  *diskTier
	WarmTier *diskTier
	ColdTier *coldTier
	// fields with registered functions as per the config files
	write  func(b *block.Block) (string, error)
	read   func(hash string, round int64) (b *block.Block, err error)
	delete func(hash string) error
}

func (sm *BlockStore) Write(b *block.Block) error {
	if b == nil {
		return errors.New("cannot write nil block")
	}

	Logger.Info("Writing block: " + b.Hash)
	blockPath, err := sm.write(b)
	if err != nil {
		Logger.Error(err.Error())
		panic(err)
	}

	Logger.Info(fmt.Sprintf("Block %v written to %v successfully", b.Hash, blockPath))

	return nil
}

func (sm *BlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	Logger.Info(fmt.Sprintf("Reading block summary for block: %v", bs.Hash))
	return sm.read(bs.Hash, bs.Round)
}

func (sm *BlockStore) Read(hash string, round int64) (b *block.Block, err error) {
	Logger.Info("Reading block: " + b.Hash)
	return sm.read(hash, round)
}

func (sm *BlockStore) Delete(hash string) error {
	return nil // Not implemented
}

func InitializeStore(sViper *viper.Viper, ctx context.Context) error {
	Logger.Info("Initializing storages")
	var mode string
	var storageType int

	storageType = sViper.GetInt("storage_type")

	if storageType == 0 {
		panic(errors.New("Storage Type is a required field"))
	}

	mode = sViper.GetString("mode")
	if mode == "" {
		mode = "start"
	}

	/*var bmrPath, qmrPath = DefaultBlockMetaRecordDB, DefaultQueryMetaRecordDB
	boltConfigMap := sViper.GetStringMapString("bolt")
	if boltConfigMap == nil {
		bmrPath = DefaultBlockMetaRecordDB
		qmrPath = DefaultQueryMetaRecordDB
	} else {

		if boltConfigMap["block_meta_record_path"] == "" {
			bmrPath = DefaultBlockMetaRecordDB
		}

		if boltConfigMap["query_meta_record_path"] == "" {
			qmrPath = DefaultQueryMetaRecordDB
		}
	}*/

	switch mode {
	case "start", "recover":
		InitMetaRecordDB("localhost", "6379", true) // Removes existing metadata and creates new db
	default:
		InitMetaRecordDB("localhost", "6379", "", false)
	}

	switch Tiering(storageType) {
	default:
		panic(errors.New("Unknown Tiering"))

	case HotOnly:
		hViper := sViper.Sub("hot")
		if hViper == nil {
			panic(ErrHotStorageConfNotProvided)
		}
		Store.Tiering = HotOnly
		Store.HotTier = volumeInit(HOT, hViper, mode) // Will panic if wrong setup is provided

		Store.write = func(b *block.Block) (string, error) {
			data, err := getBlockData(b)
			if err != nil {
				return "", err
			}

			blockPath, err := Store.HotTier.write(b, data)
			if err != nil {
				return "", err
			}

			bwr := &BlockWhereRecord{
				Hash:      b.Hash,
				Tiering:   HotTier,
				BlockPath: blockPath,
			}
			if err := bwr.AddOrUpdate(); err != nil {
				return "", err
			}

			return blockPath, nil
		}

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				Logger.Error(err.Error())
				return
			}

			b, err = readFromDiskTier(bwr.BlockPath)
			if err != nil {
				Logger.Error(err.Error())
			}

			return
		}

	case WarmOnly:
		wViper := sViper.Sub("warm")
		if wViper == nil {
			panic(ErrWarmStorageConfNotProvided)
		}

		Store.Tiering = WarmOnly
		Store.WarmTier = volumeInit(WARM, wViper, mode) // will panic if wrong setup is provided

		Store.write = func(b *block.Block) (string, error) {
			data, err := getBlockData(b)
			if err != nil {
				return "", err
			}

			blockPath, err := Store.WarmTier.write(b, data)
			if err != nil {
				return "", err
			}

			bwr := BlockWhereRecord{
				Hash:      b.Hash,
				Tiering:   WarmTier,
				BlockPath: blockPath,
			}
			if err := bwr.AddOrUpdate(); err != nil {
				return "", err
			}

			return blockPath, nil
		}

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				Logger.Error(err.Error())
				return nil, err
			}

			b, err = readFromDiskTier(bwr.BlockPath)

			if err != nil {
				Logger.Error(err.Error())
			}

			return
		}

	case CacheAndWarm:
		wViper := sViper.Sub("warm")
		if wViper == nil {
			panic(ErrWarmStorageConfNotProvided)
		}

		cViper := sViper.Sub("cache")
		if cViper == nil {
			panic(ErrCacheStorageConfNotProvided)
		}

		writePolicy := cViper.GetString("write_policy")
		switch writePolicy {
		case WriteBack, WriteThrough:
		case "":
			writePolicy = WriteBack
		default:
			panic(ErrCacheWritePolicyNotSupported(writePolicy))
		}

		Store.Tiering = CacheAndWarm
		Store.WarmTier = volumeInit(WARM, wViper, mode) // will panic if wrong setup is provided

		cacheInit(cViper)

		var writeFunc func(b *block.Block) (string, error)

		switch writePolicy {
		case WriteThrough:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.WarmTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   WarmTier,
					BlockPath: blockPath,
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}
				accessTime := time.Now()
				go addToCache(b.Hash, data, &accessTime)

				return blockPath, nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.WarmTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   WarmTier,
					BlockPath: blockPath,
				}
				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}

				return blockPath, nil
			}
		}
		Store.write = writeFunc

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			b, err = readFromCache(hash)
			if err == nil {
				accesTime := time.Now()
				go Store.Cache.UpadateMetaData(hash, &accesTime)
				return
			}

			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				Logger.Error(err.Error())
				return
			}

			b, err = readFromDiskTier(bwr.BlockPath)
			if err != nil {
				return
			}

			go func() {
				data, _ := getBlockData(b)
				accesTime := time.Now()
				addToCache(b.Hash, data, &accesTime)
			}()

			return
		}

	case CacheAndCold:
		coViper := sViper.Sub("cold")
		if coViper == nil {
			panic(ErrColdStorageConfNotProvided)
		}

		cViper := sViper.Sub("cache")
		if cViper == nil {
			panic(ErrCacheStorageConfNotProvided)
		}

		writePolicy := cViper.GetString("write_policy")
		switch writePolicy {
		case WriteBack, WriteThrough:
		case "":
			writePolicy = WriteBack
		default:
			panic(fmt.Errorf("Cache write policy %v is not supported", writePolicy))
		}

		Store.Tiering = CacheAndCold
		Store.ColdTier = coldInit(coViper, mode)
		Store.Cache = cacheInit(cViper)

		var writeFunc func(b *block.Block) (string, error)

		switch writePolicy {
		case WriteThrough:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.ColdTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   ColdTier,
					BlockPath: blockPath,
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}
				accessTime := time.Now()
				go addToCache(b.Hash, data, &accessTime)

				return blockPath, nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.ColdTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   ColdTier,
					BlockPath: blockPath,
				}
				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}

				return blockPath, nil
			}
		}

		Store.write = writeFunc

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			b, err = readFromCache(hash)
			if err == nil {
				accesTime := time.Now()
				go Store.Cache.UpadateMetaData(hash, &accesTime)
				return
			}

			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				return
			}

			b, err = readFromColdTier(bwr.ColdPath, hash)

			if err != nil {
				Logger.Error(err.Error())
			}

			go func() {
				data, _ := getBlockData(b)
				accessTime := time.Now()
				addToCache(hash, data, &accessTime)
			}()

			return
		}

	case HotAndCold:
		hViper := sViper.Sub("hot")
		if hViper == nil {
			panic(ErrHotStorageConfNotProvided)
		}

		cViper := sViper.Sub("cold")
		if cViper == nil {
			panic(ErrColdStorageConfNotProvided)
		}

		Store.Tiering = HotAndCold
		Store.HotTier = volumeInit(HOT, hViper, mode)

		Store.ColdTier = coldInit(cViper, mode)

		Store.write = func(b *block.Block) (string, error) {
			data, err := getBlockData(b)
			if err != nil {
				return "", err
			}

			blockPath, err := Store.HotTier.write(b, data)
			if err != nil {
				return "", err
			}

			bwr := &BlockWhereRecord{
				Hash:      b.Hash,
				Tiering:   HotTier,
				BlockPath: blockPath,
			}
			if err := bwr.AddOrUpdate(); err != nil {
				return "", err
			}

			ub := UnmovedBlockRecord{
				CreatedAt: b.ToTime(),
				Hash:      b.Hash,
			}

			if err := ub.Add(); err != nil {
				return "", err
			}

			return blockPath, nil
		}

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				return
			}

			switch bwr.Tiering {
			case HotTier:
				b, err = readFromDiskTier(bwr.BlockPath)
			case ColdTier:
				b, err = readFromColdTier(bwr.ColdPath, hash)
			case HotAndColdTier:
				b, err = readFromDiskTier(bwr.BlockPath)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr.ColdPath, hash)
			}

			if err != nil {
				Logger.Error(err.Error())
			}

			return
		}
	case WarmAndCold:
		wViper := sViper.Sub("warm")
		if wViper == nil {
			panic(ErrWarmStorageConfNotProvided)
		}

		cViper := sViper.Sub("cold")
		if cViper == nil {
			panic(ErrColdStorageConfNotProvided)
		}

		Store.Tiering = WarmAndCold
		Store.WarmTier = volumeInit(WARM, wViper, mode)
		Store.ColdTier = coldInit(cViper, mode)

		Store.write = func(b *block.Block) (string, error) {
			data, err := getBlockData(b)
			if err != nil {
				return "", err
			}

			blockPath, err := Store.WarmTier.write(b, data)
			if err != nil {
				return "", err
			}

			bwr := BlockWhereRecord{
				Hash:      b.Hash,
				BlockPath: blockPath,
				Tiering:   WarmTier,
			}
			if err := bwr.AddOrUpdate(); err != nil {
				return "", err
			}

			ub := UnmovedBlockRecord{
				CreatedAt: b.ToTime(),
				Hash:      b.Hash,
			}

			if err := ub.Add(); err != nil {
				return "", err
			}

			return blockPath, nil
		}

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				return
			}

			switch bwr.Tiering {
			case WarmTier:
				b, err = readFromDiskTier(bwr.BlockPath)
			case ColdTier:
				b, err = readFromColdTier(bwr.ColdPath, hash)
			case WarmAndColdTier:
				b, err = readFromDiskTier(bwr.BlockPath)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr.BlockPath, hash)
			}

			if err != nil {
				Logger.Error(err.Error())
			}

			return
		}
	case CacheHotAndCold:
		cViper := sViper.Sub("cache")
		if cViper == nil {
			panic(ErrCacheStorageConfNotProvided)
		}

		writePolicy := cViper.GetString("write_policy")
		switch writePolicy {
		case WriteBack, WriteThrough:
		case "":
			writePolicy = DefaultCacheWritePolicy
		default:
			panic(ErrCacheWritePolicyNotSupported(writePolicy))
		}

		hViper := sViper.Sub("hot")
		if hViper == nil {
			panic(ErrHotStorageConfNotProvided)
		}

		coViper := sViper.Sub("cold")
		if coViper == nil {
			panic(ErrColdStorageConfNotProvided)
		}

		Store.Tiering = CacheHotAndCold
		Store.Cache = cacheInit(cViper)
		Store.HotTier = volumeInit(HOT, hViper, mode)
		Store.ColdTier = coldInit(coViper, mode)

		var writeFunc func(b *block.Block) (string, error)

		switch writePolicy {
		case WriteThrough:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.HotTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}
				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}

				ub := UnmovedBlockRecord{
					CreatedAt: b.ToTime(),
					Hash:      b.Hash,
				}

				if err := ub.Add(); err != nil {
					return "", err
				}
				accessTime := time.Now()
				go addToCache(b.Hash, data, &accessTime)

				return blockPath, nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.HotTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}
				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}

				ub := UnmovedBlockRecord{
					CreatedAt: b.ToTime(),
					Hash:      b.Hash,
				}

				if err := ub.Add(); err != nil {
					return "", err
				}

				return blockPath, err
			}
		}

		Store.write = writeFunc

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			b, err = readFromCache(hash)
			if err == nil {
				accesTime := time.Now()
				go Store.Cache.UpadateMetaData(hash, &accesTime)
				return
			}

			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				Logger.Error(err.Error())
				return
			}

			switch bwr.Tiering {
			case HotTier:
				b, err = readFromDiskTier(bwr.BlockPath)
			case ColdTier:
				b, err = readFromColdTier(bwr.ColdPath, hash)
			case HotAndColdTier:
				b, err = readFromDiskTier(bwr.BlockPath)
				if b != nil {
					break
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr.ColdPath, hash)
			}

			if err != nil {
				Logger.Error(err.Error())
				return
			}

			go func() {
				data, _ := getBlockData(b)
				accessTime := time.Now()
				addToCache(hash, data, &accessTime)
			}()

			return
		}

	case CacheWarmAndCold: //
		cViper := sViper.Sub("cache")
		if cViper == nil {
			panic(ErrCacheStorageConfNotProvided)
		}

		writePolicy := cViper.GetString("write_policy")
		switch writePolicy {
		case WriteBack, WriteThrough:
		case "":
			writePolicy = DefaultCacheWritePolicy
		default:
			panic(ErrCacheWritePolicyNotSupported(writePolicy))
		}

		wViper := sViper.Sub("warm")
		if wViper == nil {

			panic(ErrWarmStorageConfNotProvided)
		}

		coViper := sViper.Sub("cold")
		if coViper == nil {
			panic(ErrColdStorageConfNotProvided)
		}

		Store.Tiering = CacheWarmAndCold
		Store.Cache = cacheInit(cViper)
		Store.WarmTier = volumeInit(WARM, wViper, mode)
		Store.ColdTier = coldInit(coViper, mode)

		var writeFunc func(b *block.Block) (string, error)

		switch writePolicy {
		case WriteThrough:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.WarmTier.write(b, data)
				if err != nil {
					return "", err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   WarmTier,
					BlockPath: blockPath,
				}
				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}

				ub := UnmovedBlockRecord{
					CreatedAt: b.ToTime(),
					Hash:      b.Hash,
				}

				if err := ub.Add(); err != nil {
					return "", err
				}
				accessTime := time.Now()
				go addToCache(b.Hash, data, &accessTime)

				return blockPath, nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) (string, error) {
				data, err := getBlockData(b)
				if err != nil {
					return "", err
				}

				blockPath, err := Store.WarmTier.write(b, data)
				if err != nil {
					return "", err
				}
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   WarmTier,
					BlockPath: blockPath,
				}
				if err := bwr.AddOrUpdate(); err != nil {
					return "", err
				}

				ub := UnmovedBlockRecord{
					CreatedAt: b.ToTime(),
					Hash:      b.Hash,
				}

				if err := ub.Add(); err != nil {
					return "", err
				}

				return blockPath, nil
			}
		}

		Store.write = writeFunc

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			b, err = readFromCache(hash)
			if err == nil {
				accesTime := time.Now()
				go Store.Cache.UpadateMetaData(hash, &accesTime)
				return
			}

			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				return
			}

			switch bwr.Tiering {
			case WarmTier:
				b, err = readFromDiskTier(bwr.BlockPath)
			case ColdTier:
				b, err = readFromColdTier(bwr.BlockPath, hash)
			case WarmAndColdTier:
				b, err = readFromDiskTier(bwr.BlockPath)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr.BlockPath, hash)
			}

			go func() {
				data, _ := getBlockData(b)
				accessTime := time.Now()
				addToCache(hash, data, &accessTime)
			}()

			return
		}
	}

	switch Store.Tiering {
	case HotAndCold, WarmAndCold, CacheWarmAndCold, CacheHotAndCold:
		go setupColdWorker(ctx)
		go setupVolumeRevivingWorker(ctx)
	}

	switch Store.Tiering {
	case CacheAndCold, CacheAndWarm, CacheHotAndCold, CacheWarmAndCold:
		go setupCacheReplacement(ctx, Store.Cache)
	}

	return nil
}

func getBlockData(b *block.Block) ([]byte, error) {
	return json.Marshal(b)
}

func readFromDiskTier(bPath string) (b *block.Block, err error) {
	b = new(block.Block)
	f, err := os.Open(bPath)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}
	defer f.Close()

	r, err := zlib.NewReader(f)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}
	defer r.Close()

	err = datastore.ReadJSON(r, b)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}

	return
}

func readFromCache(hash string) (b *block.Block, err error) {
	b = new(block.Block)
	f, err := Store.Cache.Read(hash)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}
	defer f.Close()

	err = datastore.ReadJSON(f, b)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}

	return
}

func readFromColdTier(hash, coldPath string) (b *block.Block, err error) {
	var blockReader io.ReadCloser
	if blockReader, err = Store.ColdTier.read(coldPath, hash); err != nil {
		return
	}
	defer blockReader.Close()

	r, err := zlib.NewReader(blockReader)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}
	defer r.Close()

	err = datastore.ReadJSON(r, b)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}

	return
}

func GetStore() *BlockStore {
	return &Store
}

func addToCache(hash string, data []byte, accessTime *time.Time) {
	if err := Store.Cache.Write(hash, data, accessTime); err != nil {
		Logger.Error(err.Error())
		return
	}
}
