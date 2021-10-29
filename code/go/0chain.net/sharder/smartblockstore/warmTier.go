package smartblockstore

import (
	"bufio"
	"compress/zlib"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"golang.org/x/sys/unix"
)

// Expectation; 15 to 30 million blocks per year
//Max block size 1.6MB
//Consider average block size to be 1MB

// var ErrDirKiloLimit = errors.New("") //todo write specific error formatter
const (
	WDCL  = 1000
	WKilo = "WK" //Contains 1000 directories that contains 1000 blocks each so 10^6 blocks; So 1000 K directories contains 10^9 blocks
	// Mega    = "M"  //Contains 1000 K directories so each M directory contains 10^9 blocks.
	// Giga    = "G"  //Contains 1000 M directories so each G directory contains 10 ^12 blocks.
	// Peta    = "P"  //Contains 1000 G directories so each P directory contains 10^15 blocks.
	// Exa     = "E"  //Contains 1000 P directories so each E directory contains 10^18 blocks.
	// Zillion = "Z"  //Contains 1000 E directories so each Z directory contains 10^21 blocks. After this we would require new integer
	//range. Longest path would be E0...999/P0...999/G0...999/M0...999/K0...999/0...999/{hash}.txt/.dat
	// eg. E0/P1/G0/M999/1/{hash}.file
	//A 100KB average block size would consume space for G directories about 10^17B > peta bytes
	//I suppose it should be limited to K directories so space is about 10^6
	//Max block size is 1.6MB so if we consider block size to be around average of 1MB then
	// space consumed by all K directories i.e. K0...999 is 10^3* 10^6*1MB is 1PB
)

type wTier struct {
	volumes          []warmVolume
	volume           *warmVolume
	prevVolInd       int
	selectNextVolume func(volumes []warmVolume, prevVolInd int) (*warmVolume, int)
}

var warmTier wTier

type warmVolume struct {
	path                    string
	sizeToMaintain          uint64
	allowedBlockNumbers     uint64
	allowedBlockSize        uint64
	blocksSize, blocksCount uint64
	//Available size is only reliable when other process does not store data in it. So it must be made sure that only blocks will be stored otherwise
	//some strategy will not function properly
	availableSize uint64
	// subDir                  string
	// Every new blocks will be stored in curDir unless it reaches count of certain number eg; 1000 blocks
	//since Kilo directories will contain 10^9 blocks resulting around 1PB of data so we can initially create empty directory
	// as K0/0 and if directory 0 has 1000 blocks then K0/1 is created which will be current curDir
	//This way if K0 contains 1000 such directories then inside subDir("data/blocks") K1/0 will be created and further stored
	//When K999 is filled then further block storage in this volume is prevented.
	//We can modify selectDir function later on if there can be more than 10^9 blocks in a volume.
	curWKInd        uint32 //K index; K0, K1, etc.
	curDirInd       uint32 // Dir index; 0, 1, etc.
	curDirBlockNums uint32
}

//Add blocks sequentially to the directories; K0/0, K0/1 ... K1/0, K1/1, ..., K999/0, K999/1, K0/0, K0/1,...
func (v *warmVolume) selectDir() error {
	if v.curDirBlockNums < WDCL-1 {
		blocksPath := filepath.Join(v.path, fmt.Sprintf("%v%v/%v", WKilo, v.curWKInd, v.curDirInd))
		_, err := os.Stat(blocksPath)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(blocksPath, 0644); err != nil {
				return err
			}
		}
		return nil
	}

	if v.curDirInd < WDCL-1 {
		dirInd := v.curDirInd + 1
		blocksPath := filepath.Join(v.path, fmt.Sprintf("K%v/%v", v.curWKInd, dirInd))
		blocksCount, err := countFiles(blocksPath)

		if err != nil && errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(blocksPath, 0644)
			if err != nil {
				return err
			}
			v.curDirInd = dirInd
			v.curDirBlockNums = 0
			return nil
		} else if err != nil {
			return err
		}

		if blocksCount >= WDCL {
			return ErrVolumeFull(v.path)
		}

		v.curDirInd = dirInd
		v.curDirBlockNums = uint32(blocksCount)
		return nil
	}

	var wkInd uint32
	if v.curWKInd < WDCL-1 {
		wkInd = v.curWKInd + 1
	} else {
		wkInd = 0
	}

	dirInd := uint32(0)
	blocksPath := filepath.Join(v.path, fmt.Sprintf("K%v/%v", wkInd, dirInd))
	blocksCount, err := countFiles(blocksPath)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(blocksPath, 0644)
		if err != nil {
			return err
		}
		v.curDirInd = dirInd
		v.curDirBlockNums = 0

		return nil
	} else if err != nil {
		return err
	}

	if blocksCount >= WDCL {
		return ErrVolumeFull(v.path)
	}

	v.curWKInd = wkInd
	v.curDirInd = dirInd
	v.curDirBlockNums = uint32(blocksCount)
	return nil
}

func (wv *warmVolume) write(b *block.Block, data []byte) (bPath string, err error) {
	bPath = path.Join(wv.path, fmt.Sprintf("K%v/%v", wv.curWKInd, wv.curDirInd), fmt.Sprintf("%v.%v", b.Hash, fileExt))
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

	wv.curDirBlockNums++
	wv.updateCount(1)
	wv.updateSize(int64(n))
	return
}

func (wv *warmVolume) read(hash, blockPath string) (*block.Block, error) {
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

func (v *warmVolume) updateSize(n int64) {
	if n < 0 {
		v.blocksSize -= uint64(n)
		v.availableSize += uint64(n)
	} else {
		v.blocksSize += uint64(n)
		v.availableSize -= uint64(n)
	}
}

func (v *warmVolume) updateCount(n int64) {
	if n < 0 {
		v.blocksCount -= uint64(n)
	} else {
		v.blocksCount += uint64(n)
	}
}

//*******************************************************Volume Strategy**********************************************************
func (v *warmVolume) isAbleToStoreBlock() (ableToStore bool) {
	//check available size; available inodes
	var volStat unix.Statfs_t
	err := unix.Statfs(v.path, &volStat)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	if v.blocksSize >= v.allowedBlockSize {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block size. Allowed: %v, Total block size: %v", v.allowedBlockSize, v.blocksSize))
		return
	}

	if v.blocksCount >= v.allowedBlockNumbers {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block numbers. Allowed: %v, Total block size: %v", v.allowedBlockNumbers, v.blocksCount))
		return
	}

	if float64(volStat.Ffree)/float64(volStat.Bavail) < 0.1 { //return false if available inodes is lesser than 10%
		Logger.Error(fmt.Sprintf("Less than 10%% available inodes for volume: %v", v.path))
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if availableSize/(1024*1024*1024) < uint64(v.sizeToMaintain) { //don't accept volume if it has lesser than size to maintain
		//log status
		return
	}

	if unix.Access(v.path, unix.W_OK) != nil {
		return
	}

	if err := v.selectDir(); err != nil {
		Logger.Error(ErrSelectDir(v.path, err))
		return
	}

	return true
}

func warmInit(wConf map[string]interface{}) {
	volumesI, ok := wConf["volumes"]
	if !ok {
		panic(errors.New("Volumes config not available"))
	}
	volumes := volumesI.([]map[string]interface{})
	checkWarmVolumes(volumes)

	var strategy string
	strategyI, ok := wConf["strategy"]
	if !ok {
		strategy = DefaultWarmStrategy
	} else {
		strategy = strategyI.(string)
	}

	Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))
	var f func(warmVolumes []warmVolume, prevInd int) (*warmVolume, int)

	switch strategy {
	default:
		panic(fmt.Errorf("Strategy %v is not supported", strategy))
	case Random:
		f = func(warmVolumes []warmVolume, prevInd int) (*warmVolume, int) {
			var selectedVolume *warmVolume
			var selectedIndex int
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for len(warmVolumes) > 0 {
				ind := r.Intn(len(warmVolumes))
				selectedVolume = &warmVolumes[ind]
				if selectedVolume.isAbleToStoreBlock() {
					selectedIndex = ind
					break
				}
				warmVolumes[ind] = warmVolumes[len(warmVolumes)-1]
				warmVolumes = warmVolumes[:len(warmVolumes)-1]
			}

			warmTier.volumes = warmVolumes
			return selectedVolume, selectedIndex
		}
	case RoundRobin:
		f = func(warmVolumes []warmVolume, prevInd int) (*warmVolume, int) { //return volume path
			var selectedVolume *warmVolume
			prevVolume := warmVolumes[prevInd]
			var selectedIndex int

			if prevInd < 0 {
				prevInd = -1
			}

			for i := prevInd + 1; i != prevInd; i++ {
				if len(warmVolumes) == 0 {
					break
				}
				if i >= len(warmVolumes) {
					i = len(warmVolumes) - i
				}
				if i < 0 {
					i = 0 //i can be negative when a selected volume fail to store block.
				}

				v := warmVolumes[i]
				if v.isAbleToStoreBlock() {
					selectedVolume = &v
					selectedIndex = i
					break
				} else {
					warmVolumes = append(warmVolumes[:i], warmVolumes[i+1:]...)
					if i < prevInd {
						prevInd--
					}
					i--
				}
			}
			if selectedVolume == nil {
				if prevVolume.isAbleToStoreBlock() {
					selectedVolume = &prevVolume
					selectedIndex = prevInd
				}
			}
			warmTier.volumes = warmVolumes
			return selectedVolume, selectedIndex
		}
	case MinCountFirst:
		f = func(warmVolumes []warmVolume, prevInd int) (*warmVolume, int) {
			var selectedVolume *warmVolume
			var selectedIndex int

			count := len(warmVolumes)
			for i := 0; i < count; i++ {
				v := warmVolumes[i]

				if !v.isAbleToStoreBlock() {
					warmVolumes = append(warmVolumes[:i], warmVolumes[i+1:]...)
					i--
					count--
					continue
				}

				if selectedVolume == nil {
					selectedVolume = &v
					selectedIndex = i
					continue
				}

				if v.blocksCount < selectedVolume.blocksCount {
					selectedVolume = &v
					selectedIndex = i
				}
			}

			warmTier.volumes = warmVolumes
			return selectedVolume, selectedIndex
		}
	case MinSizeFirst:
		f = func(warmVolumes []warmVolume, prevInd int) (*warmVolume, int) {
			var selectedVolume *warmVolume
			var selectedIndex int

			count := len(warmVolumes)
			for i := 0; i < count; i++ {
				v := warmVolumes[i]

				if !v.isAbleToStoreBlock() {
					warmVolumes = append(warmVolumes[:i], warmVolumes[i+1:]...)
					i--
					count--
					continue
				}

				if selectedVolume == nil {
					selectedVolume = &v
					selectedIndex = i
					continue
				}

				if v.availableSize > selectedVolume.availableSize {
					selectedVolume = &v
					selectedIndex = i
				}
			}
			warmTier.volumes = warmVolumes
			return selectedVolume, selectedIndex
		}

	}

	warmTier.selectNextVolume = f
}

func checkWarmVolumes(volumesInfo []map[string]interface{}) {
	for _, v := range volumesInfo {
		vPathI, ok := v["path"]
		if !ok {
			continue
		}
		vPath := vPathI.(string)

		var volStat unix.Statfs_t
		err := unix.Statfs(vPath, &volStat)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		if volStat.Files/volStat.Ffree < 10 { //dont' store if inodes less than 10 percent
			Logger.Error(fmt.Sprintf("Volume %v has less than 10%% available inodes", vPath))
			continue
		}
		availableSize := volStat.Bfree * uint64(volStat.Bsize)
		if availableSize/(1024*1024*1024) < 2 { //don't accept volume if it has lesser than 2GB
			Logger.Error(fmt.Sprintf("Volume %v has less than 2GB available", vPath))
			continue
		}

		err = os.RemoveAll(vPath)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		if err := os.MkdirAll(vPath, 0644); err != nil {
			Logger.Error(err.Error())
			continue
		}

		curDir := fmt.Sprintf("%v%v/%v", WKilo, 0, 0)
		path := filepath.Join(vPath, curDir)
		err = os.MkdirAll(path, 0644)
		if err != nil {
			Logger.Error(fmt.Sprintf("Could not create path: %v in volume %v; Got error \"%v\"", path, v, err))
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := v["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := v["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := v["maintain_size"]
		if ok {
			sizeToMaintain = sizeToMaintainI.(uint64)
		}
		warmTier.volumes = append(warmTier.volumes, warmVolume{
			path:                vPath,
			availableSize:       availableSize,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
			sizeToMaintain:      sizeToMaintain,
		})
	}

	if len(warmTier.volumes) < len(volumesInfo)/2 { //Atleast 50% volumes must be able to store blocks
		//log status
		panic(errors.New("Atleast 50% volumes must be able to store blocks"))
	}
}

//**********************************************Recover metadata: check volumes and cold storage*********************
func RecoverMetaData(volumesInfo []map[string]interface{}, coldConfig interface{}) error {
	//Given that meta record is corrupted it still can be recovered.
	//Start sharder with recovery mode which will check for all the blocks in the volumes and update the meta record.

	var size uint64
	var count uint64
	for _, v := range volumesInfo {
		volumePathI, ok := v["path"]
		if !ok {
			panic(fmt.Errorf("Path value is required"))
		}
		volumePath := volumePathI.(string)
		err := filepath.Walk(volumePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				//TODO Add information to meta data recorder
				count++
				size += uint64(info.Size())
			}
			return nil

		})
		if err != nil {
			//log error
			return err
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := v["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := v["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := v["maintain_size"]
		if ok {
			sizeToMaintain = sizeToMaintainI.(uint64)
		}

		volume := warmVolume{
			path:                volumePath,
			blocksSize:          size,
			blocksCount:         count,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
			sizeToMaintain:      sizeToMaintain,
		}
		warmTier.volumes = append(warmTier.volumes, volume)
	}

	//Also update meta data from cold storage
	return nil
}

//**********************************************Repair volumes: Get and store missing blocks*************************

//RepairVolumes It gets and store missing blocks because some partition failed at some point. If SSD also failed along with some
//HDD then it is necessary to know which rounds this sharder was responsible to store.
//Q1: If sharder goes offline and comes back what is its status in the blockchain network?
//Q2: Can we know if this sharder was responsible to store some round block?
func RepairVolumes(volumesTorepair []map[string]interface{}) error {
	return nil
}
