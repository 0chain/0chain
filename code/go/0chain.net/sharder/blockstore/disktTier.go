package blockstore

import (
	"bufio"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/viper"
	"golang.org/x/sys/unix"
)

// var dTier diskTier

const (
	//Contains 2000 directories that contains 2000 blocks each, so one twoKilo directory contains 4*10^6blocks.
	//So 2000 such twokilo directories will contain 8*10^9 blocks
	HK = "HK"
	//Hot directory content limit
	HDCL = 2000

	//Contains 1000 directories that contains 1000 blocks each, so one kilo directories contains 10^9 blocks
	WK = "WK"
	//Warm directory content limit
	WDCL = 1000
)

type selectedDiskVolume struct {
	volume  *volume
	prevInd int
	err     error
}

type diskTier struct {
	Volumes          []*volume //List of hot volumes
	SelectNextVolume func(volumes []*volume, prevInd int)
	SelectedVolumeCh <-chan selectedDiskVolume //volume that will be used to store blocks next
	PrevVolInd       int
	// Mu               sync.Mutex
	//Directory content limit
	DCL       int
	DirPrefix string
}

func (d *diskTier) write(b *block.Block, data []byte) (blockPath string, err error) {
	Logger.Info("Waiting channel for selected volume")
	sdv := <-d.SelectedVolumeCh
	if sdv.err != nil {
		return "", sdv.err
	}

	d.PrevVolInd = sdv.prevInd

	if blockPath, err = sdv.volume.write(b, data, d); err != nil {
		return
	}

	go d.SelectNextVolume(d.Volumes, d.PrevVolInd)

	return
}

type volume struct {
	Path string

	AllowedBlockNumbers uint64
	AllowedBlockSize    uint64

	SizeToMaintain   uint64
	InodesToMaintain uint64
	BlocksSize       uint64
	BlocksCount      uint64

	//used in selecting directory
	CurKInd         int
	CurDirInd       int
	CurDirBlockNums int
	//Lock required when updadign blocks size, count.
	Mu sync.Mutex
}

func (v *volume) selectDir(dTier *diskTier) error {
	if v.CurDirBlockNums < dTier.DCL-1 {
		blocksPath := filepath.Join(v.Path, fmt.Sprintf("%v%v/%v", dTier.DirPrefix, v.CurKInd, v.CurDirInd))
		_, err := os.Stat(blocksPath)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(blocksPath, 0644); err != nil {
				return err
			}
		}
		return nil
	}

	if v.CurDirInd < dTier.DCL-1 {
		dirInd := v.CurDirInd + 1
		blocksPath := filepath.Join(v.Path, fmt.Sprintf("%v%v/%v", dTier.DirPrefix, v.CurKInd, dirInd))
		blocksCount, err := countFiles(blocksPath)

		if err != nil && errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(blocksPath, 0644)
			if err != nil {
				return err
			}
			v.CurDirInd = dirInd
			v.CurDirBlockNums = 0
		} else if err != nil {
			return err
		}

		if blocksCount >= dTier.DCL {
			return ErrVolumeFull(v.Path)
		}

		v.CurDirInd = dirInd
		v.CurDirBlockNums = blocksCount

		return updateCurIndexes(filepath.Join(v.Path, IndexStateFileName), v.CurKInd, v.CurDirInd, v.CurDirBlockNums)
	}

	var kInd int
	if v.CurKInd < dTier.DCL-1 {
		kInd = v.CurKInd + 1
	} else {
		kInd = 0
	}

	dirInd := 0
	blocksPath := filepath.Join(v.Path, fmt.Sprintf("%v%v/%v", dTier.DirPrefix, kInd, dirInd))
	blocksCount, err := countFiles(blocksPath)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(blocksPath, 0644)
		if err != nil {
			return err
		}
		v.CurDirInd = dirInd
		v.CurDirBlockNums = 0

		return updateCurIndexes(filepath.Join(v.Path, IndexStateFileName), v.CurKInd, v.CurDirInd, v.CurDirBlockNums)
	} else if err != nil {
		return err
	}

	if blocksCount >= dTier.DCL {
		return ErrVolumeFull(v.Path)
	}

	v.CurKInd = kInd
	v.CurDirInd = dirInd
	v.CurDirBlockNums = blocksCount

	return updateCurIndexes(filepath.Join(v.Path, IndexStateFileName), v.CurKInd, v.CurDirInd, v.CurDirBlockNums)
}

func (v *volume) write(b *block.Block, data []byte, dTier *diskTier) (bPath string, err error) {
	bPath = path.Join(v.Path, fmt.Sprintf("%v%v/%v", dTier.DirPrefix, v.CurKInd, v.CurDirInd), fmt.Sprintf("%v.%v", b.Hash, fileExt))

	var f *os.File
	f, err = os.Create(bPath)
	if err != nil {
		return
	}

	bf := bufio.NewWriterSize(f, 64*1024)
	volumeWriter, err := zlib.NewWriterLevel(f, zlib.BestCompression)
	if err != nil {
		return
	}

	var n int
	n, err = volumeWriter.Write(data)
	if err != nil {
		volumeWriter.Close()
		os.Remove(bPath)
		return
	}

	if err = volumeWriter.Close(); err != nil {
		f.Close()
		os.Remove(bPath)
		return
	}
	if err = bf.Flush(); err != nil {
		f.Close()
		os.Remove(bPath)
		return
	}
	if err = f.Close(); err != nil {
		os.Remove(bPath)
		return
	}

	v.CurDirBlockNums++
	v.updateCount(1)
	v.updateSize(int64(n))
	return
}

func (v *volume) read(hash, blockPath string) (*block.Block, error) {
	f, err := os.Open(blockPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var b *block.Block
	err = datastore.ReadJSON(r, b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//When a block is moved to cold tier delete function will be called
func (v *volume) delete(hash, path string) error {
	finfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	size := finfo.Size()

	err = os.Remove(path)
	if err != nil {
		return err
	}

	v.updateCount(-1)
	v.updateSize(-size)
	return nil
}

func (v *volume) updateSize(n int64) {
	v.Mu.Lock()
	defer v.Mu.Unlock()
	if n < 0 {
		v.BlocksSize -= uint64(n)
	} else {
		v.BlocksSize += uint64(n)
	}
}

func (v *volume) updateCount(n int64) {
	v.Mu.Lock()
	defer v.Mu.Unlock()
	if n < 0 {
		v.BlocksCount -= uint64(n)
	} else {
		v.BlocksCount += uint64(n)
	}
}

func (v *volume) isAbleToStoreBlock(dTier *diskTier) (ableToStore bool) {
	var volStat unix.Statfs_t
	err := unix.Statfs(v.Path, &volStat)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	if v.AllowedBlockSize != 0 && v.BlocksSize >= v.AllowedBlockSize {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block size. Allowed: %v, Total block size: %v", v.AllowedBlockSize, v.BlocksSize))
		return
	}

	if v.AllowedBlockNumbers != 0 && v.BlocksCount >= v.AllowedBlockNumbers {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block numbers. Allowed: %v, Total block size: %v", v.AllowedBlockNumbers, v.BlocksCount))
		return
	}

	if v.InodesToMaintain != 0 && volStat.Ffree <= v.InodesToMaintain {
		Logger.Error(fmt.Sprintf("Available Inodes for volume %v is less than inodes to maintain(%v)", v.Path, v.InodesToMaintain))
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if v.SizeToMaintain != 0 && availableSize/(1024*1024*1024) < uint64(v.SizeToMaintain) {
		Logger.Error(fmt.Sprintf("Available size for volume %v is less than size to maintain(%v)", v.Path, v.SizeToMaintain))
		return
	}

	if unix.Access(v.Path, unix.W_OK) != nil {
		return
	}

	if err := v.selectDir(dTier); err != nil {
		Logger.Error(ErrSelectDir(v.Path, err))
		return
	}

	return true
}

func volumeInit(tierType string, vViper *viper.Viper, mode string) *diskTier {
	strategy := vViper.GetString("strategy")
	if strategy == "" {
		strategy = DefaultVolumeStrategy
	}

	volumesI := vViper.Get("volumes")
	if volumesI == nil {
		panic(errors.New("Volumes config not available"))

	}

	volumesMapI := volumesI.([]interface{})
	var volumesMap []map[string]interface{}
	for _, volumeI := range volumesMapI {
		m := make(map[string]interface{})
		volIMap := volumeI.(map[interface{}]interface{})
		for k, v := range volIMap {
			sK := k.(string)
			m[sK] = v
		}

		volumesMap = append(volumesMap, m)
	}

	var dTier diskTier
	if tierType == "hot" {
		dTier = diskTier{
			DCL:       HDCL,
			DirPrefix: HK,
		}
	} else {
		dTier = diskTier{
			DCL:       WDCL,
			DirPrefix: WK,
		}
	}

	Logger.Info(fmt.Sprintf("Initializing volumes in %v mode", mode))
	switch mode {
	case "start":
		//Delete all existing data and start fresh
		startVolumes(volumesMap, &dTier) //will panic if right config setup is not provided
	case "restart": //Nothing is lost but sharder was off for maintenance mode
		restartVolumes(volumesMap, &dTier)
	case "recover": //Metadata is lost
		recoverVolumeMetaData(volumesMap, &dTier)
	case "repair": //Metadata is present but some disk failed
		panic("Repair mode not implemented")
	case "repair_and_recover": //Metadata is lost and some disk failed
		panic("Repair and recover mode not implemented")
	default:
		panic(fmt.Errorf("%v mode is not supported", mode))
	}

	Logger.Info(fmt.Sprintf("Successfully ran volumeInit in %v mode", mode))

	Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))
	var f func(volumes []*volume, prevInd int)
	diskVolumeSelectChan := make(chan selectedDiskVolume, 1)

	switch strategy {
	default:
		panic(fmt.Errorf("Strategy %v is not supported", strategy))
	case "random":
		f = func(volumes []*volume, prevInd int) {
			var selectedVolume *volume
			var selectedIndex int

			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			for len(volumes) > 0 {
				ind := r.Intn(len(volumes))
				sv := volumes[ind]

				if sv.isAbleToStoreBlock(&dTier) {
					selectedIndex = ind
					selectedVolume = sv
					break
				}

				volumes[ind] = volumes[len(volumes)-1]
				volumes = volumes[:len(volumes)-1]
			}

			dTier.Volumes = volumes

			if selectedVolume == nil {
				diskVolumeSelectChan <- selectedDiskVolume{
					err: ErrUnableToSelectVolume,
				}
			} else {
				diskVolumeSelectChan <- selectedDiskVolume{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}
		}
	case "round_robin":
		f = func(volumes []*volume, prevInd int) {
			var selectedVolume *volume
			prevVolume := volumes[prevInd]
			var selectedIndex int

			if prevInd < 0 {
				prevInd = -1
			}

			for i := prevInd + 1; i != prevInd; i++ {
				if len(volumes) == 0 {
					break
				}

				if i >= len(volumes) {
					i = len(volumes) - i
				}
				if i < 0 {
					i = 0
				}

				v := volumes[i]
				if v.isAbleToStoreBlock(&dTier) {
					selectedVolume = v
					selectedIndex = i

					break
				} else {
					volumes = append(volumes[:i], volumes[i+1:]...)

					if i < prevInd {
						prevInd--
					}

					i--
				}

			}

			if selectedVolume == nil {
				if prevVolume.isAbleToStoreBlock(&dTier) {
					selectedVolume = prevVolume
					selectedIndex = 0
				}
			}

			dTier.Volumes = volumes

			if selectedVolume == nil {
				diskVolumeSelectChan <- selectedDiskVolume{
					err: ErrUnableToSelectVolume,
				}
			} else {
				diskVolumeSelectChan <- selectedDiskVolume{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}
		}
	case "min_size_first":
		f = func(volumes []*volume, prevInd int) {
			var selectedVolume *volume
			var selectedIndex int

			totalVolumes := len(volumes)
			for i := 0; i < totalVolumes; i++ {
				if len(volumes) == 0 {
					break
				}

				v := volumes[i]
				if !v.isAbleToStoreBlock(&dTier) {
					volumes = append(volumes[:i], volumes[i+1:]...)
					i--
					totalVolumes--
					continue
				}

				if selectedVolume == nil {
					selectedVolume = v
					selectedIndex = i
					continue
				}

				if v.BlocksSize < selectedVolume.BlocksSize {
					selectedVolume = v
					selectedIndex = i
				}
			}

			dTier.Volumes = volumes

			if selectedVolume == nil {
				diskVolumeSelectChan <- selectedDiskVolume{
					err: ErrUnableToSelectVolume,
				}
			} else {
				diskVolumeSelectChan <- selectedDiskVolume{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}
		}
	case "min_count_first":
		f = func(volumes []*volume, prevInd int) {
			var selectedVolume *volume
			var selectedIndex int

			totalVolumes := len(volumes)
			for i := 0; i < totalVolumes; i++ {
				if len(volumes) == 0 {
					break
				}

				v := volumes[i]
				if !v.isAbleToStoreBlock(&dTier) {
					volumes = append(volumes[:i], volumes[i+1:]...)
					i--
					totalVolumes--
					continue
				}

				if selectedVolume == nil {
					selectedVolume = v
					selectedIndex = i
				}

				if v.BlocksCount < selectedVolume.BlocksCount {
					selectedVolume = v
					selectedIndex = i
				}
			}

			dTier.Volumes = volumes

			if selectedVolume == nil {
				diskVolumeSelectChan <- selectedDiskVolume{
					err: ErrUnableToSelectVolume,
				}
			} else {
				diskVolumeSelectChan <- selectedDiskVolume{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}

		}
	}

	dTier.SelectedVolumeCh = diskVolumeSelectChan
	dTier.SelectNextVolume = f

	go dTier.SelectNextVolume(dTier.Volumes, dTier.PrevVolInd)

	return &dTier
}

func startVolumes(volumes []map[string]interface{}, dTier *diskTier) {
	startvolumes(volumes, true, dTier)
}

func restartVolumes(volumes []map[string]interface{}, dTier *diskTier) {
	startvolumes(volumes, false, dTier)
}

func startvolumes(mVolumes []map[string]interface{}, shouldDelete bool, dTier *diskTier) {
	//Remove db
	//Remove all the blocks

	for _, volI := range mVolumes {
		vPathI, ok := volI["path"]
		if !ok {
			Logger.Error("Discarding volume; Path field is required")
			continue
		}

		vPath := vPathI.(string)

		var curDirInd, curKInd, curDirBlockNums int
		var totalBlocksCount, totalBlocksSize uint64
		var err error
		if shouldDelete {
			if err := os.RemoveAll(vPath); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err := os.MkdirAll(vPath, 0644); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err := updateCurIndexes(filepath.Join(vPath, IndexStateFileName), 0, 0, 0); err != nil {
				Logger.Error(err.Error())
				continue
			}
		} else {
			curKInd, curDirInd, curDirBlockNums, err = getCurIndexes(filepath.Join(vPath, "index.state"))
			if err != nil {
				Logger.Error(err.Error())
				continue
			}

			totalBlocksCount, totalBlocksSize = countBlocksInVolumes(vPath, dTier.DirPrefix, dTier.DCL)
		}

		availableSize, totalInodes, availableInodes, err := getAvailableSizeAndInodes(vPath)

		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := volI["size_to_maintain"]
		if ok {
			sizeToMaintain, err = getUint64ValueFromYamlConfig(sizeToMaintainI) // try to convert it to uint64 directly from yaml parser(viper)
			if err != nil {
				panic(err)
			}
		}

		if availableSize/(1024^3) <= sizeToMaintain {
			Logger.Error(ErrSizeLimit(vPath, sizeToMaintain).Error())
			continue
		}

		var inodesToMaintain uint64
		inodesToMaintainI, ok := volI["inodes_to_maintain"]
		if ok {
			inodesToMaintain, err = getUint64ValueFromYamlConfig(inodesToMaintainI) // try to convert it to uint64 directly from yaml parser(viper)
			if err != nil {
				panic(err)
			}
		}
		if float64(100*availableInodes)/float64(totalInodes) <= float64(inodesToMaintain) {
			Logger.Error(ErrInodesLimit(vPath, inodesToMaintain).Error())
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := volI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers, err = getUint64ValueFromYamlConfig(allowedBlockNumbersI) // try to convert it to uint64 directly from yaml parser(viper)
			if err != nil {
				panic(err)
			}
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := volI["allowed_block_size"]
		if ok {
			allowedBlockSize, err = getUint64ValueFromYamlConfig(allowedBlockSizeI) // try to convert it to uint64 directly from yaml parser(viper)
			if err != nil {
				panic(err)
			}
		}

		//Create index state which stores curDirBlockNums, curDir index and curKIndex

		dTier.Volumes = append(dTier.Volumes, &volume{
			Path:                vPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			BlocksSize:          totalBlocksSize,
			BlocksCount:         totalBlocksCount,
			SizeToMaintain:      sizeToMaintain,
			CurKInd:             curKInd,
			CurDirInd:           curDirInd,
			CurDirBlockNums:     curDirBlockNums,
		})
	}

	if len(dTier.Volumes) < len(mVolumes)/2 {
		panic(ErrFiftyPercent)
	}
}

//This function will recover metadata
func recoverVolumeMetaData(mVolumes []map[string]interface{}, dTier *diskTier) {
	for _, mVolume := range mVolumes {
		volPathI, ok := mVolume["path"]
		if !ok {
			panic("Every volume path is required for recovering metadata")
		}

		volPath := volPathI.(string)
		Logger.Info(fmt.Sprintf("Recovering metadata from volume: %v", volPath))

		recoverWG := sync.WaitGroup{}
		guideChannel := make(chan struct{}, 10)

		grandCount := struct {
			totalBlocksCount uint64
			recoveredCount   uint64
			totalBlocksSize  uint64
			mu               sync.Mutex
		}{}

		var shouldRecover bool
		recoveryI, ok := mVolume["recovery"]
		if ok {
			shouldRecover = recoveryI.(bool)
		}

		if shouldRecover {

			for i := 0; i < dTier.DCL; i++ {
				volIndexPath := filepath.Join(volPath, fmt.Sprintf("%v%v", dTier.DirPrefix, i))
				if _, err := os.Stat(volIndexPath); err != nil {
					Logger.Debug(fmt.Sprintf("Error while recovering metadata for index %v; Full path: %v; err: %v", i, volIndexPath, err))
					continue
				}

				for j := 0; j < dTier.DCL; j++ {
					blockSubDirPath := filepath.Join(volIndexPath, fmt.Sprintf("%v", j))
					if _, err := os.Stat(blockSubDirPath); err != nil {
						Logger.Debug(err.Error())
						continue
					}

					guideChannel <- struct{}{}
					recoverWG.Add(1)

					//TODO which is better? To use go routines for multi disk operations on single disk or for multi disk operations
					//for multi disks? Need some benchmark
					go func(gPath string) { //gPath Path for goroutine
						defer recoverWG.Done()
						defer func() {
							<-guideChannel
						}()

						var recoverCount, totalBlocksCount int

						var f *os.File
						f, _ = os.Open(gPath)
						defer f.Close()

						var dirEntries []os.DirEntry
						var err error
						for {
							dirEntries, err = f.ReadDir(1000)
							if errors.Is(err, io.EOF) {
								err = nil
								break
							}
							for _, dirEntry := range dirEntries {
								var bwr BlockWhereRecord
								var ubr UnmovedBlockRecord
								var errorOccurred bool
								var blockSize uint64
								fileName := dirEntry.Name()
								hash := strings.Split(fileName, ".")[0]
								blockPath := filepath.Join(gPath, fileName)

								finfo, err := dirEntry.Info()
								if err != nil {
									Logger.Error(fmt.Sprintf("Error: %v while getting file info for file: %v", err, blockPath))
									errorOccurred = true
									goto CountUpdate
								}

								blockSize = uint64(finfo.Size())
								bwr = BlockWhereRecord{
									Hash:      hash,
									Tiering:   HotTier,
									BlockPath: blockPath,
								}

								ubr = UnmovedBlockRecord{
									Hash:      hash,
									CreatedAt: finfo.ModTime(),
								}

								if err, uErr := bwr.AddOrUpdate(), ubr.Add(); !(err == nil && uErr == nil) {
									Logger.Error(fmt.Sprintf("BwrError: %v, UbrError: %v, while adding metadata for file: %v", err, uErr, blockPath))
									errorOccurred = true
									goto CountUpdate
								} else {
									continue
								}

							CountUpdate:
								totalBlocksCount++
								grandCount.mu.Lock()
								grandCount.totalBlocksCount++
								grandCount.totalBlocksSize += blockSize
								if !errorOccurred {
									grandCount.recoveredCount++
								}
								grandCount.mu.Unlock()
								recoverCount++
							}
						}
						Logger.Info(fmt.Sprintf("%v Meta records recovered of %v blocks from path: %v", recoverCount, totalBlocksCount, gPath))

					}(blockSubDirPath)

				}
			}
			recoverWG.Wait() //wait for all goroutine to complete
			Logger.Info("Completed meta data recovery")

		} else {
			if err := os.RemoveAll(volPath); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err := os.MkdirAll(volPath, 0644); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err := updateCurIndexes(filepath.Join(volPath, IndexStateFileName), 0, 0, 0); err != nil {
				Logger.Error(err.Error())
				continue
			}
		}
		//Check available size and inodes and add volume to volume pool
		availableSize, totalInodes, availableInodes, err := getAvailableSizeAndInodes(volPath)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		/*
			curKInd, curDirInd and curBlockNums are important parameters while selecting Directory to write new blocks.
			The new block path is always; fmt.Sprintf("%v%v/%v", "WK", curKInd, curDirInd) when next volume is selected.
			If curBlockNums exceeds some number then the directory is skipped and jumped to next directory and if that next directory is full then volume raises error; ErrVolumeFull

			So above parameters are like the state of volume.
			Also since the blocks is regularly moved if cold Tiering is enabled it will be difficult to know the indexes so each time a volume is selected its state is written into
			"index.state" file inside volumePath.

			If index file is lost then one can put blocks in any order and update the index file but the directory should be of above format and limit should be maintained.

			For recovered metadata unmoved blocks creation time will be the file creation time(As for linux; last modidification time).
		*/
		curKInd, curDirInd, curBlockNums, err := getCurIndexes(filepath.Join(volPath, IndexStateFileName))
		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := mVolume["size_to_maintain"]
		if ok {
			sizeToMaintain, err = getUint64ValueFromYamlConfig(sizeToMaintainI)
			if err != nil {
				panic(err)
			}
		}

		if availableSize/(1024^3) <= sizeToMaintain {
			Logger.Error(ErrSizeLimit(volPath, sizeToMaintain).Error())
			continue
		}

		var inodesToMaintain uint64
		inodesToMaintainI, ok := mVolume["inodes_to_maintain"]
		if ok {
			inodesToMaintain, err = getUint64ValueFromYamlConfig(inodesToMaintainI)
			if err != nil {
				panic(err)
			}
		}
		if float64(100*availableInodes)/float64(totalInodes) <= float64(inodesToMaintain) {
			Logger.Error(ErrInodesLimit(volPath, inodesToMaintain).Error())
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := mVolume["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers, err = getUint64ValueFromYamlConfig(allowedBlockNumbersI)
			if err != nil {
				panic(err)
			}
		}

		if allowedBlockNumbers != 0 && grandCount.totalBlocksCount > allowedBlockNumbers {
			Logger.Error(ErrAllowedCountLimit(volPath, allowedBlockNumbers).Error())
			continue
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := mVolume["allowed_block_size"]
		if ok {
			allowedBlockSize, err = getUint64ValueFromYamlConfig(allowedBlockSizeI)
			if err != nil {
				panic(err)
			}
		}

		if allowedBlockSize != 0 && grandCount.totalBlocksSize > allowedBlockSize {
			Logger.Error(ErrAllowedSizeLimit(volPath, allowedBlockSize).Error())
			continue
		}

		dTier.Volumes = append(dTier.Volumes, &volume{
			Path:                volPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			SizeToMaintain:      sizeToMaintain,
			BlocksCount:         uint64(grandCount.totalBlocksCount),
			CurKInd:             curKInd,
			CurDirInd:           curDirInd,
			CurDirBlockNums:     curBlockNums,
		})
	}

	if len(dTier.Volumes) < len(mVolumes)/2 {
		panic(ErrFiftyPercent)
	}
}
