package blockstore

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
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

func mockTierVolumes(path string, size int) []*volume {
	list := make([]*volume, 0, size)
	for i := 0; i < size; i++ {
		vol := mockVolume(path, 0, 0, 0)
		list = append(list, &vol)
	}
	return list
}

func mockVolume(path string, kInd, CurDirInd, CurDirBlockNums int) volume {
	return volume{
		Path:                path,
		SizeToMaintain:      10,
		InodesToMaintain:    10,
		AllowedBlockNumbers: uint64(math.Pow10(10)),
		AllowedBlockSize:    500 * 1024 * 1024,
		CurKInd:             kInd,
		CurDirInd:           CurDirInd,
		CurDirBlockNums:     CurDirBlockNums,
	}
}

func mockFileSystem(path, dirPrefix, fileName string, dcl int) (countFiles, size uint64, err error) {
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
				fName := fileName + "_" + strconv.Itoa(x)
				fileForCount := filepath.Join(sPath, fName)
				fTemp, err := os.Create(fileForCount)
				if err != nil {
					log.Fatal(err)
				}
				countFiles++
				for j := 0; j < 100; j++ {
					_, _ = fTemp.WriteString("Hello, Bench\n")
				}

				info, _ := fTemp.Stat()
				size += uint64(info.Size())
				fTemp.Close()

			}
		}
	}

	return countFiles, size, err
}

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
		"volumes":         mockVolumes(t, 0),
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
			"storage_type": 4,
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
