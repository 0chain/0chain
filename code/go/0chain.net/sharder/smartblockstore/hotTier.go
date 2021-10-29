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
}

type hotVolume struct {
	path                    string
	allowedBlockNumbers     uint64
	allowedBlockSize        uint64
	sizeToMaintain          uint64
	blocksSize, blocksCount uint64
	availableSize           uint64
	curHKInd                uint32
	curDirInd               uint32
	curDirBlockNums         uint32
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
		hv.availableSize += uint64(n)
	} else {
		hv.blocksSize += uint64(n)
		hv.availableSize -= uint64(n)
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

	if float64(volStat.Ffree)/float64(volStat.Bavail) < 0.1 { //return false if available inodes is lesser than 10%
		Logger.Error(fmt.Sprintf("Less than 10%% available inodes for volume: %v", hv.path))
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if availableSize/(1024*1024*1024) < uint64(hv.sizeToMaintain) { //don't accept volume if it has lesser than size to maintain
		//log status
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

func hotInit(hConf map[string]interface{}) {
	volumesI, ok := hConf["volumes"]
	if !ok {
		panic(errors.New("Volumes config not available"))
	}

	volumes := volumesI.([]map[string]interface{})
	checkHotVolumes(volumes)

	var strategy string
	strategyI, ok := hConf["strategy"]
	if !ok {
		strategy = DefaultHotStrategy
	} else {
		strategy = strategyI.(string)
	}

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

				if v.availableSize > selectedVolume.availableSize {
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

func checkHotVolumes(volumes []map[string]interface{}) {
	for _, volI := range volumes {
		vPathI, ok := volI["path"]
		if !ok {
			Logger.Error("Discarding volume; Path field is required")
			continue
		}
		vPath := vPathI.(string)
		var volStat unix.Statfs_t
		err := unix.Statfs(vPath, &volStat)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}
		if volStat.Ffree*10/volStat.Files < 10 { //ignore if inodes is less than 10 percent
			Logger.Error("Less than 10% available inodes")
			continue
		}

		availableSize := volStat.Bfree * uint64(volStat.Bsize)
		if availableSize/(1024*1024*1024) < 2 { //ignore if size is lesser than 2GB
			Logger.Error("Less than 2GB available size")
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

		var sizeToMaintain uint64
		sizeToMaintainI, ok := volI["size_to_maintain"]
		if ok {
			sizeToMaintain = sizeToMaintainI.(uint64)
		}

		hotTier.volumes = append(hotTier.volumes, hotVolume{
			path:                vPath,
			availableSize:       availableSize,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
			sizeToMaintain:      sizeToMaintain,
		})
	}
	if len(volumes) < len(hotTier.volumes)/2 {
		panic(errors.New("Atlest 50%% volumes must be able to store blocks"))
	}
}
