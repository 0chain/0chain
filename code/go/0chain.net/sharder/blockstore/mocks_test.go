package blockstore

import (
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
)

type (
	volumeMock struct {
		Path                string `yaml:"path"`
		Recovery            bool   `yaml:"recovery"`
		SizeToMain          uint64 `yaml:"size_to_main"`          // In GB
		InodesToMaintain    uint64 `yaml:"inodes_to_maintain"`    // In percent
		AllowedBlockNumbers int    `yaml:"allowed_block_numbers"` // Can be any numbers
		AllowedBlockSize    int    `yaml:"allowed_block_size"`    // In GB
	}

	testingT interface {
		require.TestingT
		dirsGenerator
	}

	dirsGenerator interface {
		TempDir() string
	}
)

func mockVolume(t testingT) *volumeMock {
	return &volumeMock{
		Path:                t.TempDir(),
		Recovery:            false,
		SizeToMain:          10,
		InodesToMaintain:    10,
		AllowedBlockNumbers: int(math.Pow10(10)),
		AllowedBlockSize:    500 * 1024 * 1024,
	}

}

func mockVolumes(t testingT, size int) []*volumeMock {
	list := make([]*volumeMock, 0, size)
	for i := 0; i < size; i++ {
		list = append(list, mockVolume(t))
	}
	return list
}

func simpleConfigMap(t testingT) map[string]interface{} {
	mainCfg := map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        "min_size_first",
		"volumes":         mockVolumes(t, 5),
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

	cfg.Set("bolt", map[string]string{
		"block_meta_record_path": t.TempDir(),
		"query_meta_record_path": t.TempDir(),
	})

	return cfg
}

func mockHotConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		map[string]interface{}{
			"hot":          simpleConfigMap(t),
			"storage_type": int(HotOnly),
		},
	)

	return cfg
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

func mockFileSystem(path, dirPrefix string, dcl int) (countFiles, size uint64, err error) {
	if dcl < 3 {
		dcl = 3
	}
	for i := 0; i < dcl; i++ {
		subPath := filepath.Join(path, dirPrefix+fmt.Sprint(i))
		_ = os.Mkdir(subPath, 0777)
		for j := 0; j < dcl; j++ {
			sPath := filepath.Join(subPath, fmt.Sprint(j))
			_ = os.Mkdir(sPath, 0777)
			for x := 0; x < dcl; x++ {
				if (i == dcl-1) && (j == dcl-1) && x == dcl-1 {
					continue
				}
				b := mockBlock()
				filePath := filepath.Join(sPath, fmt.Sprintf("%v%v", b.Hash, ".dat"))
				fTemp, err := os.Create(filePath)
				if err != nil {
					log.Fatal(err)
				}
				countFiles++

				info, _ := fTemp.Stat()
				size += uint64(info.Size())
				_ = fTemp.Close()
			}
		}
	}

	return countFiles, size, err
}

func mockDTierMinSizeFirstConfigMap(t testingT) map[string]interface{} {
	mainCfg := map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        minSizeFirst,
		"volumes":         mockVolumes(t, 5),
	}

	return mainCfg
}

func mockDTierRandomConfigMap(t testingT) map[string]interface{} {
	mainCfg := map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        random,
		"volumes":         mockVolumes(t, 5),
	}

	return mainCfg
}

func mockDTierRoundRobinConfigMap(t testingT) map[string]interface{} {
	mainCfg := map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        roundRobin,
		"volumes":         mockVolumes(t, 5),
	}

	return mainCfg
}

func mockDTierMinCountFirstConfigMap(t testingT) map[string]interface{} {
	mainCfg := map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        minCountFirst,
		"volumes":         mockVolumes(t, 5),
	}

	return mainCfg
}

func mockDTierMinSizeFirstConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		mockDTierMinSizeFirstConfigMap(t),
	)

	return cfg
}

func mockDTierRandomConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		mockDTierRandomConfigMap(t),
	)

	return cfg
}

func mockDTierRoundRobinConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		mockDTierRoundRobinConfigMap(t),
	)

	return cfg
}

func mockDTierMinCountFirstConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		mockDTierMinCountFirstConfigMap(t),
	)

	return cfg
}

func mockDTierNilVolumesConfig(t testingT) *viper.Viper {
	cfg := mockConfig(
		t,
		map[string]interface{}{
			"block_movies_in": 720,
			"strategy":        random,
			"volumes":         mockVolumes(t, 0),
		},
	)

	return cfg
}

func mockBWR() *BlockWhereRecord {
	bin, _ := time.Now().MarshalBinary()
	hash := sha3.Sum256(bin)
	return &BlockWhereRecord{
		Hash:      hex.EncodeToString(hash[:]),
		BlockPath: hex.EncodeToString(hash[:]),
		ColdPath:  hex.EncodeToString(hash[:]),
		Tiering:   0,
	}
}

func mockUBR() *UnmovedBlockRecord {
	now := time.Now().Truncate(time.Microsecond)
	bin, _ := time.Now().MarshalBinary()
	hash := sha3.Sum256(bin)
	return &UnmovedBlockRecord{
		Hash:      hex.EncodeToString(hash[:]),
		CreatedAt: now,
	}
}

func mockCacheAccess() *cacheAccess {
	now := time.Now().Truncate(time.Microsecond)
	bin, _ := time.Now().MarshalBinary()
	hash := sha3.Sum256(bin)
	return &cacheAccess{
		Hash:       hex.EncodeToString(hash[:]),
		AccessTime: &now,
	}
}
