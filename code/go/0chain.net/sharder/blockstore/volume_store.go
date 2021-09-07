//Tiering is done to achieve large storage capacity, disk failures and performance as cache disk(SSD) will be used for latest and
// frequently used blocks.
//Hot tiering: Block data is in the cache disk
//Warm tiering: Block data in in HDD
//Cold tiering: Block data is in minio/s3/blobber server

package blockstore

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"0chain.net/chaincore/block"
)

const (
	Random        = "random"
	RoundRobin    = "round_robin"
	MinSizeFirst  = "min_size_first"
	MinCountFirst = "min_count_first"
)

// Expectation; 15 to 30 million blocks per year
//1600 KB
//Consider average to be 1MB

// var ErrDirKiloLimit = errors.New("") //todo write specific error formatter
const (
	Kilo    = "K" //Contains 1000 directories that contains 1000 blocks each so 10^6 blocks
	Mega    = "M" //Contains 1000 K directories so 10^9 blocks.
	Giga    = "G" //Contains 1000 M directories so 10 ^12 blocks.
	Peta    = "P" //Contains 1000 G directories so 10^15 blocks.
	Exa     = "E" //Contains 1000 P directories so 10^18 blocks.
	Zillion = "Z" //Contains 1000 E directories so 10^21 blocks. After this we would require new integer range
	//Longest path would be E0...999/P0...999/G0...999/M0...999/K0...999/0...999/{hash}.txt/.dat
	// eg. E0/P1/G0/M999/1/{hash}.file
	//A 100KB average block size would consume space for G directories about 10^17B > peta bytes
	//I suppose it should be limited to K directories so space is about 10^6
	//Max block size is 1.6MB so if we consider block size to be around average of 1MB then
	// space consumed by all K directories i.e. K0...999 is 10^3* 10^6*1MB is 1PB
)

type Volume struct { //Write is dependent on Volume struct but Read is independent
	rootPath                string
	blocksSize, blocksCount uint64
	//Available size is crude to be reliable because there can be other process that stores data in the disk making this
	//field unreliable. But as far as block storage is only concerned then it is reliable as the size is related
	//only to the addition or deletion of a block file.
	availableSize uint64
	subDir        string
	// Every new blocks will be stored in curDir unless it reaches count of certain number eg; 1000 blocks
	//since Kilo directories will contain 10^9 blocks resulting around 1PB of data so we can initially create empty directory
	// as K0/0 and if directory 0 has 1000 blocks then K0/1 is created which will be current curDir
	//This way if K0 contains 1000 such directories then inside subDir("data/blocks") K1/0 will be created and further stored
	//When K999 is filled then further block storage in this volume is prevented.
	//We can modify selectDir function later on if there can be more than 10^9 blocks in a volume.
	curDir string

	// Contains count of blocks in currentDirectory
	curDirBlockNums uint16 // Shows how many blocks are in current directory

	//todo Locking mechanism is required
}

//This function checks number of blocks stored in a directory and if it has 1000 blocks then it creates new directory
//and stores new blocks in it.
func (v *Volume) SelectDir() error {
	if v.curDirBlockNums < 999 {
		return nil
	}

	kDir, cDir := filepath.Split(v.curDir) //kDir --> k0,k1, etc; childDir cDir --> 0, 1, etc.
	cDirInt, _ := strconv.Atoi(cDir)
	if cDirInt < 999 {
		cDirInt++
		curDir := fmt.Sprintf("%v%v", kDir, cDirInt)
		newPath := filepath.Join(v.rootPath, v.subDir, curDir)
		err := os.Mkdir(newPath, 0644)
		if err != nil {
			return err
		}
		v.curDir = curDir
		v.curDirBlockNums = 0
		return nil

	}

	kIndex, _ := strconv.Atoi(strings.TrimRight(strings.TrimRight(kDir, "/"), "K"))
	if kIndex >= 999 {
		return fmt.Errorf("Volume %v has reached its Kilo limit", v.rootPath)
	}
	newkIndex := kIndex + 1
	kDir = fmt.Sprintf("%v%v/", Kilo, newkIndex)
	curDir := fmt.Sprintf("%v%v", kDir, "0")
	newPath := filepath.Join(v.rootPath, v.subDir, curDir)
	err := os.MkdirAll(newPath, 0644)
	if err != nil {
		return err
	}
	v.curDir = curDir
	v.curDirBlockNums = 0
	return nil
}

func MoveBlocks(v *Volume) {
	//Feature unrequired as there can't be more than Kilo directories
}

func (v *Volume) Write(b *block.Block, data []byte) (bPath string, err error) {
	err = v.SelectDir()
	if err != nil {
		return
	}
	bPath = path.Join(v.rootPath, v.subDir, v.curDir, fmt.Sprintf("%v.%v", b.Hash, fileExt))
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

func (v *Volume) updateSize(n int64) {
	if n < 0 {
		v.blocksSize -= uint64(n)
		v.availableSize += uint64(n)
	} else {
		v.blocksSize += uint64(n)
		v.availableSize -= uint64(n)
	}
}

func (v *Volume) updateCount(n int64) {
	if n < 0 {
		v.blocksCount -= uint64(n)
	} else {
		v.blocksCount += uint64(n)
	}
}

func (v *Volume) isAbleToStoreBlock() (ableToStore bool) {
	//check available size; available inodes
	var volStat unix.Statfs_t
	err := unix.Statfs(v.rootPath, &volStat)
	if err != nil {
		//log error
		return
	}

	if volStat.Files/volStat.Ffree < 10 { //dont' store if inodes less than 10 percent
		//log status
		return
	}
	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if availableSize/(1024*1024*1024) < 2 { //don't accept volume if it has lesser than 2GB
		//log status
		return
	}
	return unix.Access(v.rootPath, unix.W_OK) == nil
}

func volumeStrategy(strategy string) func(volumes *[]Volume, prevInd int) (*Volume, int) {
	//It seems better to remove volume from volumes list when it is unable to store blocks further
	switch strategy {
	case Random:
		return func(rVolumes *[]Volume, prevInd int) (*Volume, int) { //return volume path
			volumes := *rVolumes
			validVolumes := make([]Volume, len(volumes))
			copy(validVolumes, volumes)
			var selectedVolume *Volume
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
		return func(rVolumes *[]Volume, prevInd int) (*Volume, int) { //return volume path
			volumes := *rVolumes
			var selectedVolume *Volume
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
		return func(rVolumes *[]Volume, prevInd int) (*Volume, int) {
			volumes := *rVolumes
			var selectedVolume *Volume
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
		return func(rVolumes *[]Volume, prevInd int) (*Volume, int) {
			volumes := *rVolumes
			var selectedVolume *Volume
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

//This function will clean data/blocks directory.
func checkVolumes(volumesPath []string) (volumes []Volume) {
	for _, v := range volumesPath {
		var volStat unix.Statfs_t
		err := unix.Statfs(v, &volStat)
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

		subDir := "data/blocks"
		os.RemoveAll(subDir)
		curDir := fmt.Sprintf("%v%v/%v", Kilo, 0, 0)
		path := filepath.Join(v, subDir, curDir)
		err = os.MkdirAll(path, 0644)
		if err != nil {
			panic(fmt.Errorf("Could not create path: %v in volume %v; Got error \"%v\"", path, v, err))
		}

		volumes = append(volumes, Volume{
			subDir:        subDir,
			rootPath:      v,
			availableSize: 0,
			curDir:        curDir,
		})
	}
	if len(volumes) < len(volumesPath)/2 {
		//log status
		//panic here
	}
	return
}

func RecoverVolumes(volumesPath []string) (volumes []Volume) {
	//Given that meta record is corrupted it still can be recovered.
	//Start sharder with recovery mode which will check for all the blocks in the volumes and update the meta record.

	var size uint64
	var count uint64
	for _, v := range volumesPath {
		blocksPath := filepath.Join(v, "data/blocks")
		err := filepath.Walk(blocksPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				//Add information to meta data recorder
				count++
				size += uint64(info.Size())
			}
			return nil

		})
		if err != nil {
			//log error
			continue
		}

		volume := Volume{rootPath: v, blocksSize: size, blocksCount: count}
		volumes = append(volumes, volume)
	}
	return
}
