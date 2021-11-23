package blockstore

import (
	"math"
	"os"
	"path/filepath"

	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type (
	volumeMock struct {
		Path                string `mapstructure:"path" yaml:"path"`
		Recovery            bool   `mapstructure:"recovery" yaml:"recovery"`
		SizeToMain          uint64 `mapstructure:"size_to_maintain" yaml:"size_to_maintain"`           // In GB
		InodesToMaintain    uint64 `mapstructure:"inodes_to_maintain" yaml:"inodes_to_maintain"`       // In percent
		AllowedBlockNumbers int    `mapstructure:"allowed_block_numbers" yaml:"allowed_block_numbers"` // Can be any numbers
		AllowedBlockSize    int    `mapstructure:"allowed_block_size" yaml:"allowed_block_size"`       // In GB
	}

	testingT interface {
		require.TestingT
		dirsGenerator
	}

	dirsGenerator interface {
		TempDir() string
	}
)

func (b *volumeMock) convertToMap() map[string]interface{} {
	return map[string]interface{}{
		"path":                  b.Path,
		"recovery":              b.Path,
		"size_to_main":          b.Path,
		"inodes_to_maintain":    b.InodesToMaintain,
		"allowed_block_numbers": b.AllowedBlockNumbers,
		"allowed_block_size":    b.AllowedBlockSize,
	}
}

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
	return map[string]interface{}{
		"block_movies_in": 720,
		"strategy":        "min_size_first",
		"volumes":         mockVolumes(t, 2),
	}
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

	return cfg
}
