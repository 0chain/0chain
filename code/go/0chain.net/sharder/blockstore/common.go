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
	fileExt            = ".dat.zlib"
	IndexStateFileName = "index.state"

	//DefaultStrategies
	DefaultVolumeStrategy = "random"
	DefaultColdStrategy   = "random"
	DefaultCacheStrategy  = "random"

	//BlockLimit Limit the number of blocks to store in any tier
	BlockLimitNumber = 10000000000 // 10^10
	MB               = 1024 * 1024
	GB               = 1024 * MB
)

//Common errors
var (
	ErrInodesLimit = func(vPath string, inodesToMaintain uint64) error {
		return fmt.Errorf("volume %v has inodes lesser than inodes to maintain, %v", vPath, inodesToMaintain)
	}

	ErrSizeLimit = func(vPath string, sizeToMaintain uint64) error {
		return fmt.Errorf("volume %v has size lesser than size to maintain, %vGB", vPath, sizeToMaintain)
	}

	ErrAllowedSizeLimit = func(vPath string, allowedSizeLimit uint64) error {
		return fmt.Errorf("allowed size limit, %v, for volume %v reached.", allowedSizeLimit, vPath)
	}

	ErrAllowedCountLimit = func(vPath string, allowedCountLimit uint64) error {
		return fmt.Errorf("allowed block number limit, %v, for volume %v reached.", allowedCountLimit, vPath)
	}

	ErrVolumeFull = func(volPath string) error {
		return fmt.Errorf("volume %v is full.", volPath)
	}

	ErrSelectDir = func(volPath string, err error) string {
		return fmt.Sprintf("error while selecting dir; volume path: %v, err: %v", volPath, err)
	}

	ErrStrategyNotSupported = func(strategy string) error {
		return fmt.Errorf("strategy %v is not supported", strategy)
	}

	ErrStorageTypeNotSupported = func(storageType string) error {
		return fmt.Errorf("storage type %v is not supported", storageType)
	}

	ErrCacheWritePolicyNotSupported = func(writePolicy string) error {
		return fmt.Errorf("cache write policy %v is not supported", writePolicy)
	}

	ErrFiftyPercent                = errors.New("at least 50%% volumes must be able to store blocks")
	ErrCacheStorageConfNotProvided = errors.New("storage type includes cache but cache config not provided")
	ErrDiskStorageConfNotProvided  = errors.New("storage type includes disk tier but disk config not provided")
	ErrColdStorageConfNotProvided  = errors.New("storage type includes cold tier but cold tier config not provided")
	ErrUnableToSelectVolume        = errors.New("unable to select any available volume")
	ErrUnableToSelectColdStorage   = errors.New("unable to select any available cold storage")
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

func getAvailableSizeAndInodes(vPath string) (availableSize, totalInodes, availableInodes uint64, err error) {
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

func getCurIndexes(fPath string) (curKInd, curDirInd int, err error) {
	var f *os.File
	if f, err = os.Open(fPath); err != nil {
		return
	}
	defer f.Close()

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

func getCurrentDirBlockNums(dir string) (int, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	return len(dirEntries), nil
}

func updateCurIndexes(fPath string, curKInd, curDirInd int) error {
	f, err := os.Create(fPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(fmt.Sprintf("%v\n%v", curKInd, curDirInd)))

	return err
}

func countBlocksInVolumes(vPath, dirPrefix string, dcl int) (uint64, uint64) {
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
			defer f.Close()

			for {
				dirEntries, err := f.ReadDir(1000)
				if errors.Is(err, io.EOF) {
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

// Converts integer and string representation of number to uint64.
// 10 * 10 * 10 is returned as uint64(1000); 10^4 is returned as uint64(10000)
func getUint64ValueFromYamlConfig(v interface{}) (uint64, error) {
	switch v := v.(type) {
	case int:
		return uint64(v), nil
	case string:
		vStr := strings.ReplaceAll(v, " ", "")
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
		} else {
			return 0, fmt.Errorf("could not convert %s to uint64", vStr)
		}

	}
	return 0, fmt.Errorf("type unsupported: %T", v)
}

// Converts integer and string representation of number to int.
// 10 * 10 * 10 is returned as int(1000); 10^4 is returned as int(10000)
func getintValueFromYamlConfig(v interface{}) (int, error) {
	switch v := v.(type) {
	case int:
		return v, nil
	case string:
		vStr := strings.ReplaceAll(v, " ", "")
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
			return int(n), nil

		} else if strings.Contains(vStr, "*") {
			var value = int(1)
			res := strings.Split(vStr, "*")
			for _, r := range res {
				i, err := strconv.Atoi(r)
				if err != nil {
					return 0, err
				}

				value *= i
			}
			return value, nil
		} else {
			return 0, fmt.Errorf("could not convert %s to int", vStr)
		}
	}
	return 0, fmt.Errorf("type unsupported: %T", v)
}

func getVolumePathFromBlockPath(bPath string) string {
	bPath = filepath.Clean(bPath)
	splittedPaths := strings.Split(bPath, "/")

	/*
		Example bPath = /path/to/blocks/blocks/HK0/199/blockname.dat
		path returned --> /path/to/blocks
		bPath = /another/path/to/blocks/blocks/HK0/199/blockname.dat
		path returned --> /another/path/to/blocks
	*/
	return strings.Join(splittedPaths[:len(splittedPaths)-4], "/")
}

type Mutex chan struct{}

func (mu Mutex) Lock() {
	mu <- struct{}{}
}

func (mu Mutex) Unlock() {
	select {
	case <-mu:
	default:
		panic("trying to unlock unlocked lock")
	}
}

func (mu Mutex) TryLock() bool {
	select {
	case mu <- struct{}{}:
		return true
	default:
		return false
	}
}
