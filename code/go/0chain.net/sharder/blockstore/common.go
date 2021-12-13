package blockstore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	fileExt            = ".dat"
	IndexStateFileName = "index.state"

	// strategies
	Random        = "random"
	RoundRobin    = "round_robin"
	MinSizeFirst  = "min_size_first"
	MinCountFirst = "min_count_first"

	// DefaultStrategies
	DefaultVolumeStrategy = "random"
	DefaultColdStrategy   = "random"
	DefaultCacheStrategy  = "random"

	// BlockLimit Limit the number of blocks to store in any tier
	BlockLimitNumber = 1000000000 // 10 powered 9
	GB               = 1024 * 1024 * 1024
)

type (
	CountBlocksInVolumes         func(vPath, dirPrefix string, dcl int) (uint64, uint64)
	CountFiles                   func(dirPath string) (count int, err error)
	GetAvailableSizeAndInodes    func(vPath string) (availableSize, totalInodes, availableInodes uint64, err error)
	GetCurrentDirBlockNums       func(dir string) (int, error)
	GetCurIndexes                func(fPath string) (curKInd, curDirInd int, err error)
	GetUint64ValueFromYamlConfig func(v interface{}) (uint64, error)
	UpdateCurIndexes             func(fPath string, curKInd, curDirInd int) error
)

// Common errors
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
	ErrUnableToSelectVolume        = errors.New("Unable to select any available volume")
	ErrUnableToSelectColdStorage   = errors.New("Unable to select any available cold storage")
)

var countFiles CountFiles = func(dirPath string) (count int, err error) {
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

var getAvailableSizeAndInodes GetAvailableSizeAndInodes = func(vPath string) (availableSize, totalInodes, availableInodes uint64, err error) {
	var volStat unix.Statfs_t
	err = unix.Statfs(vPath, &volStat)
	if err != nil {
		return
	}

	availableInodes = volStat.Ffree
	totalInodes = volStat.Files
	availableSize = volStat.Bfree * uint64(volStat.Bsize)
	return
}

var getCurIndexes GetCurIndexes = func(fPath string) (curKInd, curDirInd int, err error) {
	var f *os.File
	if f, err = os.Open(fPath); err != nil {
		return
	}

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		err = errors.New("current K Index and directory index missing")
		return
	}
	curKIndStr := scanner.Text()

	if !scanner.Scan() {
		err = errors.New("current Directory Index missing")
		return
	}
	curDirIndStr := scanner.Text()

	curKInd, err = strconv.Atoi(curKIndStr)
	if err != nil {
		return
	}

	curDirInd, err = strconv.Atoi(curDirIndStr)
	if err != nil {
		return
	}

	return
}

var getCurrentDirBlockNums GetCurrentDirBlockNums = func(dir string) (int, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	return len(dirEntries), nil
}

var updateCurIndexes UpdateCurIndexes = func(fPath string, curKInd, curDirInd int) error {
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(fmt.Sprintf("%v\n%v", curKInd, curDirInd)))

	return err
}

var countBlocksInVolumes CountBlocksInVolumes = func(vPath, dirPrefix string, dcl int) (uint64, uint64) {
	grandCount := &struct {
		totalBlocksSize uint64
		mu              sync.Mutex
	}{}

	var totalBlocksCount uint64
	guideChannel := make(chan struct{}, 50)
	wg := sync.WaitGroup{}

	for i := 0; i < dcl; i++ {
		subDirPath := filepath.Join(vPath, fmt.Sprintf("%v%v", dirPrefix, i))
		_, err := os.Stat(subDirPath)
		if err != nil {
			continue
		}

		for j := 0; j < dcl; j++ {
			blocksPath := filepath.Join(subDirPath, fmt.Sprint(j))
			_, err = os.Stat(subDirPath)
			if err != nil {
				continue
			}

			f, err := os.Open(blocksPath)
			if err != nil {
				continue
			}

			for {
				dirEntries, err := f.ReadDir(1000)
				if errors.Is(err, io.EOF) {
					err = nil
					break
				}

				totalBlocksCount += uint64(len(dirEntries))

				for _, dirEntry := range dirEntries {
					guideChannel <- struct{}{}
					wg.Add(1)

					go func(dE fs.DirEntry) {
						defer func() {
							<-guideChannel
							wg.Done()
						}()

						finfo, err := dE.Info()
						if err != nil {
							return
						}
						grandCount.mu.Lock()
						defer grandCount.mu.Unlock()

						grandCount.totalBlocksSize += uint64(finfo.Size())
					}(dirEntry)

				}
			}
		}
	}

	wg.Wait()

	return grandCount.totalBlocksSize, totalBlocksCount
}

// Converts integer and string representation of number to uint64. 10 * 10 * 10 is returned as uint64(1000); 10^4 is returned as uint64(10000)
var getUint64ValueFromYamlConfig GetUint64ValueFromYamlConfig = func(v interface{}) (uint64, error) {
	switch v.(type) {
	case int:
		return uint64(v.(int)), nil
	case string:
		vStr := v.(string)
		vStr = strings.ReplaceAll(vStr, " ", "")
		if strings.Contains(vStr, "^") {
			res := strings.Split(vStr, "^")
			r1, err := strconv.Atoi(res[0])
			if err != nil {
				return 0, err
			}

			r2, err := strconv.Atoi(res[1])
			if err != nil {
				return 0, err
			}

			n := math.Pow(float64(r1), float64(r2))
			return uint64(n), nil

		} else if strings.Contains(vStr, "*") {
			var value = uint64(1)
			res := strings.Split(vStr, "*")
			for _, r := range res {
				i, err := strconv.Atoi(r)
				if err != nil {
					return 0, err
				}

				value *= uint64(i)
			}
			return value, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("Type unsupported: %T", v))
}

func getVolumePathFromBlockPath(bPath string) string {
	bPath = filepath.Clean(bPath)
	splittedPaths := strings.Split(bPath, "/")

	/*
		Example bPath = /path/to/blocks/HK0/199/blockname.dat
		path returned --> /path/to/blocks
		bPath = /another/path/to/blocks/HK0/199/blockname.dat
		path returned --> /another/path/to/blocks
	*/
	return strings.Join(splittedPaths[:len(splittedPaths)-3], "/")
}
