package blockstore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
)

func init() {
	memoryStorage := memorystore.GetStorageProvider()
	block.SetupEntity(memoryStorage)
}

func TestBlockStoreWriteReadFromDisk(t *testing.T) {
	basePath := "block_store_path"
	err := os.Mkdir(basePath, 0700)
	require.NoError(t, err)

	defer os.RemoveAll(basePath)

	bStore := &BlockStore{
		basePath:              basePath,
		blockMetadataProvider: datastore.GetEntityMetadata("block"),
	}

	b := new(block.Block)
	b.Hash = "new hash"
	err = bStore.writeToDisk(b.Hash, b)
	require.NoError(t, err)

	bPath := filepath.Join(basePath, getBlockFilePath(b.Hash))

	_, err = os.Stat(bPath)
	require.NoError(t, err)

	b1, err := bStore.readFromDisk(b.Hash)
	require.NoError(t, err)

	require.Equal(t, b.Hash, b1.Hash)
}

func TestBlockStoreWriteReadFromCache(t *testing.T) {
	cachePath := "path_to_cache"
	err := os.Mkdir(cachePath, 0700)
	require.NoError(t, err)

	defer os.RemoveAll(cachePath)
	config := `
cache:
    path: "path_to_cache"
    size: "1GB"
`
	viper.GetViper().SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewReader([]byte(config)))
	require.NoError(t, err)

	bStore := &BlockStore{
		blockMetadataProvider: datastore.GetEntityMetadata("block"),
	}

	require.NotPanics(t, func() {
		bStore.cache = initCache(viper.Sub("cache"))
	})

	b := new(block.Block)
	b.Hash = "new hash"

	err = bStore.writeBlockToCache(b.Hash, b)
	require.NoError(t, err)

	data, err := bStore.cache.Read(b.Hash)
	require.NoError(t, err)

	b1 := bStore.blockMetadataProvider.Instance().(*block.Block)
	r := bytes.NewReader(data)
	err = datastore.ReadMsgpack(r, b1)
	require.NoError(t, err)

	require.Equal(t, b.Hash, b1.Hash)
}
