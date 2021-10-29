package smartblockstore

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	fileExt = ".dat"
	//strategies
	Random        = "random"
	RoundRobin    = "round_robin"
	MinSizeFirst  = "min_size_first"
	MinCountFirst = "min_count_first"

	//DefaultStrategies
	DefaultHotStrategy   = "random"
	DefaultWarmStrategy  = "random"
	DefaultColdStrategy  = "random"
	DefaultCacheStrategy = "random"
)

//Common errors
var (
	ErrInodesLimit = func(vPath string) error {
		return fmt.Errorf("Volume %v has less than 10%% available inodes", vPath)
	}

	ErrSizeLimit = func(vPath string) error {
		return fmt.Errorf("Volume %v has less than 2GB available space", vPath)
	}

	ErrVolumeFull = func(volPath string) error {
		return fmt.Errorf("Volume %v is full.", volPath)
	}

	ErrSelectDir = func(volPath string, err error) string {
		return fmt.Sprintf("Error while selecting dir; volume path: %v, err: %v", volPath, err)
	}

	ErrStrategyNotSupported = func(strategy string) error {
		return fmt.Errorf("Strategy %v is not supported", strategy)
	}

	ErrStorageTypeNotSupported = func(storageType string) error {
		return fmt.Errorf("Storage type %v is not supported", storageType)
	}
)

func countFiles(dirPath string) (count int, err error) {
	var f *os.File
	f, err = os.Open(dirPath)
	if err != nil {
		return
	}
	defer f.Close()
	var dirs []os.DirEntry
	for {
		dirs, err = f.ReadDir(1000)
		if errors.Is(err, io.EOF) {
			err = nil
			break
		}
		count += len(dirs)
		if err != nil {
			return
		}
	}
	return
}
