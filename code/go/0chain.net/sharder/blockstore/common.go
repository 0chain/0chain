package blockstore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	fileExt            = ".dat"
	IndexStateFileName = "index.state"

	//strategies
	Random        = "random"
	RoundRobin    = "round_robin"
	MinSizeFirst  = "min_size_first"
	MinCountFirst = "min_count_first"

	//DefaultStrategies
	DefaultVolumeStrategy = "random"
	DefaultColdStrategy   = "random"
	DefaultCacheStrategy  = "random"
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
	ErrUnableToSelectVolume        = errors.New("Unable to select any available volume")
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

func getCurIndexes(fPath string) (curKInd, curDirInd, curBlockNums int, err error) {
	var f *os.File
	if f, err = os.Open(fPath); err != nil {
		return
	}

	scanner := bufio.NewScanner(f)
	curKIndStr := scanner.Text()
	more := scanner.Scan()
	if more == false {
		err = errors.New("Current Directory Index missing")
		return
	}
	curDirIndStr := scanner.Text()
	more = scanner.Scan()
	if more == false {
		err = errors.New("Current Directory Block numbers missing")
		return
	}
	curBlockNumsStr := scanner.Text()

	curKInd, err = strconv.Atoi(curKIndStr)
	if err != nil {
		return
	}

	curDirInd, err = strconv.Atoi(curDirIndStr)
	if err != nil {
		return
	}

	curBlockNums, err = strconv.Atoi(curBlockNumsStr)
	if err != nil {
		return
	}

	return
}

func updateCurIndexes(fPath string, curKInd, curDirInd, curBlockNums int) error {
	f, err := os.Create(fPath)
	if err != nil {
		return nil
	}

	_, err = f.Write([]byte(fmt.Sprintf("%v\n%v\n%v", curDirInd, curDirInd, curBlockNums)))

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
