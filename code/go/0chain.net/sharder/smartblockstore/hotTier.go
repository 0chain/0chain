package smartblockstore

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
	"golang.org/x/sys/unix"
)

var hotTier hTier

const (
	//Contains 2000 directories that contains 2000 blocks each, so one twoKilo directory contains 4*10^6blocks.
	//So 2000 such twokilo directories will contain 8*10^9 blocks
	HK = "HK"
	//Hot directory content limit
	HDCL = 2000
)

type hTier struct { //Hot Tier
	volumes          []hotVolume //List of hot volumes
	selectNextVolume func(hotVolumes []hotVolume, prevInd int) (*hotVolume, int)
	volume           *hotVolume //volume that will be used to store blocks next
	prevVolInd       int
	mu               sync.Mutex
}

type hotVolume struct {
	path string

	allowedBlockNumbers uint64
	allowedBlockSize    uint64

	sizeToMaintain   uint64
	inodesToMaintain uint64
	blocksSize       uint64
	blocksCount      uint64

	//used in selecting directory
	curHKInd        uint32
	curDirInd       uint32
	curDirBlockNums uint32
}

func (hv *hotVolume) selectDir() error {
	if hv.curDirBlockNums < HDCL-1 {
		return nil
	}
	if hv.curDirInd < HDCL-1 {
		dirInd := hv.curDirInd + 1
		blocksPath := filepath.Join(hv.path, fmt.Sprintf("%v%v/%v", HK, hv.curHKInd, dirInd))
		blocksCount, err := countFiles(blocksPath)

		if err != nil && errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(blocksPath, 0644)
			if err != nil {
				return err
			}
			hv.curDirInd = dirInd
			hv.curDirBlockNums = 0
		} else if err != nil {
			return err
		}

		if blocksCount >= HDCL {
			return ErrVolumeFull(hv.path)
		}

		hv.curDirInd = dirInd
		hv.curDirBlockNums = uint32(blocksCount)
		return nil
	}

	var hkInd uint32
	if hv.curHKInd < HDCL-1 {
		hkInd = hv.curHKInd + 1
	} else {
		hkInd = 0
	}

	dirInd := uint32(0)
	blocksPath := filepath.Join(hv.path, fmt.Sprintf("%v%v/%v", HK, hkInd, dirInd))
	blocksCount, err := countFiles(blocksPath)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(blocksPath, 0644)
		if err != nil {
			return err
		}
		hv.curDirInd = dirInd
		hv.curDirBlockNums = 0
		return nil
	} else if err != nil {
		return err
	}

	if blocksCount >= HDCL {
		return ErrVolumeFull(hv.path)
	}

	hv.curHKInd = hkInd
	hv.curDirInd = dirInd
	hv.curDirBlockNums = uint32(blocksCount)
	return nil
}

func (hv *hotVolume) write(b *block.Block, data []byte) (bPath string, err error) {
	bPath = path.Join(hv.path, fmt.Sprintf("K%v/%v", hv.curHKInd, hv.curDirInd), fmt.Sprintf("%v.%v", b.Hash, fileExt))

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

	//This block doesn't belong here
	wg := sync.WaitGroup{}
	wg.Add(2)

	var bwrErr, ubErr error
	var bwr BlockWhereRecord
	var ub UnmovedBlockRecord
	go func() {
		defer wg.Done()
		bwr = BlockWhereRecord{
			Hash:      b.Hash,
			Tiering:   HotTier,
			BlockPath: bPath,
		}

		bwrErr = bwr.AddOrUpdate()

	}()

	go func() {
		defer wg.Done()
		ub = UnmovedBlockRecord{
			CreatedAt: b.ToTime(),
			Hash:      b.Hash,
		}

		ubErr = ub.Add()
	}()

	wg.Wait()

	if bwrErr != nil || ubErr != nil {
		Logger.Error(err.Error())
		Logger.Info(fmt.Sprintf("Removing block: %v and its meta record", bPath))

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			defer wg.Done()
			DeleteBlockWhereRecord(b.Hash)
		}()

		go func() {
			defer wg.Done()
			ub.Delete()
		}()
		wg.Wait()

		os.Remove(bPath)
		return
	}
	//Above block doesn't belong here
	hv.curDirBlockNums++
	hv.updateCount(1)
	hv.updateSize(int64(n))
	return
}

func (hv *hotVolume) read(hash, blockPath string) (*block.Block, error) {
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
func (hv *hotVolume) delete(hash, path string) error {
	finfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	size := finfo.Size()

	err = os.Remove(path)
	if err != nil {
		return err
	}

	hv.updateCount(-1)
	hv.updateSize(-size)
	return nil
}

func (hv *hotVolume) updateSize(n int64) {
	if n < 0 {
		hv.blocksSize -= uint64(n)
	} else {
		hv.blocksSize += uint64(n)
	}
}

func (hv *hotVolume) updateCount(n int64) {
	if n < 0 {
		hv.blocksCount -= uint64(n)
	} else {
		hv.blocksCount += uint64(n)
	}
}

func (hv *hotVolume) isAbleToStoreBlock() (ableToStore bool) {
	var volStat unix.Statfs_t
	err := unix.Statfs(hv.path, &volStat)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	if hv.blocksSize >= hv.allowedBlockSize {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block size. Allowed: %v, Total block size: %v", hv.allowedBlockSize, hv.blocksSize))
		return
	}

	if hv.blocksCount >= hv.allowedBlockNumbers {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block numbers. Allowed: %v, Total block size: %v", hv.allowedBlockNumbers, hv.blocksCount))
		return
	}

	if volStat.Ffree < hv.inodesToMaintain {
		Logger.Error(fmt.Sprintf("Available Inodes for volume %v is less than inodes to maintain(%v)", hv.path, hv.inodesToMaintain))
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if availableSize/(1024*1024*1024) < uint64(hv.sizeToMaintain) {
		Logger.Error(fmt.Sprintf("Available size for volume %v is less than size to maintain(%v)", hv.path, hv.sizeToMaintain))
		return
	}

	if unix.Access(hv.path, unix.W_OK) != nil {
		return
	}

	if err := hv.selectDir(); err != nil {
		Logger.Error(ErrSelectDir(hv.path, err))
		return
	}

	return true
}

func hotInit(hConf map[string]interface{}, mode string) {
	volumesI, ok := hConf["volumes"]
	if !ok {
		panic(errors.New("Volumes config not available"))
	}

	volumes := volumesI.([]map[string]interface{})

	var strategy string
	strategyI, ok := hConf["strategy"]
	if !ok {
		strategy = DefaultHotStrategy
	} else {
		strategy = strategyI.(string)
	}

	Logger.Info(fmt.Sprintf("Running hotInit in %v mode", mode))
	switch mode {
	case "start":
		//Delete all existing data and start fresh
		startHotVolumes(volumes) //will panic if right config setup is not provided
	case "restart": //Nothing is lost but sharder was off for maintenance move
		restartHotVolumes(volumes)
		//
	case "recover": //Metadata is lost
		recoverHotMetaData(volumes)
	case "repair": //Metadata is present but some disk failed
		panic("Repair mode not implemented")
	case "repair_and_recover": //Metadata is lost and some disk failed
		panic("Repair and recover mode not implemented")
	default:
		panic(fmt.Errorf("%v mode is not supported", mode))
	}
	Logger.Info(fmt.Sprintf("Successfully ran hotInit in %v mode", mode))

	Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))
	var f func(hotVolumes []hotVolume, prevInd int) (*hotVolume, int)

	switch strategy {
	default:
		panic(fmt.Errorf("Strategy %v is not supported", strategy))
	case "random":
		f = func(hotVolumes []hotVolume, prevInd int) (*hotVolume, int) {
			var selectedVolume *hotVolume
			var selectedIndex int

			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			for len(hotVolumes) > 0 {
				ind := r.Intn(len(hotVolumes))
				selectedVolume = &hotVolumes[ind]

				if selectedVolume.isAbleToStoreBlock() {
					selectedIndex = ind
					break
				}

				hotVolumes[ind] = hotVolumes[len(hotVolumes)-1]
				hotVolumes = hotVolumes[:len(hotVolumes)-1]
			}

			hotTier.volumes = hotVolumes
			return selectedVolume, selectedIndex
		}
	case "round_robin":
		f = func(hotVolumes []hotVolume, prevInd int) (*hotVolume, int) {
			var selectedVolume *hotVolume
			prevVolume := hotVolumes[prevInd]
			var selectedIndex int

			if prevInd < 0 {
				prevInd = -1
			}

			for i := prevInd + 1; i != prevInd; i++ {
				if len(hotVolumes) == 0 {
					break
				}

				if i >= len(hotVolumes) {
					i = len(hotVolumes) - i
				}
				if i < 0 {
					i = 0
				}

				v := hotVolumes[i]
				if v.isAbleToStoreBlock() {
					selectedVolume = &v
					selectedIndex = i

					break
				} else {
					hotVolumes = append(hotVolumes[:i], hotVolumes[i+1:]...)

					if i < prevInd {
						prevInd--
					}

					i--
				}

			}

			if selectedVolume == nil {
				if prevVolume.isAbleToStoreBlock() {
					selectedVolume = &prevVolume
					selectedIndex = 0
				}
			}

			hotTier.volumes = hotVolumes
			return selectedVolume, selectedIndex
		}
	case "min_size_first":
		f = func(hotVolumes []hotVolume, prevInd int) (*hotVolume, int) {
			var selectedVolume *hotVolume
			var selectedIndex int

			totalVolumes := len(hotVolumes)
			for i := 0; i < totalVolumes; i++ {
				if len(hotVolumes) == 0 {
					break
				}

				v := hotVolumes[i]
				if !v.isAbleToStoreBlock() {
					hotVolumes = append(hotVolumes[:i], hotVolumes[i+1:]...)
					i--
					totalVolumes--
					continue
				}

				if selectedVolume == nil {
					selectedVolume = &v
					selectedIndex = i
					continue
				}

				if v.blocksSize < selectedVolume.blocksSize {
					selectedVolume = &v
					selectedIndex = i
				}
			}

			hotTier.volumes = hotVolumes
			return selectedVolume, selectedIndex
		}
	case "min_count_first":
		f = func(hotVolumes []hotVolume, prevInd int) (*hotVolume, int) {
			var selectedVolume *hotVolume
			var selectedIndex int

			totalVolumes := len(hotVolumes)
			for i := 0; i < totalVolumes; i++ {
				if len(hotVolumes) == 0 {
					break
				}

				v := hotVolumes[i]
				if !v.isAbleToStoreBlock() {
					hotVolumes = append(hotVolumes[:i], hotVolumes[i+1:]...)
					i--
					totalVolumes--
					continue
				}

				if selectedVolume == nil {
					selectedVolume = &v
					selectedIndex = i
				}

				if v.blocksCount < selectedVolume.blocksCount {
					selectedVolume = &v
					selectedIndex = i
				}
			}

			hotTier.volumes = hotVolumes
			return selectedVolume, selectedIndex
		}
	}

	hotTier.selectNextVolume = f
}

func startHotVolumes(volumes []map[string]interface{}) {
	startVolumes(volumes, true)
}

func restartHotVolumes(volumes []map[string]interface{}) {
	startVolumes(volumes, false)
}

func startVolumes(volumes []map[string]interface{}, shouldDelete bool) {
	//Remove db
	//Remove all the blocks

	for _, volI := range volumes {
		vPathI, ok := volI["path"]
		if !ok {
			Logger.Error("Discarding volume; Path field is required")
			continue
		}

		vPath := vPathI.(string)

		if shouldDelete {
			if err := os.RemoveAll(vPath); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err := os.MkdirAll(vPath, 0644); err != nil {
				Logger.Error(err.Error())
				continue
			}
		}

		availableSize, availableInodes, err := getAvailableSizeAndInodes(vPath)

		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := volI["size_to_maintain"]
		if ok {
			sizeToMaintain = sizeToMaintainI.(uint64)
		}

		if availableSize/(1024^3) <= sizeToMaintain {
			Logger.Error(ErrSizeLimit(vPath, sizeToMaintain).Error())
			continue
		}

		var inodesToMaintain uint64
		inodesToMaintainI, ok := volI["inodes_to_maintain"]
		if ok {
			inodesToMaintain = inodesToMaintainI.(uint64)
		}
		if availableInodes <= inodesToMaintain {
			Logger.Error(ErrInodesLimit(vPath, inodesToMaintain).Error())
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := volI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := volI["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
		}

		hotTier.volumes = append(hotTier.volumes, hotVolume{
			path:                vPath,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
			sizeToMaintain:      sizeToMaintain,
		})
	}

	if len(volumes) < len(hotTier.volumes)/2 {
		panic(errors.New("Atleast 50%% volumes must be able to store blocks"))
	}
}

func repairHotVolumes() {

}

//This function will recover metadata
func recoverHotMetaData(volumes []map[string]interface{}) {
	for _, volume := range volumes {
		volPathI, ok := volume["path"]
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

		for i := 0; i < HDCL; i++ {
			hotIndexPath := filepath.Join(volPath, fmt.Sprintf("%v%v", HK, i))
			if _, err := os.Stat(hotIndexPath); err != nil {
				Logger.Debug(fmt.Sprintf("Error while recovering metadata for index %v; Full path: %v; err: %v", i, hotIndexPath, err))
				continue
			}

			for j := 0; j < HDCL; j++ {
				blockSubDirPath := filepath.Join(hotIndexPath, fmt.Sprint("%v", j))
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

							if err := bwr.AddOrUpdate(); err != nil {
								Logger.Error(fmt.Sprintf("Error: %v, while reading file: %v", err, blockPath))
								errorOccurred = true
								goto CountUpdate
							}

						CountUpdate:
							totalBlocksCount++
							grandCount.mu.Lock()
							grandCount.totalBlocksCount++
							grandCount.totalBlocksSize += blockSize
							if errorOccurred {
								continue
							}
							grandCount.recoveredCount++
							grandCount.mu.Unlock()
							recoverCount++
						}
					}
					Logger.Info(fmt.Sprintf("%v Meta records recovered of %v blocks from path: %v", recoverCount, totalBlocksCount, gPath))

				}(blockSubDirPath)

			}
		}
		recoverWG.Wait() //wait for all goroutine to complete

		//Check available size and inodes and add volume to volume pool
		availableSize, availableInodes, err := getAvailableSizeAndInodes(volPath)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := volume["size_to_maintain"]
		if ok {
			sizeToMaintain = sizeToMaintainI.(uint64)
		}

		if availableSize/(1024^3) <= sizeToMaintain {
			Logger.Error(ErrSizeLimit(volPath, sizeToMaintain).Error())
			continue
		}

		var inodesToMaintain uint64
		inodesToMaintainI, ok := volume["inodes_to_maintain"]
		if ok {
			inodesToMaintain = inodesToMaintainI.(uint64)
		}
		if availableInodes <= inodesToMaintain {
			Logger.Error(ErrInodesLimit(volPath, inodesToMaintain).Error())
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := volume["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		if allowedBlockNumbers != 0 && grandCount.totalBlocksCount > allowedBlockNumbers {
			Logger.Error(ErrAllowedCountLimit(volPath, allowedBlockNumbers).Error())
			continue
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := volume["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
		}

		if allowedBlockSize != 0 && grandCount.totalBlocksSize > allowedBlockSize {
			Logger.Error(ErrAllowedSizeLimit(volPath, allowedBlockSize).Error())
			continue
		}

		hotTier.volumes = append(hotTier.volumes, hotVolume{
			path:                volPath,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
			sizeToMaintain:      sizeToMaintain,
			blocksCount:         uint64(grandCount.totalBlocksCount),
		})
	}
}

func recoverColdMetaData() {

}
