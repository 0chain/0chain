package smartblockstore

import (
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
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
	ErrInodesLimit = func(vPath string, inodesToMaintain uint64) error {
		return fmt.Errorf("Volume %v has inodes lesser than inodes to maintain, %v", vPath, inodesToMaintain)
	}

	ErrSizeLimit = func(vPath string, sizeToMaintain uint64) error {
		return fmt.Errorf("Volume %v has size lesser than size to maintain, %vGB", vPath, sizeToMaintain)
	}

	ErrAllowedSizeLimit = func(vPath string, allowedSizeLimit uint64) error {
		return fmt.Errorf("Allowed size limit, %v, for volume %v reached.", allowedSizeLimit, vPath)
	}

	ErrAllowedCountLimit = func(vPath string, allowedCountLimit uint64) error {
		return fmt.Errorf("Allowed block number limit, %v, for volume %v reached.", allowedCountLimit, vPath)
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

	ErrFiftyPercent                = errors.New("At least 50%% volumes must be able to store blocks")
	ErrCacheStorageConfNotProvided = errors.New("Storage type includes cache but cache config not provided")
	ErrHotStorageConfNotProvided   = errors.New("Storage type includes hot tier but hot tier config not provided")
	ErrWarmStorageConfNotProvided  = errors.New("Storage type includes warm tier but warm tier config not provided")
	ErrColdStorageConfNotProvided  = errors.New("Storage type includes cold tier but cold tier config not provided")
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

func getAvailableSizeAndInodes(vPath string) (availableSize, availableInodes uint64, err error) {
	var volStat unix.Statfs_t
	err = unix.Statfs(vPath, &volStat)
	if err != nil {
		return
	}

	availableInodes = volStat.Ffree
	availableSize = volStat.Bfree * uint64(volStat.Bsize)
	return
}
