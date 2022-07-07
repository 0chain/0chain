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
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/viper"
	"golang.org/x/sys/unix"
)

var volumesMap map[string]*volume
var unableVolumes map[string]*volume

const (
	DirPrefix = "blocks/K"
	// Hot directory content limit
	// Contains 2000 directories that contains 2000 blocks each, so one twoKilo directory contains 4*10^6blocks.
	// So 2000 such twokilo directories will contain 8*10^9 blocks
	DCL = 2000

	// minSizeFirst will choose volume that has stored lesser blocks size
	MinSizeFirst = "min_size_first"
	// random will choose volume randomly
	Random = "random"
	// roundRobin will choose volume one after other
	RoundRobin = "round_robin"
	// minCountFirst will choose volume that has stored lesser number of blocks
	MinCountFirst = "min_count_first"
	// fillFirst will first fill the volume and move to other volume
	FillFirst = "fill_first"
)

type selectedDiskVolume struct {
	volume  *volume
	prevInd int
	err     error
}

type diskTier struct {
	Volumes          []*volume // List of hot volumes
	SelectNextVolume func(volumes []*volume, prevInd int)
	SelectedVolumeCh <-chan selectedDiskVolume // volume that will be used to store blocks next
	PrevVolInd       int
	Mu               Mutex
}

// removeSelectedVolume will remove volumes from the list and put it in unableVolumes
func (d *diskTier) removeSelectedVolume() {
	// If some other process has acquired lock then it will either remove or continue to store
	// blocks
	if !d.Mu.TryLock() {
		return
	}
	defer d.Mu.Unlock()

	selectedVolume := d.Volumes[d.PrevVolInd]
	unableVolumes[selectedVolume.Path] = selectedVolume
	d.Volumes = append(d.Volumes[:d.PrevVolInd], d.Volumes[d.PrevVolInd+1:]...)
	d.PrevVolInd--
}

func (d *diskTier) write(b *block.Block, data []byte) (blockPath string, err error) {
	defer func() {
		logging.Logger.Info("Selecting next volume")
		go d.SelectNextVolume(d.Volumes, d.PrevVolInd)
	}()

	for {
		logging.Logger.Info(fmt.Sprintf("Waiting channel for selected volume to write block %v", b.Hash))
		sdv := <-d.SelectedVolumeCh
		if sdv.err != nil {
			return "", sdv.err
		}

		d.PrevVolInd = sdv.prevInd

		if blockPath, err = sdv.volume.write(b, data); err != nil {
			logging.Logger.Error(err.Error())
			d.removeSelectedVolume()
			go d.SelectNextVolume(d.Volumes, d.PrevVolInd)
			continue
		}

		return
	}
}

func (dTier *diskTier) read(bPath string) (b *block.Block, err error) {
	b = new(block.Block)
	f, err := os.Open(bPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	err = datastore.ReadJSON(r, b)
	if err != nil {
		return nil, err
	}

	return
}

type volume struct {
	Path string

	AllowedBlockNumbers uint64
	AllowedBlockSize    uint64

	SizeToMaintain   uint64
	InodesToMaintain uint64

	CountMu     *sync.Mutex // Count Mutex to update blockssize and blockscount
	BlocksSize  uint64
	BlocksCount uint64

	// used in selecting directory
	IndMu     *sync.Mutex // Index mutex to update indexes and current directory blocks count
	CurKInd   int
	CurDirInd int
	// CurDirBlockNums is count of number of blocks in current directory
	// Lock is not required to change value as it is either incremented by 1 or set to zero.
	// And both are valid.
	CurDirBlockNums int
}

func (v *volume) selectDir() error {
	v.IndMu.Lock()
	defer v.IndMu.Unlock()

	switch {
	case v.CurDirBlockNums < DCL:

		blocksPath := filepath.Join(v.Path, fmt.Sprintf("%v%v/%v", DirPrefix, v.CurKInd, v.CurDirInd))
		_, err := os.Stat(blocksPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if err := os.MkdirAll(blocksPath, 0777); err != nil {
					return err
				}
				return nil
			}
		}
		return err

	case v.CurDirInd < DCL-1:
		dirInd := v.CurDirInd + 1
		blocksPath := filepath.Join(v.Path, fmt.Sprintf("%v%v/%v", DirPrefix, v.CurKInd, dirInd))
		blocksCount, err := countFiles(blocksPath)

		if err != nil && errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(blocksPath, 0777)
			if err != nil {
				return err
			}
			v.CurDirInd = dirInd
			v.CurDirBlockNums = 0
		} else if err != nil {
			return err
		}

		if blocksCount >= DCL {
			return ErrVolumeFull(v.Path)
		}

		v.CurDirInd = dirInd
		// blocksCount < DCL means that a worker is moving blocks to cold tier so that it will
		// eventually be limited to DCL. Putting v.CurDirBlockNums = blocksCount will result in
		// partial fill up of directories.
		v.CurDirBlockNums = 0

		return updateCurIndexes(filepath.Join(v.Path, IndexStateFileName), v.CurKInd, v.CurDirInd)
	}

	var kInd int
	if v.CurKInd < DCL-1 {
		kInd = v.CurKInd + 1
	}

	dirInd := 0
	blocksPath := filepath.Join(v.Path, fmt.Sprintf("%v%v/%v", DirPrefix, kInd, dirInd))
	blocksCount, err := countFiles(blocksPath)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(blocksPath, 0777)
		if err != nil {
			return err
		}

		v.CurKInd = kInd
		v.CurDirInd = dirInd
		v.CurDirBlockNums = 0

		return updateCurIndexes(filepath.Join(v.Path, IndexStateFileName), v.CurKInd, v.CurDirInd)
	} else if err != nil {
		return err
	}

	if blocksCount >= DCL {
		return ErrVolumeFull(v.Path)
	}

	v.CurKInd = kInd
	v.CurDirInd = dirInd
	// blocksCount < DCL means that a worker is moving blocks to cold tier so that it will
	// eventually be limited to DCL. Putting v.CurDirBlockNums = blocksCount will result in
	// partial fill up of directories.
	v.CurDirBlockNums = 0

	return updateCurIndexes(filepath.Join(v.Path, IndexStateFileName), v.CurKInd, v.CurDirInd)
}

func (v *volume) write(b *block.Block, data []byte) (bPath string, err error) {
	bPath = path.Join(v.Path,
		fmt.Sprintf("%v%v/%v", DirPrefix, v.CurKInd, v.CurDirInd),
		fmt.Sprintf("%v%v", b.Hash, fileExt))

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
	v.updateCountAndSize(1, int64(n))

	return
}

// When a block is moved to cold tier delete function will be called
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

	v.updateCountAndSize(-1, -size)
	return nil
}

func (v *volume) updateCountAndSize(count, size int64) {
	v.CountMu.Lock()
	v.BlocksCount += uint64(count)
	v.BlocksSize += uint64(size)
	v.CountMu.Unlock()
}

func (v *volume) isAbleToStoreBlock() (ableToStore bool) {
	var volStat unix.Statfs_t
	err := unix.Statfs(v.Path, &volStat)
	if err != nil {
		logging.Logger.Error(err.Error())
		return
	}

	if v.AllowedBlockSize != 0 && v.BlocksSize >= v.AllowedBlockSize {
		logging.Logger.Error(fmt.Sprintf(
			"Storage limited by allowed block size. Allowed: %v, Total block size: %v",
			v.AllowedBlockSize, v.BlocksSize))
		return
	}

	if v.AllowedBlockNumbers != 0 && v.BlocksCount >= v.AllowedBlockNumbers {
		logging.Logger.Error(fmt.Sprintf(
			"Storage limited by allowed block numbers. Allowed: %v, Total blocks count: %v",
			v.AllowedBlockNumbers, v.BlocksCount))
		return
	}

	if v.InodesToMaintain != 0 && volStat.Ffree <= v.InodesToMaintain {
		logging.Logger.Error(fmt.Sprintf(
			"Available Inodes for volume %v is less than inodes to maintain(%v)",
			v.Path, v.InodesToMaintain))
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if v.SizeToMaintain != 0 && availableSize < uint64(v.SizeToMaintain) {
		logging.Logger.Error(fmt.Sprintf(
			"Available size for volume %v is less than size to maintain(%v)",
			v.Path, v.SizeToMaintain))
		return
	}

	if unix.Access(v.Path, unix.W_OK) != nil {
		return
	}

	if err := v.selectDir(); err != nil {
		logging.Logger.Error(ErrSelectDir(v.Path, err))
		return
	}

	return true
}

func initDisk(vViper *viper.Viper, mode string) *diskTier {
	strategy := vViper.GetString("strategy")
	if strategy == "" {
		strategy = DefaultVolumeStrategy
	}

	volumesI := vViper.Get("volumes")
	if volumesI == nil {
		panic(errors.New("Volumes config not available"))

	}

	volumesMapI := volumesI.([]interface{})
	var volsMap []map[string]interface{}
	for _, volumeI := range volumesMapI {
		m := volumeI.(map[string]interface{})
		volsMap = append(volsMap, m)
	}

	var dTier diskTier

	logging.Logger.Info(fmt.Sprintf("Initializing volumes in %v mode", mode))
	switch mode {
	case "start", "recover":
		// Delete all existing data and start fresh
		// If meta data is lost, it is better and sufficient to rely on proximity/deep scan
		// so that we can completely delete existing data and start fresh
		startVolumes(volsMap, &dTier) // will panic if right config setup is not provided
	case "restart": // Nothing is lost but sharder was off for maintenance mode
		restartVolumes(volsMap, &dTier)
	default:
		panic(fmt.Errorf("%v mode is not supported", mode))
	}

	logging.Logger.Info(fmt.Sprintf("Successfully ran volumeInit in %v mode", mode))

	logging.Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))
	var f func(volumes []*volume, prevInd int)
	diskVolumeSelectChan := make(chan selectedDiskVolume, 1)

	switch strategy {
	default:
		panic(fmt.Sprintf("strategy %v is not supported", strategy))
	case Random:
		f = func(volumes []*volume, prevInd int) {
			dTier.Mu.Lock()
			defer dTier.Mu.Unlock()

			var selectedVolume *volume
			var selectedIndex int

			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			for len(volumes) > 0 {
				ind := r.Intn(len(volumes))
				sv := volumes[ind]

				if sv.isAbleToStoreBlock() {
					selectedIndex = ind
					selectedVolume = sv
					break
				}

				unableVolumes[sv.Path] = sv
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
	case RoundRobin:
		f = func(volumes []*volume, prevInd int) {
			dTier.Mu.Lock()
			defer dTier.Mu.Unlock()

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
				if v.isAbleToStoreBlock() {
					selectedVolume = v
					selectedIndex = i

					break
				} else {
					unableVolumes[v.Path] = v
					volumes = append(volumes[:i], volumes[i+1:]...)

					if i < prevInd {
						prevInd--
					}

					i--

				}

			}

			if selectedVolume == nil {
				if prevVolume.isAbleToStoreBlock() {
					selectedVolume = prevVolume
					selectedIndex = prevInd
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
	case MinSizeFirst:
		f = func(volumes []*volume, prevInd int) {
			dTier.Mu.Lock()
			defer dTier.Mu.Unlock()

			var selectedVolume *volume
			var selectedIndex int

			totalVolumes := len(volumes)
			for i := 0; i < totalVolumes; i++ {
				if len(volumes) == 0 {
					break
				}

				v := volumes[i]
				if !v.isAbleToStoreBlock() {
					unableVolumes[v.Path] = v

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
	case MinCountFirst:
		f = func(volumes []*volume, prevInd int) {
			dTier.Mu.Lock()
			defer dTier.Mu.Unlock()

			var selectedVolume *volume
			var selectedIndex int

			totalVolumes := len(volumes)
			for i := 0; i < totalVolumes; i++ {
				if len(volumes) == 0 {
					break
				}

				v := volumes[i]
				if !v.isAbleToStoreBlock() {
					unableVolumes[v.Path] = v

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
	case FillFirst:
		f = func(volumes []*volume, prevInd int) {
			// Work to do
		}
	}

	dTier.SelectedVolumeCh = diskVolumeSelectChan
	dTier.SelectNextVolume = f
	dTier.Mu = make(Mutex, 1)

	volumesMap = make(map[string]*volume, len(dTier.Volumes))
	unableVolumes = make(map[string]*volume)

	for _, vol := range dTier.Volumes {
		volumesMap[vol.Path] = vol
	}

	go dTier.SelectNextVolume(dTier.Volumes, dTier.PrevVolInd)

	return &dTier
}

func startvolumes(mVolumes []map[string]interface{}, shouldDelete bool, dTier *diskTier) {
	for _, volI := range mVolumes {
		vPathI, ok := volI["path"]
		if !ok {
			logging.Logger.Error("Discarding volume; Path field is required")
			continue
		}

		vPath := vPathI.(string)

		var curDirInd, curKInd, curDirBlockNums int
		var totalBlocksCount, totalBlocksSize uint64
		var err error
		if shouldDelete {
			if err := os.RemoveAll(vPath); err != nil {
				logging.Logger.Error(err.Error())
				continue
			}
			if err := os.MkdirAll(vPath, 0777); err != nil {
				logging.Logger.Error(err.Error())
				continue
			}

			if err := updateCurIndexes(filepath.Join(vPath, IndexStateFileName), 0, 0); err != nil {
				logging.Logger.Error(err.Error())
				continue
			}
		} else {
			curKInd, curDirInd, err = getCurIndexes(filepath.Join(vPath, IndexStateFileName))
			if err != nil {
				logging.Logger.Error(err.Error())
				continue
			}
			bDir := filepath.Join(vPath, fmt.Sprintf("%v%v", DirPrefix, curKInd), fmt.Sprint(curDirInd))
			curDirBlockNums, err = getCurrentDirBlockNums(bDir)
			if err != nil {
				logging.Logger.Error(err.Error())
				continue
			}
			totalBlocksCount, totalBlocksSize = countBlocksInVolumes(vPath, DirPrefix, DCL)
		}

		availableSize, totalInodes, availableInodes, err := getAvailableSizeAndInodes(vPath)

		if err != nil {
			logging.Logger.Error(err.Error())
			continue
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := volI["size_to_maintain"]
		if ok {
			sizeToMaintain, err = getUint64ValueFromYamlConfig(sizeToMaintainI)
			if err != nil {
				panic(err)
			}

			sizeToMaintain *= GB
		}

		if availableSize <= sizeToMaintain {
			logging.Logger.Error(ErrSizeLimit(vPath, sizeToMaintain).Error())
			continue
		}

		var inodesToMaintain uint64
		inodesToMaintainI, ok := volI["inodes_to_maintain"]
		if ok {
			inodesToMaintain, err = getUint64ValueFromYamlConfig(inodesToMaintainI)
			if err != nil {
				panic(err)
			}
		}
		if float64(100*availableInodes)/float64(totalInodes) <= float64(inodesToMaintain) {
			logging.Logger.Error(ErrInodesLimit(vPath, inodesToMaintain).Error())
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := volI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers, err = getUint64ValueFromYamlConfig(allowedBlockNumbersI)
			if err != nil {
				panic(err)
			}
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := volI["allowed_block_size"]
		if ok {
			allowedBlockSize, err = getUint64ValueFromYamlConfig(allowedBlockSizeI)
			if err != nil {
				panic(err)
			}

			allowedBlockSize *= GB
		}

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
			CountMu:             &sync.Mutex{},
			IndMu:               &sync.Mutex{},
		})
	}

	if len(dTier.Volumes) < len(mVolumes)/2 {
		panic(ErrFiftyPercent)
	}
}

func startVolumes(volumes []map[string]interface{}, dTier *diskTier) {
	startvolumes(volumes, true, dTier)
}

func restartVolumes(volumes []map[string]interface{}, dTier *diskTier) {
	startvolumes(volumes, false, dTier)
}
