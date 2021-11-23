package blockstore

import (
	"math"
	"os"
	"path/filepath"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type (
	volumeMock struct {
		Path                string `yaml:"path"`
		Recovery            bool   `yaml:"recovery"`
		SizeToMain          uint64 `yaml:"size_to_main"`          // In GB
		InodesToMaintain    uint64 `yaml:"inodes_to_maintain"`    // In percent
		AllowedBlockNumbers uint64 `yaml:"allowed_block_numbers"` // Can be any numbers
		AllowedBlockSize    uint64 `yaml:"allowed_block_size"`    // In GB
	}

	testingT interface {
		require.TestingT
		dirsGenerator
	}

	dirsGenerator interface {
		TempDir() string
	}
)

func mockVolumes(t testingT, size int) []volumeMock {
	list := make([]volumeMock, 0, size)
	for i := 0; i < size; i++ {
		vol := volumeMock{
			Path:                t.TempDir(),
			Recovery:            false,
			SizeToMain:          10,
			InodesToMaintain:    10,
			AllowedBlockNumbers: uint64(math.Pow10(10)),
			AllowedBlockSize:    500 * 1024 * 1024,
		}
		list = append(list, vol)
	}
	return list
}

func simpleConfigMap(t testingT) map[string]interface{} {
	mainCfg := map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        "min_size_first",
		"volumes":         mockVolumes(t, 2),
	}

	return mainCfg
}

func mockConfig(t testingT, configMap map[string]interface{}) *viper.Viper {
	filePath := filepath.Join(t.TempDir(), "cfg.yaml")
	file, err := os.Create(filePath)
	require.NoError(t, err)

	err = yaml.NewEncoder(file).Encode(configMap)
	require.NoError(t, err)

	cfg := viper.New()
	err = cfg.ReadConfigFile(filePath)
	require.NoError(t, err)

	cfg.Set(boltCfgKey, map[string]string{
		blockMetaRecordPathMapKey: t.TempDir(),
		queryMetaRecordPathMapKey: t.TempDir(),
	})

	return cfg
}

func mockHotConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		map[string]interface{}{
			"hot":             simpleConfigMap(t),
			storageTypeCfgKey: int(HotTier),
		},
	)
	return cfg
}

func mockHotAndColdConfig(t testingT) *viper.Viper {
	return mockConfig(
		t,
		map[string]interface{}{
			"hot": simpleConfigMap(t),
			"cold": map[string]interface{}{
				"storage": map[string]interface{}{
					"type": "disk",
					"disk": map[string]interface{}{
						"strategy": RoundRobin,
						"volumes":  mockVolumes(t, 2),
					},
				},
			},
			storageTypeCfgKey: int(HotAndCold),
		},
	)
}

func mockBlock() *block.Block {
	ts := time.Now()
	return &block.Block{
		UnverifiedBlockBody: block.UnverifiedBlockBody{
			Round: int64(ts.Nanosecond()),
		},
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash(ts.String()),
		},
	}
}

func mockBlockWhereRecord() *BlockWhereRecord {
	return &BlockWhereRecord{
		Hash:      encryption.Hash(time.Now().String()),
		Tiering:   WarmTier,
		BlockPath: "block-path",
		CachePath: "cache-path",
		ColdPath:  "cold-path",
	}
}
