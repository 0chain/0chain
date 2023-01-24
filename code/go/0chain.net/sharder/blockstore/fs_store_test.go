package blockstore

import (
	"bytes"
	"fmt"
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

func TestGetMPrefixDir(t *testing.T) {

	type input struct {
		mPrefix        string
		roundRemainder int64
	}
	tests := map[int64]*input{
		1: {
			mPrefix:        fmt.Sprintf("%s%d", mPrefix, 0),
			roundRemainder: 1,
		},
		twoMillion: {
			mPrefix:        fmt.Sprintf("%s%d", mPrefix, 1),
			roundRemainder: 0,
		},
		twoMillion*2 + 1: {
			mPrefix:        fmt.Sprintf("%s%d", mPrefix, 2),
			roundRemainder: 1,
		},
		twoMillion*300 + 1000: {
			mPrefix:        fmt.Sprintf("%s%d", mPrefix, 300),
			roundRemainder: 1000,
		},
	}

	for round := range tests {
		t.Run(fmt.Sprintf("Test for round %d", round), func(t *testing.T) {
			in := tests[round]
			require.NotNil(t, in)
			mPrefix, roundRemainder := getMPrefixDir(round)
			require.Equal(t, in.mPrefix, mPrefix)
			require.Equal(t, in.roundRemainder, roundRemainder)
		})
	}
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
	b.Round = 1
	err = bStore.writeToDisk(b)
	require.NoError(t, err)

	bPath := filepath.Join(basePath, getBlockFilePath(b.Hash, b.Round))

	_, err = os.Stat(bPath)
	require.NoError(t, err)

	b1, err := bStore.readFromDisk(b.Hash, b.Round)
	require.NoError(t, err)

	require.Equal(t, b.Hash, b1.Hash)
	require.Equal(t, b.Round, b1.Round)
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
	b.Round = 1

	err = bStore.writeBlockToCache(b)
	require.NoError(t, err)

	data, err := bStore.cache.Read(b.Hash)
	require.NoError(t, err)

	b1 := bStore.blockMetadataProvider.Instance().(*block.Block)
	r := bytes.NewReader(data)
	err = datastore.ReadMsgpack(r, b1)
	require.NoError(t, err)

	require.Equal(t, b.Hash, b1.Hash)
	require.Equal(t, b.Round, b1.Round)
}
