//cache
package smartblockstore

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	. "0chain.net/core/logging"
)

const (
	Mfu      = "mfu"   //most frequently used
	Ra       = "ra"    //recently added
	RaAndMfu = "ramfu" //recently added and most frequently used

	ChK   = "ChK"
	ChDCL = 2000

	WriteThrough = "writethrough"
	WriteBack    = "writeback"
)

var chTier cacheTier

type cacheTier struct {
	CacheWrite        string //writethrough, writeback
	SelectedVolumeCh  <-chan *volumeSelected
	Volumes           []*cacheVolume
	SelectNextStorage func(volumes []*cacheVolume, prevInd int)
	CacheBlock        func(hash string, data []byte) error
	RemoveBlock       func()
	prevVolInd        int
	DirPrefix         string
	DCL               int
	Mu                sync.Mutex
}

type volumeSelected struct {
	volume  *cacheVolume
	prevInd int
	err     error
}

func (ct *cacheTier) write(hash string, data []byte) (cachePath string, err error) {
	sv := <-ct.SelectedVolumeCh
	chTier.prevVolInd = sv.prevInd

	if sv.err != nil {
		Logger.Error(sv.err.Error())
		return "", sv.err
	}

	if cachePath, err = sv.volume.write(hash, data); err != nil {
		return
	}

	//TODO /sif cache is full error; prune cache
	go ct.SelectNextStorage(ct.Volumes, ct.prevVolInd)
	return
}

type cacheVolume struct {
	Path                string
	SizeToMaintain      uint64
	InodesToMaintain    uint64
	AllowedBlockNumbers uint64
	AllowedBlockSize    uint64
	BlocksCount         uint64
	BlocksSize          uint64
	//This field will determine when to poll and clean cache's blocks.
	PollInterval int
}

type cacheInfo struct {
	Hash                  string
	BlockCreateTime       time.Time
	BlockLatestAccessTime time.Time
	BlockAccessCount      uint
}

func (v *cacheVolume) isAbleToStoreBlock() (ableToStore bool) {
	return true
}

func (v *cacheVolume) write(hash string, data []byte) (cachePath string, err error) {
	if err != nil {
		//log error
	}
	return
}

func deleteFromCache() {
	//
}

//Check for old blocks and clean cache
func pollCache() {
	//
}

func getFromCache() (*block.Block, error) {
	//
	return nil, nil
}

func cacheInit(cConf map[string]interface{}) {
	Logger.Info("Initializing cache")
	volumesI, ok := cConf["cache"]
	if !ok {
		panic("volume config not available")
	}

	volumes := volumesI.([]map[string]interface{})

	var volumeStrategy string
	volumeStrategyI, ok := cConf["volume_strategy"]
	if !ok {
		volumeStrategy = DefaultCacheStrategy
	} else {
		volumeStrategy = volumeStrategyI.(string)
	}

	var cacheStrategy string
	cacheStrategyI, ok := cConf["cache_strategy"]
	if !ok {
		cacheStrategy = Ra
	} else {
		cacheStrategy = cacheStrategyI.(string)
	}

	var cacheWrite string
	cacheWriteI, ok := cConf["cache_write"]
	if !ok {
		cacheWrite = WriteBack
	} else {
		cacheWrite = cacheWriteI.(string)
	}

	if cacheWrite != WriteBack && cacheWrite != WriteThrough {
		panic(fmt.Errorf("Cache write policy %v is not supported", cacheWrite))
	}

	Logger.Info(fmt.Sprintf("Registering function for volume strategy: %v", volumeStrategy))
	var vf func(volumes []*cacheVolume, prevInd int)

	selectedVolumeChan := make(chan *volumeSelected, 1)

	switch volumeStrategy {
	default:
		panic(ErrStrategyNotSupported(volumeStrategy))
	case Random:
		vf = func(volumes []*cacheVolume, prevInd int) {
			var selectedVolume *cacheVolume
			var selectedIndex int

			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			for len(volumes) > 0 {
				ind := r.Intn(len(volumes))
				sv := volumes[ind]

				if sv.isAbleToStoreBlock() {
					selectedVolume = sv
					selectedIndex = ind
					break
				}

				volumes[ind] = volumes[len(volumes)-1]
				volumes = volumes[:len(volumes)-1]
			}

			chTier.Volumes = volumes

			if selectedVolume == nil {
				selectedVolumeChan <- &volumeSelected{
					err: ErrUnableToSelectVolume,
				}

			} else {

				selectedVolumeChan <- &volumeSelected{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}

		}
	case RoundRobin:
		vf = func(volumes []*cacheVolume, prevInd int) {
			var selectedVolume *cacheVolume
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
					selectedIndex = 0
				}
			}

			chTier.Volumes = volumes
			if selectedVolume == nil {
				selectedVolumeChan <- &volumeSelected{
					err: ErrUnableToSelectVolume,
				}
			} else {
				selectedVolumeChan <- &volumeSelected{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}
		}
	case MinSizeFirst:
		vf = func(volumes []*cacheVolume, prevInd int) {
			var selectedVolume *cacheVolume
			var selectedIndex int

			totalVolumes := len(volumes)
			for i := 0; i < totalVolumes; i++ {
				if len(volumes) == 0 {
					break
				}

				v := volumes[i]
				if !v.isAbleToStoreBlock() {
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

			chTier.Volumes = volumes
			if selectedVolume == nil {
				selectedVolumeChan <- &volumeSelected{
					err: ErrUnableToSelectVolume,
				}
			} else {
				selectedVolumeChan <- &volumeSelected{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}
		}
	case MinCountFirst:
		vf = func(volumes []*cacheVolume, prevInd int) {
			var selectedVolume *cacheVolume
			var selectedIndex int

			totalVolumes := len(volumes)
			for i := 0; i < totalVolumes; i++ {
				if len(volumes) == 0 {
					break
				}

				v := volumes[i]
				if !v.isAbleToStoreBlock() {
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

			chTier.Volumes = volumes
			if selectedVolume == nil {
				selectedVolumeChan <- &volumeSelected{
					err: ErrUnableToSelectVolume,
				}
			} else {
				selectedVolumeChan <- &volumeSelected{
					volume:  selectedVolume,
					prevInd: selectedIndex,
				}
			}
		}
	}

	Logger.Info(fmt.Sprintf("Registering function for cache strategy: %v", cacheStrategy))

	var cf func()

	switch cacheStrategy {
	default:
		panic(ErrStrategyNotSupported(cacheStrategy))
	case Mfu:
		_ = cf
		//
	case Ra:
		//
	case RaAndMfu:
		//
	}

	startCacheVolumes(volumes)

	chTier.SelectedVolumeCh = selectedVolumeChan
	chTier.SelectNextStorage = vf
	chTier.DCL = ChDCL
	chTier.DirPrefix = ChK
}

func startCacheVolumes(mVolumes []map[string]interface{}) {
	for _, volI := range mVolumes {
		vPathI, ok := volI["path"]
		if !ok {
			Logger.Error("Discarding volume; Path field is required")
			continue
		}

		vPath := vPathI.(string)

		if err := os.RemoveAll(vPath); err != nil {
			Logger.Error(err.Error())
			continue
		}

		if err := os.MkdirAll(vPath, 0644); err != nil {
			Logger.Error(err.Error())
			continue
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

		chTier.Volumes = append(chTier.Volumes, &cacheVolume{
			Path:                vPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			SizeToMaintain:      sizeToMaintain,
		})
	}

	if len(chTier.Volumes) < len(mVolumes)/2 {
		panic(ErrFiftyPercent)
	}

}

/*


CreatedTime = hash
LastAccessTime = hash
FrequentAccessCount = hash
CreatedTime:AccessCount = hash


*/
