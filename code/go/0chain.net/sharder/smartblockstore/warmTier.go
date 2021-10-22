package blockstore

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
	"golang.org/x/sys/unix"
)

const (
	Random        = "random"
	RoundRobin    = "round_robin"
	MinSizeFirst  = "min_size_first"
	MinCountFirst = "min_count_first"
)

// Expectation; 15 to 30 million blocks per year
//Max block size 1.6MB
//Consider average block size to be 1MB

// var ErrDirKiloLimit = errors.New("") //todo write specific error formatter
const (
	Kilo    = "K" //Contains 1000 directories that contains 1000 blocks each so 10^6 blocks
	Mega    = "M" //Contains 1000 K directories so each M directory contains 10^9 blocks.
	Giga    = "G" //Contains 1000 M directories so each G directory contains 10 ^12 blocks.
	Peta    = "P" //Contains 1000 G directories so each P directory contains 10^15 blocks.
	Exa     = "E" //Contains 1000 P directories so each E directory contains 10^18 blocks.
	Zillion = "Z" //Contains 1000 E directories so each Z directory contains 10^21 blocks. After this we would require new integer
	//range. Longest path would be E0...999/P0...999/G0...999/M0...999/K0...999/0...999/{hash}.txt/.dat
	// eg. E0/P1/G0/M999/1/{hash}.file
	//A 100KB average block size would consume space for G directories about 10^17B > peta bytes
	//I suppose it should be limited to K directories so space is about 10^6
	//Max block size is 1.6MB so if we consider block size to be around average of 1MB then
	// space consumed by all K directories i.e. K0...999 is 10^3* 10^6*1MB is 1PB
)

var ErrVolumeFull = func(volPath string) error {
	return fmt.Errorf("Volume %v is full.", volPath)
}

func countFiles(dirPath string) (count uint32, err error) {
	var files []os.DirEntry
	files, err = os.ReadDir(dirPath)
	if err != nil {
		return
	}
	count = uint32(len(files))
	return
}

type volume struct {
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
	curKInd         uint32 //K index; K0, K1, etc.
	curDirInd       uint32 // Dir index; 0, 1, etc.
	curDirBlockNums uint32
}

//Add blocks sequentially to the directories; K0/0, K0/1 ... K1/0, K1/1, ..., K999/0, K999/1, K0/0, K0/1,...
func (v *volume) selectDir() error {
	if v.curDirBlockNums < 999 {
		return nil
	}

	if v.curDirInd < 999 {
		dirInd := v.curDirInd + 1
		blocksPath := filepath.Join(v.path, fmt.Sprintf("K%v/%v", v.curKInd, dirInd))
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

		if blocksCount >= 999 {
			return ErrVolumeFull(v.path)
		}

		v.curDirInd = dirInd
		v.curDirBlockNums = blocksCount
		return nil
	}

	if v.curKInd < 999 {
		kInd := v.curKInd + 1
		dirInd := uint32(0)
		blocksPath := filepath.Join(v.path, fmt.Sprintf("K%v/%v", kInd, dirInd))
		blocksCount, err := countFiles(blocksPath)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			err := os.Mkdir(blocksPath, 0644)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		if blocksCount >= 999 {
			return ErrVolumeFull(v.path)
		}

		v.curKInd = kInd
		v.curDirInd = dirInd
		v.curDirBlockNums = blocksCount
	}
	return nil
}

func (v *volume) Write(b *block.Block, data []byte) (bPath string, err error) {
	err = v.selectDir()
	if err != nil {
		return
	}

	bPath = path.Join(v.path, fmt.Sprintf("K%v/%v", v.curKInd, v.curDirInd), fmt.Sprintf("%v.%v", b.Hash, fileExt))
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
	v.curDirBlockNums++
	v.updateCount(1)
	v.updateSize(int64(n))
	return
}

func (v *volume) updateSize(n int64) {
	if n < 0 {
		v.blocksSize -= uint64(n)
		v.availableSize += uint64(n)
	} else {
		v.blocksSize += uint64(n)
		v.availableSize -= uint64(n)
	}
}

func (v *volume) updateCount(n int64) {
	if n < 0 {
		v.blocksCount -= uint64(n)
	} else {
		v.blocksCount += uint64(n)
	}
}

//*******************************************************Volume Strategy**********************************************************
func (v *volume) isAbleToStoreBlock() (ableToStore bool) {
	//check available size; available inodes
	var volStat unix.Statfs_t
	err := unix.Statfs(v.path, &volStat)
	if err != nil {
		//log error
		return
	}

	if v.blocksSize >= v.allowedBlockSize {
		//log status
		return
	}

	if v.blocksCount >= v.allowedBlockNumbers {
		//log status
		return
	}

	if float64(volStat.Ffree)/float64(volStat.Bavail) < 0.1 { //return false if available inodes is lesser than 10%
		//log status
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if availableSize/(1024*1024*1024) < uint64(v.sizeToMaintain) { //don't accept volume if it has lesser than size to maintain
		//log status
		return
	}

	return unix.Access(v.path, unix.W_OK) == nil
}

func volumeStrategy(strategy string) func(volumes *[]volume, prevInd int) (*volume, int) {
	//It seems better to remove volume from volumes list when it is unable to store blocks further
	switch strategy {
	case Random:
		return func(rVolumes *[]volume, prevInd int) (*volume, int) { //return volume path
			volumes := *rVolumes
			validVolumes := make([]volume, len(volumes))
			copy(validVolumes, volumes)
			var selectedVolume *volume
			var selectedIndex int
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for len(validVolumes) > 0 {
				ind := r.Intn(len(validVolumes))
				selectedVolume = &validVolumes[ind]
				if selectedVolume.isAbleToStoreBlock() {
					selectedIndex = ind
					break
				} else {
					//remove an element from slice; ordering is not important
					//garbage collection is not an issue here
					validVolumes[ind] = validVolumes[len(validVolumes)-1]
					validVolumes = validVolumes[:len(validVolumes)-1]
				}
			}
			return selectedVolume, selectedIndex
		}
	case RoundRobin:
		return func(rVolumes *[]volume, prevInd int) (*volume, int) { //return volume path
			volumes := *rVolumes
			var selectedVolume *volume
			var selectedIndex int
			totalVolumes := len(volumes)

			if prevInd < 0 {
				prevInd = -1
			}

			for i := prevInd + 1; i != prevInd; i++ {
				if i >= totalVolumes {
					if prevInd < 0 {
						break
					}
					i = totalVolumes - i
					if i < 0 {
						i = 0 //i can be negative when a selected volume fail to store block.
					}
				}
				v := volumes[i]
				if v.isAbleToStoreBlock() {
					selectedVolume = &v
					selectedIndex = i
					break
				}
			}
			if selectedVolume == nil {
				if volumes[prevInd].isAbleToStoreBlock() {
					selectedVolume = &volumes[prevInd]
					selectedIndex = prevInd
				}
			}
			return selectedVolume, selectedIndex
		}
	case MinCountFirst:
		return func(rVolumes *[]volume, prevInd int) (*volume, int) {
			volumes := *rVolumes
			var selectedVolume *volume
			var selectedIndex int
			for i := 0; i < len(volumes); i++ {
				v := volumes[i]
				if !v.isAbleToStoreBlock() {
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
			return selectedVolume, selectedIndex
		}
	case MinSizeFirst:
		return func(rVolumes *[]volume, prevInd int) (*volume, int) {
			volumes := *rVolumes
			var selectedVolume *volume
			var selectedIndex int
			for i := 0; i < len(volumes); i++ {
				v := volumes[i]
				if !v.isAbleToStoreBlock() {
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
			return selectedVolume, selectedIndex
		}
	default:
		//
		panic(fmt.Errorf("Stragegy %v not defined", strategy))
	}
}

type wTier struct {
	volumes    []volume
	nextVolume *volume
	prevVolInd int
	pickVolume func(volumes *[]volume, prevVolInd int) (*volume, int)
}

var warmTier wTier

func startVolumes(volumesInfo []map[string]interface{}) (volumes []volume) {
	// TODO also check if inodes available is enough for allowed blocks number
	for _, v := range volumesInfo {
		vPath, ok := v["path"]
		if !ok {
			continue
		}
		volumePath := vPath.(string)

		var volStat unix.Statfs_t
		err := unix.Statfs(volumePath, &volStat)
		if err != nil {
			//log error
			continue
		}

		if volStat.Files/volStat.Ffree < 10 { //dont' store if inodes less than 10 percent
			//log status
			continue
		}
		availableSize := volStat.Bfree * uint64(volStat.Bsize)
		if availableSize/(1024*1024*1024) < 2 { //don't accept volume if it has lesser than 2GB
			//log status
			continue
		}

		dirents, err := os.ReadDir(volumePath)
		if err != nil {
			//log error
			continue
		}

		var removeErr error
		for _, dirent := range dirents {
			p := filepath.Join(volumePath, dirent.Name())
			removeErr = os.RemoveAll(p)
			if removeErr != nil {
				break
			}
		}
		if removeErr != nil {
			//log error
			continue
		}

		curDir := fmt.Sprintf("%v%v/%v", Kilo, 0, 0)
		path := filepath.Join(volumePath, curDir)
		err = os.MkdirAll(path, 0644)
		if err != nil {
			//log error fmt.Errorf("Could not create path: %v in volume %v; Got error \"%v\"", path, v, err)
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
		warmTier.volumes = append(warmTier.volumes, volume{
			path:                volumePath,
			availableSize:       availableSize,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
			sizeToMaintain:      sizeToMaintain,
		})
	}

	if len(volumes) < len(volumesInfo)/2 { //Atleast 50% volumes must be able to store blocks
		//log status
		panic(errors.New("Atleast 50% volumes must be able to store blocks"))
	}

	// TODO also clean meta records
	return
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

		volume := volume{
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
