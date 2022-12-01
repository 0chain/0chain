package blockstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("debug", ".")
}

func getViperConfig(in *input, wd string) (*viper.Viper, error) {
	if len(in.volumes) == 0 {
		return nil, errors.New("at least one volume is required")
	}
	v := viper.New()
	v.Set("storage_type", int(in.storageType))
	v.Set("mode", in.mode)
	v.Set("disk.strategy", in.strategy)
	v.Set("rocks.dir_name", in.rocksDirname)

	vols := make([]interface{}, len(in.volumes))
	for i, m := range in.volumes {
		dir := filepath.Join(wd, m["path"].(string))
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, err
		}
		m["path"] = dir
		vols[i] = m
	}
	v.Set("disk.volumes", vols)

	if in.cache != nil {
		dir := filepath.Join(wd, in.cache["path"].(string))
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, err
		}
		v.Set("cache.path", dir)
		v.Set("cache.size", in.cache["size"])
	}

	if in.cloudStorages != nil {
		v.Set("block_movement_interval", in.blockMovementInterval)
		v.Set("cold.delete_local", in.deleteLocal)
		v.Set("cold.strategy", in.coldStrategy)

		vStorages := make([]interface{}, len(in.cloudStorages))
		for i, m := range in.cloudStorages {
			vStorages[i] = m
		}
		v.Set("cold.cloud_storages", vStorages)
	}

	return v, nil
}

type testType int

const (
	allowedBlockNumbers testType = iota
	allowedBlockSize
	dirNameTest
)

type input struct {
	name                  string
	rocksDirname          string
	mode                  string
	strategy              string
	storageType           Tiering
	blockMovementInterval time.Duration
	cache                 map[string]interface{}
	volumes               []map[string]interface{}
	coldStrategy          string
	cloudStorages         []map[string]interface{}
	deleteLocal           bool
	typeOfTest            testType
	callback              func(in *input, wd string)
}

func TestBlockStoreComponentInit(t *testing.T) {

	inputs := []*input{
		{
			name:         "Allowed number constraint",
			strategy:     RoundRobin,
			storageType:  DiskOnly,
			rocksDirname: "num_constraint",
			volumes: []map[string]interface{}{
				{
					"path":                  "vol1",
					"allowed_block_numbers": 10,
				},
				{
					"path": "vol2",
				},
			},
			typeOfTest: allowedBlockNumbers,
			callback: func(in *input, wd string) {
				os.RemoveAll(wd)
			},
		},
		{
			name:         "Allowed size constraint",
			rocksDirname: "size_constraint",
			strategy:     RoundRobin,
			storageType:  DiskOnly,
			volumes: []map[string]interface{}{
				{
					"path":               "vol2",
					"allowed_block_size": 1 * KB,
				},
			},
			typeOfTest: allowedBlockSize,
			callback: func(in *input, wd string) {
				os.RemoveAll(wd)
			},
		},
		{
			name:         "Should create new dir after directory content limit is reached",
			strategy:     RoundRobin,
			rocksDirname: "dcl_reached",
			storageType:  DiskOnly,
			volumes: []map[string]interface{}{
				{
					"path": "vol",
				},
			},
			typeOfTest: dirNameTest,
			callback: func(in *input, wd string) {
				os.RemoveAll(wd)
			},
		},
		{
			name:         "Read from cache",
			strategy:     RoundRobin,
			rocksDirname: "read_from_cache",
			storageType:  CacheAndDisk,
			volumes: []map[string]interface{}{
				{
					"path": "vol1",
				},
			},
			cache: map[string]interface{}{
				"path": "cache",
				"size": 500 * MB,
			},
			callback: func(in *input, wd string) {
				os.RemoveAll(wd)
			},
		},
		{
			name:                  "Test block movement with delete local",
			deleteLocal:           true,
			rocksDirname:          "block_movement",
			storageType:           DiskAndCold,
			blockMovementInterval: time.Second * 3,
			volumes: []map[string]interface{}{
				{
					"path": "vol1",
				},
			},
			cloudStorages: []map[string]interface{}{
				{
					"path": "cold1",
				},
			},
			callback: func(in *input, wd string) {
				os.RemoveAll(wd)
			},
		},
		// {
		// 	name:                  "Next block read should be from cache after a block is read from cold storage",
		// 	rocksDirname:          "cache_use",
		// 	deleteLocal:           true,
		// 	storageType:           CacheDiskAndCold,
		// 	blockMovementInterval: time.Second * 2,
		// 	volumes: []map[string]interface{}{
		// 		{
		// 			"path": "vol1",
		// 		},
		// 	},
		// 	cloudStorages: []map[string]interface{}{
		// 		{
		// 			"path": "cold1",
		// 		},
		// 	},
		// 	cache: map[string]interface{}{
		// 		"path": "cache",
		// 		"size": 500 * MB,
		// 	},
		// 	callback: func(in *input, wd string) {
		// 		os.RemoveAll(wd)
		// 	},
		// },
	}

	for _, in := range inputs {
		t.Run(in.name, func(t *testing.T) {
			wd := "./mnt"
			err := os.MkdirAll(wd, 0777)
			require.NoError(t, err)

			if in.callback != nil {
				defer in.callback(in, wd)
			}

			switch in.storageType {
			case DiskOnly:
				testDiskOnlyStorage(t, in, wd)
			case CacheAndDisk:
				testCacheAndDiskStorage(t, in, wd)
			case DiskAndCold:
				testDiskAndColdStorage(t, in, wd)
			case CacheDiskAndCold:
				testCacheDiskAndColdStorage(t, in, wd)
			}

		})
	}
}

func testDiskOnlyStorage(t *testing.T, in *input, wd string) {
	v, err := getViperConfig(in, wd)
	require.NoError(t, err)
	require.NotNil(t, v)

	require.NotPanics(t, func() {
		Init(context.Background(), v, wd)
	})

	require.NotNil(t, Store)
	store := Store.(*blockStore)
	require.NotNil(t, store.diskTier)
	require.Zero(t, store.blockMovementInterval)
	require.Nil(t, store.cache)
	require.Nil(t, store.coldTier)

	err = GetStore().Write(nil)
	require.Error(t, err, err)

	b := new(block.Block)
	b.Hash = "block hash"
	b.Signature = "signature"

	err = GetStore().Write(b)

	require.NoError(t, err)

	bwr, err := getBWR(b.Hash)
	require.NoError(t, err, err)

	require.Equal(t, DiskTier, bwr.Tiering)
	require.NotZero(t, bwr.BlockPath)
	require.Zero(t, bwr.ColdPath)

	b1, err := GetStore().Read(b.Hash, 0)
	require.NoError(t, err)
	require.Equal(t, b.Hash, b1.Hash)
	require.Equal(t, b.Signature, b1.Signature)

	switch in.typeOfTest {
	case allowedBlockNumbers:
		testDiskAllowedNumberConstraint(t, store)
	case allowedBlockSize:
		testDiskAllowedSizeConstraint(t, store)
	case dirNameTest:
		testDirNameChange(t, store, wd)

	}
}

func testDiskAllowedNumberConstraint(t *testing.T, store *blockStore) {
	vol := store.diskTier.Volumes[0]
	for i := 1; i < len(store.diskTier.Volumes); i++ {
		if i >= len(store.diskTier.Volumes) {
			break
		}

		if store.diskTier.Volumes[i].AllowedBlockNumbers != 0 &&
			(store.diskTier.Volumes[i].AllowedBlockNumbers < vol.AllowedBlockNumbers ||
				vol.AllowedBlockNumbers == 0) {
			vol = store.diskTier.Volumes[i]
		}
	}
	for i := uint64(0); i < vol.AllowedBlockNumbers*uint64(len(store.diskTier.Volumes))+
		uint64(len(store.diskTier.Volumes)); i++ {
		b := new(block.Block)
		b.Hash = fmt.Sprintf("hash#%d", i)
		err := GetStore().Write(b)
		require.NoError(t, err, err)
	}

	ableToStore := vol.isAbleToStoreBlock()
	require.False(t, ableToStore)
	require.Equal(t, vol.AllowedBlockNumbers, vol.BlocksCount)
	_, ok := unableVolumes[vol.Path]
	require.True(t, ok)

	for i := 0; i < len(store.diskTier.Volumes); i++ {
		require.NotEqual(t, vol.Path, store.diskTier.Volumes[i].Path)
	}
}

func testDiskAllowedSizeConstraint(t *testing.T, store *blockStore) {
	require.Equal(t, 1, len(store.diskTier.Volumes),
		"Add single volume for size constraint test")

	vol := store.diskTier.Volumes[0]
	vol.AllowedBlockSize /= GB
	require.NotEqual(t, vol.AllowedBlockSize, 0) // 0 means there is no limit
	var i int
	var shouldBreak bool
	for {
		b := new(block.Block)
		b.Hash = fmt.Sprintf("hash#%d", i)
		func() {
			defer func() {
				if i := recover(); i != nil {
					shouldBreak = true
				}
			}()
			GetStore().Write(b)
		}()
		if shouldBreak {
			break
		}
		i++
	}

	_, ok := unableVolumes[vol.Path]
	require.True(t, ok)

	for i := 0; i < len(store.diskTier.Volumes); i++ {
		require.NotEqual(t, vol.Path, store.diskTier.Volumes[i].Path)
	}
}

func testDirNameChange(t *testing.T, store *blockStore, wd string) {
	require.Equal(t, 1, len(store.diskTier.Volumes),
		"Add single volume for size constraint test")
	vol := store.diskTier.Volumes[0]

	for i := 0; i < DirectoryContentLimit+1; i++ {
		b := new(block.Block)
		b.Hash = fmt.Sprintf("hash#%d", i)
		err := GetStore().Write(b)
		require.NoError(t, err, err)
	}

	require.Equal(t, 1, vol.CurDirInd)
	require.Equal(t, 0, vol.CurKInd)

	newBlock := new(block.Block)
	newBlock.Hash = "newhash"
	expectedBlockPath := filepath.Join(
		vol.Path, fmt.Sprintf("%s%d", DirPrefix, vol.CurKInd),
		fmt.Sprint(vol.CurDirInd), fmt.Sprintf("%s%s", newBlock.Hash, fileExt),
	)
	err := GetStore().Write(newBlock)
	require.NoError(t, err)

	bwr, err := getBWR(newBlock.Hash)
	require.NoError(t, err)

	require.Equal(t, expectedBlockPath, bwr.BlockPath)
}

func testCacheAndDiskStorage(t *testing.T, in *input, wd string) {
	v, err := getViperConfig(in, wd)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		Init(context.Background(), v, wd)
	})

	require.NotNil(t, Store)
	store := Store.(*blockStore)
	require.NotNil(t, store.diskTier)
	require.Zero(t, store.blockMovementInterval)
	require.NotNil(t, store.cache)
	require.Nil(t, store.coldTier)

	err = GetStore().Write(nil)
	require.Error(t, err, err)

	b := new(block.Block)
	b.Hash = "blockhash"
	b.Signature = "signature"

	err = GetStore().Write(b)

	require.NoError(t, err)

	bwr, err := getBWR(b.Hash)
	require.NoError(t, err, err)

	require.Equal(t, DiskTier, bwr.Tiering)
	require.NotZero(t, bwr.BlockPath)
	require.Zero(t, bwr.ColdPath)

	time.Sleep(time.Second * 2)
	data, err := store.cache.Read(b.Hash)
	require.NoError(t, err, err)
	b1 := new(block.Block)
	r := bytes.NewReader(data)
	err = datastore.ReadJSON(r, b1)

	require.NoError(t, err, err)
	require.Equal(t, b.Hash, b1.Hash)
	require.Equal(t, b.Signature, b1.Signature)
}

func testDiskAndColdStorage(t *testing.T, in *input, wd string) {
	in.storageType = DiskOnly
	v, err := getViperConfig(in, wd)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		Init(context.Background(), v, wd)
	})

	require.NotNil(t, Store)
	store := Store.(*blockStore)
	in.storageType = DiskAndCold
	cTier := &coldTier{
		Mu: make(Mutex, 1),
	}
	storageSelectorChan := make(chan selectedColdStorage, 1)
	cTier.SelectNextStorage = getColdRBStrategyFunc(cTier, storageSelectorChan)
	cTier.StorageSelectorChan = storageSelectorChan

	t.Log("Registering cold storage")
	registerMockColdStorage(t, in, wd, cTier)

	go cTier.SelectNextStorage(cTier.ColdStorages, 0)

	store.coldTier = cTier

	store.blockMovementInterval = time.Second * 2
	if in.blockMovementInterval != 0 {
		store.blockMovementInterval = in.blockMovementInterval
	}

	go setupColdWorker(context.Background())

	b := new(block.Block)
	b.Hash = "coldhash"
	b.CreationDate = common.Now()

	err = GetStore().Write(b)
	require.NoError(t, err)
	err = store.addToUBR(b)
	require.NoError(t, err)
	time.Sleep(store.blockMovementInterval * 3)

	bwr, err := getBWR(b.Hash)
	require.NoError(t, err)
	require.NotZero(t, bwr.ColdPath, fmt.Sprintf("Bwr: %+v", bwr))
	if cTier.DeleteLocal {
		require.Zero(t, bwr.BlockPath)
	}
}

func testCacheDiskAndColdStorage(t *testing.T, in *input, wd string) {
	in.storageType = CacheAndDisk
	v, err := getViperConfig(in, wd)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		Init(context.Background(), v, wd)
	})

	require.NotNil(t, Store)
	store := Store.(*blockStore)
	in.storageType = CacheDiskAndCold
	cTier := &coldTier{
		Mu: make(Mutex, 1),
	}
	storageSelectorChan := make(chan selectedColdStorage, 1)
	cTier.SelectNextStorage = getColdRBStrategyFunc(cTier, storageSelectorChan)
	cTier.StorageSelectorChan = storageSelectorChan

	t.Log("Registering cold storage")
	registerMockColdStorage(t, in, wd, cTier)

	store.coldTier = cTier

	store.blockMovementInterval = time.Second * 2
	if in.blockMovementInterval != 0 {
		store.blockMovementInterval = in.blockMovementInterval
	}

	go setupColdWorker(context.Background())

	b := new(block.Block)
	b.Hash = "cachecoldhash"
	b.CreationDate = common.Now()

	err = GetStore().Write(b)
	require.NoError(t, err)
	err = store.addToUBR(b)
	require.NoError(t, err)
	data, err := store.cache.Read(b.Hash)
	require.NoError(t, err)

	b1 := new(block.Block)
	err = datastore.FromJSON(data, b1)
	require.NoError(t, err)

	require.Equal(t, b.Hash, b1.Hash)

	cache := store.cache.(*cache)
	cPath := filepath.Join(cache.path, b.Hash)
	err = os.Remove(cPath)
	require.NoError(t, err)

	time.Sleep(store.blockMovementInterval * 2) // wait for block to move

	b, err = GetStore().Read(b.Hash, 0)
	require.NoError(t, err)

	fStat, err := os.Stat(cPath)
	require.NoError(t, err)

	require.Equal(t, b.Hash, fStat.Name())
}

func registerMockColdStorage(t *testing.T, in *input, wd string, cTier *coldTier) {
	require.GreaterOrEqual(t, len(in.cloudStorages), 1)
	coldStoragesMap = make(map[string]coldStorageProvider)
	for _, m := range in.cloudStorages {
		p := filepath.Join(wd, m["path"].(string))
		err := os.MkdirAll(p, 0777)
		require.NoError(t, err)
		var allowedBlockNumbers uint64
		if m["allowed_block_numbers"] != nil {
			allowedBlockNumbers = m["allowed_block_numbers"].(uint64)
		}
		coldStorage := MockColdStorage{
			path:                p,
			allowedBlockNumbers: allowedBlockNumbers,
		}

		coldStoragesMap[p] = coldStorage
		cTier.ColdStorages = append(cTier.ColdStorages, coldStorage)
	}

	cTier.DeleteLocal = in.deleteLocal
}

type MockColdStorage struct {
	path                string
	allowedBlockNumbers uint64
}

func (mc MockColdStorage) moveBlock(hash, blockpath string) (string, error) {
	newPath := filepath.Join(mc.path, hash)
	err := os.Rename(filepath.Join("./", blockpath), newPath)
	if err != nil {
		return "", err
	}
	return newPath, nil
}

func (mc MockColdStorage) getBlock(hash string) ([]byte, error) {
	p := filepath.Join(mc.path, hash)
	return os.ReadFile(p)
}
