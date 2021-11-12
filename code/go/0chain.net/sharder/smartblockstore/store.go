package smartblockstore

import (
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
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
	HOT   = "Hot"
	WARM  = "Warm"
	CACHE = "Cache"
	COLD  = "Cold"
)

var smartStore SmartStore
var workerMap map[string]interface{}

type SmartStore struct {
	Tiering  Tiering
	Cache    *cacheTier
	HotTier  *diskTier
	WarmTier *diskTier
	ColdTier *coldTier
	//fields with registered functions as per the config files
	write  func(b *block.Block) error
	read   func(hash string, round int64) (b *block.Block, err error)
	delete func(hash string) error
}

func (sm *SmartStore) Write(b *block.Block) error {
	return sm.write(b)
}

func (sm *SmartStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	return sm.read(bs.Hash, bs.Round)
}

func (sm *SmartStore) Read(hash string, round int64) (b *block.Block, err error) {
	return sm.read(hash, round)
}

func (sm *SmartStore) Delete(hash string) error {
	return nil // Not implemented
}

func InitializeSmartStore(sConf map[string]interface{}, ctx context.Context) error {
	var mode string
	var storageType int

	storageTypeI, ok := sConf["storage_type"]
	if !ok {
		panic(errors.New("Storage Type is a required field"))
	}
	storageType = storageTypeI.(int)

	modeI, ok := sConf["mode"]
	if !ok {
		mode = "start"
	} else {
		mode = modeI.(string)
	}

	switch mode {
	case "start", "recover":
		InitMetaRecordDB(true) //Removes existing metadata and creates new db
	default:
		InitMetaRecordDB(false)
	}

	switch Tiering(storageType) {
	default:
		panic(errors.New("Unknown Tiering"))

	case HotOnly:
		hotI, ok := sConf["hot"]
		if !ok {
			panic(ErrHotStorageConfNotProvided)
		}
		hotMap := hotI.(map[string]interface{})

		smartStore.HotTier = volumeInit(HOT, hotMap, mode) //Will panic if wrong setup is provided

		smartStore.write = func(b *block.Block) error {
			data, err := getBlockData(b)
			if err != nil {
				Logger.Error(err.Error())
				return err
			}

			smartStore.HotTier.write(b, data)
			return nil
		}

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		warmI, ok := sConf["warm"]
		if !ok {
			panic(ErrWarmStorageConfNotProvided)
		}

		warmMap := warmI.(map[string]interface{})

		smartStore.WarmTier = volumeInit(WARM, warmMap, mode) //will panic if wrong setup is provided

		smartStore.write = func(b *block.Block) error {
			data, err := getBlockData(b)
			if err != nil {
				Logger.Error(err.Error())
				return err
			}

			smartStore.WarmTier.write(b, data)
			return nil
		}

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		warmI, ok := sConf["warm"]
		if !ok {
			panic(ErrWarmStorageConfNotProvided)
		}

		cacheI, ok := sConf["cache"]
		if !ok {
			panic(ErrCacheStorageConfNotProvided)
		}

		warmMap := warmI.(map[string]interface{})
		smartStore.WarmTier = volumeInit(WARM, warmMap, mode)

		cacheMap := cacheI.(map[string]interface{})
		cacheInit(cacheMap)

		var writeFunc func(b *block.Block) error

		switch smartStore.Cache.CacheWrite {
		case WriteThrough:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.WarmTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				go cacheWrite(bwr, data)

				return nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.WarmTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				return nil
			}
		}
		smartStore.write = writeFunc

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		coldI, ok := sConf["cold"]
		if !ok {
			panic(ErrColdStorageConfNotProvided)
		}

		cacheI, ok := sConf["cache"]
		if !ok {
			panic(ErrCacheStorageConfNotProvided)
		}

		coldMap := coldI.(map[string]interface{})
		smartStore.ColdTier = coldInit(coldMap, mode)

		cacheMap := cacheI.(map[string]interface{})
		cacheInit(cacheMap)

		var writeFunc func(b *block.Block) error

		switch smartStore.Cache.CacheWrite {
		case WriteThrough:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.ColdTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				go cacheWrite(bwr, data)

				return nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.ColdTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				return nil
			}
		}

		smartStore.write = writeFunc

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		hotI, ok := sConf["hot"]
		if !ok {
			panic(ErrHotStorageConfNotProvided)
		}

		coldI, ok := sConf["cold"]
		if !ok {
			panic(ErrColdStorageConfNotProvided)
		}

		hotMap := hotI.(map[string]interface{})
		smartStore.HotTier = volumeInit(HOT, hotMap, mode)

		coldMap := coldI.(map[string]interface{})
		smartStore.ColdTier = coldInit(coldMap, mode)

		smartStore.write = func(b *block.Block) error {
			data, err := getBlockData(b)
			if err != nil {
				Logger.Error(err.Error())
				return err
			}

			smartStore.HotTier.write(b, data)
			return nil
		}

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		warmI, ok := sConf["warm"]
		if !ok {
			panic(ErrWarmStorageConfNotProvided)
		}

		coldI, ok := sConf["cold"]
		if !ok {
			panic(ErrColdStorageConfNotProvided)
		}

		warmMap := warmI.(map[string]interface{})
		smartStore.WarmTier = volumeInit(WARM, warmMap, mode)

		coldMap := coldI.(map[string]interface{})
		smartStore.ColdTier = coldInit(coldMap, mode)

		smartStore.write = func(b *block.Block) error {
			data, err := getBlockData(b)
			if err != nil {
				Logger.Error(err.Error())
				return err
			}

			smartStore.WarmTier.write(b, data)
			return nil
		}

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		cacheI, ok := sConf["cache"]
		if !ok {
			panic(ErrCacheStorageConfNotProvided)
		}
		hotI, ok := sConf["hot"]
		if !ok {
			panic(ErrHotStorageConfNotProvided)
		}

		coldI, ok := sConf["cold"]
		if !ok {
			panic(ErrColdStorageConfNotProvided)
		}

		cacheMap := cacheI.(map[string]interface{})
		cacheInit(cacheMap)

		hotMap := hotI.(map[string]interface{})
		smartStore.HotTier = volumeInit(HOT, hotMap, mode)

		coldMap := coldI.(map[string]interface{})
		smartStore.ColdTier = coldInit(coldMap, mode)

		var writeFunc func(b *block.Block) error

		switch smartStore.Cache.CacheWrite {
		case WriteThrough:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.HotTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				go cacheWrite(bwr, data)
				return nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.HotTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				return nil
			}
		}

		smartStore.write = writeFunc

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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
		cacheI, ok := sConf["cache"]
		if !ok {
			panic(ErrCacheStorageConfNotProvided)
		}
		warmI, ok := sConf["warm"]
		if !ok {
			panic(ErrWarmStorageConfNotProvided)
		}

		coldI, ok := sConf["cold"]
		if !ok {
			panic(ErrColdStorageConfNotProvided)
		}

		cacheMap := cacheI.(map[string]interface{})
		cacheInit(cacheMap)

		warmMap := warmI.(map[string]interface{})
		smartStore.WarmTier = volumeInit(WARM, warmMap, mode)

		coldMap := coldI.(map[string]interface{})
		smartStore.ColdTier = coldInit(coldMap, mode)

		var writeFunc func(b *block.Block) error

		switch smartStore.Cache.CacheWrite {
		case WriteThrough:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.WarmTier.write(b, data)
				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				if err := bwr.AddOrUpdate(); err != nil {
					return err
				}

				go cacheWrite(bwr, data)

				return nil
			}
		case WriteBack:
			writeFunc = func(b *block.Block) error {
				data, err := getBlockData(b)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				blockPath, err := smartStore.WarmTier.write(b, data)
				if err != nil {
					Logger.Error(err.Error())
					return err
				}

				bwr := &BlockWhereRecord{
					Hash:      b.Hash,
					Tiering:   HotTier,
					BlockPath: blockPath,
				}

				if err := bwr.AddOrUpdate(); err != nil {
					Logger.Error(err.Error())
					return err
				}
				return nil
			}
		}

		smartStore.write = writeFunc

		smartStore.read = func(hash string, round int64) (b *block.Block, err error) {
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

	return nil
}

func cacheWrite(bwr *BlockWhereRecord, data []byte) {
	cachePath, err := smartStore.Cache.write(bwr.Hash, data)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	bwr.CachePath = cachePath
	bwr.Tiering = CacheAndHotTier
	if err := bwr.AddOrUpdate(); err != nil {
		Logger.Error(err.Error())
	}
}

func getBlockData(b *block.Block) ([]byte, error) {
	return json.Marshal(b)
}

func readFromDiskTier(bwr *BlockWhereRecord, shouldCache bool) (b *block.Block, err error) {
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
	if blockReader, err = smartStore.ColdTier.read(bwr.ColdPath, bwr.Hash); err != nil {
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

func GetStore() *SmartStore {
	return &smartStore
}
