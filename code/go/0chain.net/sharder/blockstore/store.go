package blockstore

import (
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/viper"
)

type Tiering uint8

const (
	//Cache = 1, Warm = 2, Hot = 4 and Cold = 8
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
	Cache    *cacheTier
	HotTier  *diskTier
	WarmTier *diskTier
	ColdTier *coldTier
	//fields with registered functions as per the config files
	write  func(b *block.Block) (string, error)
	read   func(hash string, round int64) (b *block.Block, err error)
	delete func(hash string) error
}

func (sm *BlockStore) Write(b *block.Block) error {
	Logger.Info("Writing block: " + b.Hash)
	blockPath, err := sm.write(b)
	if err != nil {
		Logger.Error(err.Error())
		panic(err)
	}

	Logger.Info(fmt.Sprintf("Block %v written to %v successfully", b.Hash, blockPath))

	ub := UnmovedBlockRecord{
		CreatedAt: b.ToTime(),
		Hash:      b.Hash,
	}
	if err := ub.Add(); err != nil {
		//do something
	}

	return err
}

func (sm *BlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
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

	var bmrPath, qmrPath string = DefaultBlockMetaRecordDB, DefaultQueryMetaRecordDB
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
	}

	switch mode {
	case "start", "recover":
		InitMetaRecordDB(bmrPath, qmrPath, true) //Removes existing metadata and creates new db
	default:
		InitMetaRecordDB(bmrPath, qmrPath, false)
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
		Store.HotTier = volumeInit(HOT, hViper, mode) //Will panic if wrong setup is provided

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

			b, err = readFromDiskTier(bwr, false)
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
		Store.WarmTier = volumeInit(WARM, wViper, mode) //will panic if wrong setup is provided

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

			b, err = readFromDiskTier(bwr, false)

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

		Store.Tiering = CacheAndWarm
		Store.WarmTier = volumeInit(WARM, wViper, mode) //will panic if wrong setup is provided

		cacheInit(cViper)

		var writeFunc func(b *block.Block) (string, error)

		switch Store.Cache.CacheWrite {
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

				go cacheWrite(bwr, data)

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
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				Logger.Error(err.Error())
				return
			}

			switch bwr.Tiering {

			case CacheAndWarmTier:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					return
				}
				Logger.Error(err.Error())

				b, err = readFromDiskTier(bwr, true)
			case WarmTier:
				b, err = readFromDiskTier(bwr, true)
			}

			if err != nil {
				Logger.Error(err.Error())
			}

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

		Store.Tiering = CacheAndCold
		Store.ColdTier = coldInit(coViper, mode)
		Store.Cache = cacheInit(cViper)

		var writeFunc func(b *block.Block) (string, error)

		switch Store.Cache.CacheWrite {
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

				go cacheWrite(bwr, data)

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
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				return
			}

			switch bwr.Tiering {
			case CacheAndColdTier:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					return
				}
				Logger.Error(err.Error())

				b, err = readFromColdTier(bwr, true)
			case ColdTier:
				b, err = readFromColdTier(bwr, true)
			}

			if err != nil {
				Logger.Error(err.Error())
			}

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
				b, err = readFromDiskTier(bwr, false)
			case ColdTier:
				b, err = readFromColdTier(bwr, false)
			case HotAndColdTier:
				b, err = readFromDiskTier(bwr, false)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, false)
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
				b, err = readFromDiskTier(bwr, false)
			case ColdTier:
				b, err = readFromColdTier(bwr, false)
			case WarmAndColdTier:
				b, err = readFromDiskTier(bwr, false)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, false)
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

		switch Store.Cache.CacheWrite {
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

				go cacheWrite(bwr, data)

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

				return blockPath, err
			}
		}

		Store.write = writeFunc

		Store.read = func(hash string, round int64) (b *block.Block, err error) {
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				Logger.Error(err.Error())
				return
			}

			switch bwr.Tiering {
			case HotTier:
				b, err = readFromDiskTier(bwr, false)
			case ColdTier:
				b, err = readFromColdTier(bwr, true)
			case HotAndColdTier:
				b, err = readFromDiskTier(bwr, false)
				if b != nil {
					break
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, true)
			case CacheAndColdTier:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					break
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, true)
			default:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromDiskTier(bwr, false)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, true)

			}

			if err != nil {
				Logger.Error(err.Error())
			}

			return
		}

	case CacheWarmAndCold: //
		cViper := sViper.Sub("cache")
		if cViper == nil {
			panic(ErrCacheStorageConfNotProvided)
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

		switch Store.Cache.CacheWrite {
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

				go cacheWrite(bwr, data)

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
			var bwr *BlockWhereRecord
			bwr, err = GetBlockWhereRecord(hash)
			if err != nil {
				return
			}

			switch bwr.Tiering {
			case WarmTier:
				b, err = readFromDiskTier(bwr, true)
			case ColdTier:
				b, err = readFromColdTier(bwr, true)
			case WarmAndColdTier:
				b, err = readFromDiskTier(bwr, true)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, true)
			case CacheAndWarmTier:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromDiskTier(bwr, true)
			case CacheAndColdTier:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, true)
			case CacheWarmAndColdTier:
				b, err = readFromCacheTier(bwr)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromDiskTier(bwr, true)
				if b != nil {
					return
				}
				Logger.Error(err.Error())
				b, err = readFromColdTier(bwr, true)
			}
			return
		}
	}

	switch Store.Tiering {
	case HotAndCold, WarmAndCold, CacheWarmAndCold, CacheHotAndCold:
		go setupColdWorker(ctx)
		go setupVolumeRevivingWorker(ctx)
	}
	return nil
}

func cacheWrite(bwr *BlockWhereRecord, data []byte) {
	cachePath, err := Store.Cache.write(bwr.Hash, data)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	bwr.CachePath = cachePath
	switch bwr.Tiering {
	case HotTier:
		bwr.Tiering = CacheAndHotTier
	case WarmTier:
		bwr.Tiering = CacheAndWarmTier
	case ColdTier:
		bwr.Tiering = CacheAndColdTier

	}

	if err := bwr.AddOrUpdate(); err != nil {
		Logger.Error(err.Error())
	}
}

func getBlockData(b *block.Block) ([]byte, error) {
	return json.Marshal(b)
}

func readFromDiskTier(bwr *BlockWhereRecord, shouldCache bool) (b *block.Block, err error) {
	b = new(block.Block)
	f, err := os.Open(bwr.BlockPath)
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

func readFromCacheTier(bwr *BlockWhereRecord) (b *block.Block, err error) {
	bwr, err = GetBlockWhereRecord(bwr.Hash)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}

	f, err := os.Open(bwr.CachePath)
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

func readFromColdTier(bwr *BlockWhereRecord, shouldCache bool) (b *block.Block, err error) {
	bwr, err = GetBlockWhereRecord(bwr.ColdPath)
	if err != nil {
		Logger.Error(err.Error())
		return nil, err
	}

	var blockReader io.ReadCloser
	if blockReader, err = Store.ColdTier.read(bwr.ColdPath, bwr.Hash); err != nil {
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
