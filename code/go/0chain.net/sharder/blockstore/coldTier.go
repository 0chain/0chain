package blockstore

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	. "0chain.net/core/logging"
	"0chain.net/core/viper"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sys/unix"
)

const (
	Timeout             = 5
	DefaultPollInterval = 720 //Hours
)

// var cTier coldTier
var coldStoragesMap map[string]*minioClient

type selectedColdStorage struct {
	coldStorage coldStorageProvider
	prevInd     int
	err         error
}

type coldTier struct { //Cold tier
	Strategy            string
	StorageType         string //disk, minio and blobber
	ColdStorages        []coldStorageProvider
	SelectedStorageChan <-chan selectedColdStorage
	SelectNextStorage   func(coldStorageProviders []coldStorageProvider, prevInd int)
	PrevInd             int
	DeleteLocal         bool

	Mu           sync.Mutex
	PollInterval int //in hour
}

func (ct *coldTier) write(b *block.Block, data []byte) (coldPath string, err error) {
	defer func() {
		Logger.Info("Selecting next cold storage")
		go ct.SelectNextStorage(ct.ColdStorages, ct.PrevInd)
	}()

	for {
		Logger.Info("Waiting channel for selected cold storage")
		sc := <-ct.SelectedStorageChan
		if sc.err != nil {
			return "", sc.err
		}

		ct.PrevInd = sc.prevInd

		if coldPath, err = sc.coldStorage.writeBlock(b, data); err != nil {
			Logger.Error(err.Error())
			ct.removeSelectedColdStorage()
			go ct.SelectNextStorage(ct.ColdStorages, ct.PrevInd)
			continue
		}

		return

	}
}

func (ct *coldTier) read(coldPath, hash string) (io.ReadCloser, error) {
	switch ct.StorageType {
	case "minio":
		mc, ok := coldStoragesMap[coldPath]
		if !ok {
			return nil, errors.New(fmt.Sprintf("Invalid cold path %v", coldPath))
		}

		data, err := mc.getBlock(hash)
		if err != nil {
			return nil, err
		}
		return ioutil.NopCloser(bytes.NewReader(data)), nil

	case "disk":
		return os.Open(coldPath)
	}

	return nil, nil
}

func (ct *coldTier) removeSelectedColdStorage() {
	ct.ColdStorages = append(ct.ColdStorages[:ct.PrevInd], ct.ColdStorages[ct.PrevInd+1:]...)
	ct.PrevInd--
}

func (ct *coldTier) moveBlock(hash, blockPath string) (movedPath string, err error) {
	defer func() {
		Logger.Info("Selecting next cold storage")
		go ct.SelectNextStorage(ct.ColdStorages, ct.PrevInd)
	}()

	for {
		Logger.Info("Waiting for channel to get selected cold storage")
		sc := <-ct.SelectedStorageChan
		if sc.err != nil {
			return "", sc.err
		}

		ct.PrevInd = sc.prevInd

		if movedPath, err = sc.coldStorage.moveBlock(hash, blockPath); err != nil {
			Logger.Error(err.Error())
			ct.removeSelectedColdStorage()
			go ct.SelectNextStorage(ct.ColdStorages, ct.PrevInd)
			continue
		}

		if ct.DeleteLocal {
			volume := volumesMap[getVolumePathFromBlockPath(blockPath)]
			if err := volume.delete(hash, blockPath); err != nil {
				Logger.Error(fmt.Sprintf("Error occurred while deleting %v; Error: %v", blockPath, err))
				return movedPath, nil
			}
		}
		return
	}
}

type coldStorageProvider interface {
	writeBlock(b *block.Block, data []byte) (string, error)
	moveBlock(hash, blockPath string) (string, error)
	getBlock(hash string) ([]byte, error)
	getBlocks(cfo *coldFilterOptions) ([][]byte, error)
}

type coldFilterOptions struct {
	prefix    string
	startDate time.Time
	endDate   time.Time
}

//S3 compatible storage
type minioClient struct {
	*minio.Client
	storageServiceURL string
	accessId          string
	secretAccessKey   string
	bucketName        string
	useSSL            bool

	allowedBlockNumbers uint64
	allowedBlockSize    uint64
	blocksCount         uint64
	blocksSize          uint64
}

func (mc *minioClient) initialize() (err error) {
	mc.Client, err = minio.New(mc.storageServiceURL, &minio.Options{
		Creds:  credentials.NewStaticV4(mc.accessId, mc.secretAccessKey, ""),
		Secure: mc.useSSL,
	})

	if err != nil {
		Logger.Error(err.Error())
	}

	return
}

func (mc *minioClient) writeBlock(b *block.Block, data []byte) (coldPath string, err error) {
	ctx, ctxCncl := context.WithTimeout(context.Background(), Timeout)
	defer ctxCncl()

	buf := bytes.NewReader(data)

	_, err = mc.Client.PutObject(ctx, mc.bucketName, b.Hash, buf, int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		return
	}

	coldPath = fmt.Sprintf("%v:%v", mc.storageServiceURL, mc.bucketName)

	return
}

func (mc *minioClient) moveBlock(hash, blockPath string) (string, error) {
	ctx := context.Background()
	_, err := mc.Client.FPutObject(ctx, mc.bucketName, hash, blockPath, minio.PutObjectOptions{})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v:%v", mc.storageServiceURL, mc.bucketName), nil
}

func (mc *minioClient) getBlock(hash string) ([]byte, error) {
	var ctx context.Context

	ctx = context.Background()
	statCtx, statCtxCncl := context.WithTimeout(ctx, Timeout)
	defer statCtxCncl()

	objInfo, err := mc.Client.StatObject(statCtx, mc.bucketName, hash, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	getCtx, getCtxCncl := context.WithTimeout(ctx, Timeout)
	defer getCtxCncl()

	obj, err := mc.Client.GetObject(getCtx, mc.bucketName, hash, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, objInfo.Size)
	n, err := obj.Read(buffer)

	if err != nil {
		return nil, err
	}

	if n != len(buffer) {
		return nil, errors.New("dirty bytes from cloud")
	}

	return buffer, nil
}

func (mc *minioClient) getBlocks(cfo *coldFilterOptions) ([][]byte, error) {
	return nil, nil
}

// //***************************Blobber*************************************

// type blobber struct {
// 	wallet        string
// 	allocationId  string
// 	allocationObj interface{} //put appropriate type later on

// }

// func (bl *blobber) moveBlock() {
// 	//
// }

// func (bl *blobber) getBlock() {
// 	//
// }

// func (bl *blobber) getBlocks() {
// 	//
// }
// func (bl *blobber) isAbleToStoreBlock() (ableToStore bool) {
// 	return
// }

// func (bl *blobber) getAllocationObject() error {
// 	//Get an gosdk allocation object
// 	return nil
// }

//******************************Disk*******************************************
const (
	CDCL        = 10000
	CK          = "CK"
	ColdFileExt = ".dat"
)

type coldDisk struct {
	Path string

	AllowedBlockSize    uint64
	AllowedBlockNumbers uint64

	CountMu     sync.Mutex
	BlocksSize  uint64
	BlocksCount uint64

	IndMu           sync.Mutex
	CurKInd         int
	CurDirInd       int
	CurDirBlockNums int

	SizeToMaintain   uint64
	InodesToMaintain uint64
}

func (d *coldDisk) writeBlock(b *block.Block, data []byte) (blockPath string, err error) {
	return
}

func (d *coldDisk) moveBlock(hash, oldBlockPath string) (newBlockPath string, err error) {
	r, err := os.Open(oldBlockPath)
	if err != nil {
		return
	}
	defer r.Close()

	rStat, err := r.Stat()
	if err != nil {
		return
	}

	err = d.selectDir()
	if err != nil {
		return
	}

	blockPathDir := filepath.Join(d.Path, fmt.Sprintf("%v%v/%v", CK, d.CurKInd, d.CurDirInd))
	blockPath := filepath.Join(blockPathDir, fmt.Sprintf("%v%v", hash, ColdFileExt))
	f, err := os.Create(blockPath)
	if err != nil {
		return
	}

	defer f.Close()

	bf := bufio.NewWriterSize(f, 64*1024)
	w, err := zlib.NewWriterLevel(bf, zlib.BestCompression)
	if err != nil {
		return
	}

	defer w.Close()

	n, err := io.Copy(w, r)
	if err != nil {
		return
	}

	if n != rStat.Size() {
		os.Remove(blockPath)
		return "", fmt.Errorf("Could not write all data. Data length: %v, write length: %v", rStat.Size(), n)
	}

	d.CountMu.Lock()
	defer d.CountMu.Unlock()

	d.CurDirBlockNums++
	d.updateBlocksCount(1)
	d.updateBlocksSize(n)

	return blockPath, nil
}

func (d *coldDisk) getBlock(blockPath string) ([]byte, error) {
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

	return ioutil.ReadAll(r)
}

func (d *coldDisk) updateBlocksCount(i int64) {
	if i < 0 {
		d.BlocksCount -= uint64(i)
	} else {
		d.BlocksCount += uint64(i)
	}
}

func (d *coldDisk) updateBlocksSize(s int64) {
	if s < 0 {
		d.BlocksSize -= uint64(s)
	} else {
		d.BlocksSize += uint64(s)
	}
}

func (d *coldDisk) getBlocks(cfo *coldFilterOptions) ([][]byte, error) {
	return nil, nil
}

func (d *coldDisk) selectDir() error {
	if d.CurDirBlockNums < CDCL-1 {
		blocksPath := filepath.Join(d.Path, fmt.Sprintf("%v%v/%v", CK, d.CurKInd, d.CurDirInd))
		_, err := os.Stat(blocksPath)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(blocksPath, 0644); err != nil {
				return err
			}
		}
		return nil
	}

	if d.CurDirInd < CDCL-1 {
		dirInd := d.CurDirInd + 1
		blocksPath := filepath.Join(d.Path, fmt.Sprintf("%v%v/%v", CK, d.CurKInd, dirInd))
		blocksCount, err := countFiles(blocksPath)

		if err != nil && errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(blocksPath, 0644)
			if err != nil {
				return err
			}
			d.CurDirInd = dirInd
			d.CurDirBlockNums = 0
		} else if err != nil {
			return err
		}

		if blocksCount >= CDCL {
			return ErrVolumeFull(d.Path)
		}

		d.CurDirInd = dirInd
		d.CurDirBlockNums = blocksCount

		return updateCurIndexes(filepath.Join(d.Path, IndexStateFileName), d.CurKInd, d.CurDirInd)
	}

	var kInd int
	if d.CurKInd < CDCL-1 {
		kInd = d.CurKInd + 1
	} else {
		kInd = 0
	}

	dirInd := 0
	blocksPath := filepath.Join(d.Path, fmt.Sprintf("%v%v/%v", CK, kInd, dirInd))
	blocksCount, err := countFiles(blocksPath)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(blocksPath, 0644)
		if err != nil {
			return err
		}
		d.CurDirInd = dirInd
		d.CurDirBlockNums = 0

		return updateCurIndexes(filepath.Join(d.Path, IndexStateFileName), d.CurKInd, d.CurDirInd)
	} else if err != nil {
		return err
	}

	if blocksCount >= CDCL {
		return ErrVolumeFull(d.Path)
	}

	d.CurKInd = kInd
	d.CurDirInd = dirInd
	d.CurDirBlockNums = blocksCount

	return updateCurIndexes(filepath.Join(d.Path, IndexStateFileName), d.CurKInd, d.CurDirInd)
}

func (d *coldDisk) isAbleToStoreBlock() (ableToStore bool) {
	var volStat unix.Statfs_t
	err := unix.Statfs(d.Path, &volStat)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	if d.AllowedBlockSize != 0 && d.BlocksSize >= d.AllowedBlockSize {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block size. Allowed: %v, Total block size: %v", d.AllowedBlockSize, d.BlocksSize))
		return
	}

	if d.AllowedBlockNumbers != 0 && d.BlocksCount >= d.AllowedBlockNumbers {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block numbers. Allowed: %v, Total block size: %v", d.AllowedBlockNumbers, d.BlocksCount))
		return
	}

	if d.InodesToMaintain != 0 && volStat.Ffree < d.InodesToMaintain {
		Logger.Error(fmt.Sprintf("Available Inodes for volume %v is less than inodes to maintain(%v)", d.Path, d.InodesToMaintain))
		return
	}

	if d.SizeToMaintain != 0 {
		availableSize := volStat.Bfree * uint64(volStat.Bsize)
		if availableSize < uint64(d.SizeToMaintain) {
			Logger.Error(fmt.Sprintf("Available size for volume %v is less than size to maintain(%v)", d.Path, d.SizeToMaintain))
			return
		}
	}

	if unix.Access(d.Path, unix.W_OK) != nil {
		return
	}

	if err := d.selectDir(); err != nil {
		Logger.Error(ErrSelectDir(d.Path, err))
		return
	}

	return true
}

//*****************************Strategy*************************************

func coldInit(cViper *viper.Viper, mode string) *coldTier {
	storageType := cViper.GetString("storage.type")
	if storageType == "" {
		panic("Cold storage type is required")
	}

	storageTypeDiskOrMinio := fmt.Sprintf("storage.%v", storageType)

	coldStorageI := cViper.Get(storageTypeDiskOrMinio)
	if coldStorageI == nil {
		panic(fmt.Errorf("Storage type is %v but its config is not available", storageType))
	}

	storageMapViper := cViper.Sub(storageTypeDiskOrMinio)

	cTier := new(coldTier)

	selectedColdStorageChan := make(chan selectedColdStorage, 1)
	var f func(coldVolumes []coldStorageProvider, prevInd int)

	switch storageType {
	default:
		panic(fmt.Errorf("Unknown storageType %v", storageType))
	case "disk":
		volumesI := storageMapViper.Get("volumes")
		if volumesI == nil {
			panic(errors.New("Volumes Config is not available"))
		}

		volumesMapI := volumesI.([]interface{})
		var volumesMap []map[string]interface{}

		for _, volI := range volumesMapI {
			m := make(map[string]interface{})
			volIMap := volI.(map[interface{}]interface{})
			for k, v := range volIMap {
				sK := k.(string)
				m[sK] = v
			}

			volumesMap = append(volumesMap, m)
		}

		strategy := storageMapViper.GetString("strategy")
		if strategy == "" {
			strategy = DefaultColdStrategy
		}

		Logger.Info(fmt.Sprintf("Running coldInit in %v mode", mode))
		switch mode {
		default:
			panic(fmt.Errorf("%v mode is not supported", mode))
		case "start":
			startColdVolumes(volumesMap, cTier)
		case "restart":
			restartColdVolumes(volumesMap, cTier)
		case "recover":
			recoverColdVolumeMetaData(volumesMap, cTier)
		case "repair": //Metadata is present but some disk failed
			panic("Repair mode not implemented")
		case "repair_and_recover": //Metadata is lost and some disk failed
			panic("Repair and recover mode not implemented")

		}

		Logger.Info(fmt.Sprintf("Successfully ran coldInit in %v mode", mode))

		Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))

		switch strategy {
		default:
			panic(ErrStorageTypeNotSupported(strategy))
		case RoundRobin:
			f = func(coldStorageProviders []coldStorageProvider, prevInd int) {
				cTier.Mu.Lock()
				defer cTier.Mu.Unlock()
				var selectedVolume *coldDisk
				if len(coldStorageProviders) == 0 {
					selectedColdStorageChan <- selectedColdStorage{
						err: ErrUnableToSelectColdStorage,
					}
					return
				}

				prevVolume := coldStorageProviders[prevInd].(*coldDisk)
				var selectedIndex int

				if prevInd < 0 {
					prevInd = -1
				}

				for i := prevInd + 1; i != prevInd; i++ {
					if len(coldStorageProviders) == 0 {
						break
					}

					if i >= len(coldStorageProviders) {
						i = len(coldStorageProviders) - i
					}
					if i < 0 {
						i = 0
					}

					v := coldStorageProviders[i].(*coldDisk)
					if v.isAbleToStoreBlock() {
						selectedVolume = v
						selectedIndex = i

						break
					} else {
						coldStorageProviders = append(coldStorageProviders[:i], coldStorageProviders[i+1:]...)

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

				cTier.ColdStorages = coldStorageProviders

				if selectedVolume == nil {
					selectedColdStorageChan <- selectedColdStorage{
						err: ErrUnableToSelectColdStorage,
					}
				} else {
					selectedColdStorageChan <- selectedColdStorage{
						coldStorage: selectedVolume,
						prevInd:     selectedIndex,
					}
				}

			}
		}

	case "minio":
		cloudStoragesI := storageMapViper.Get("minio")
		if cloudStoragesI == nil {
			panic(errors.New("Cloud storages config is not available"))
		}

		cloudStoragesMapI := cloudStoragesI.([]interface{})
		var cloudStoragesMap []map[string]interface{}
		for _, cloudI := range cloudStoragesMapI {
			m := make(map[string]interface{})
			cloudIMap := cloudI.(map[interface{}]interface{})
			for k, v := range cloudIMap {
				sK := k.(string)
				m[sK] = v
			}

			cloudStoragesMap = append(cloudStoragesMap, m)
		}

		strategy := storageMapViper.GetString("strategy")
		if strategy == "" {
			strategy = DefaultColdStrategy
		}

		Logger.Info(fmt.Sprintf("Running coldInit in %v mode", mode))
		switch mode {
		default:
			panic(fmt.Errorf("%v mode is not supported", mode))
		case "start":
			startCloudStorages(cloudStoragesMap, cTier)
		case "restart":
			restartCloudStorages(cloudStoragesMap, cTier)
		case "recover":
			recoverCloudMetaData(cloudStoragesMap, cTier)
		case "repair":
			panic(errors.New("Repair mode not implemented"))
		case "repair_and_recover":
			panic(errors.New("Repair and recover mode not implemented"))
		}

		Logger.Info(fmt.Sprintf("Successfully ran coldInit in %v mode", mode))

		Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))

		switch strategy {
		default:
			panic(ErrStrategyNotSupported(strategy))
		case RoundRobin:
			f = func(coldStorageProviders []coldStorageProvider, prevInd int) {
				cTier.Mu.Lock()

				defer cTier.Mu.Unlock()

				var selectedCloudStorage *minioClient
				var selectedIndex int

				if prevInd < 0 {
					prevInd = -1
				}

				for i := prevInd + 1; i != prevInd; i++ {
					if len(coldStorageProviders) == 0 {
						break
					}

					if i >= len(coldStorageProviders) {
						i = len(coldStorageProviders) - i
					}
					if i < 0 {
						i = 0
					}

					selectedCloudStorage = coldStorageProviders[i].(*minioClient)
					prevInd = i
				}

				if selectedCloudStorage == nil {
					selectedColdStorageChan <- selectedColdStorage{
						err: ErrUnableToSelectColdStorage,
					}
				} else {
					selectedColdStorageChan <- selectedColdStorage{
						coldStorage: selectedCloudStorage,
						prevInd:     selectedIndex,
					}
				}
			}
		}
	}

	pollInterval := cViper.GetInt("poll_interval")
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}

	cTier.DeleteLocal = cViper.GetBool("delete_local")
	cTier.PollInterval = pollInterval
	cTier.SelectNextStorage = f
	cTier.SelectedStorageChan = selectedColdStorageChan

	Logger.Info("Selecting first cold storage")
	go cTier.SelectNextStorage(cTier.ColdStorages, cTier.PrevInd)

	return cTier
}

func startcoldVolumes(mVolumes []map[string]interface{}, cTier *coldTier, shouldDelete bool) {
	for _, volI := range mVolumes {
		vPathI, ok := volI["path"]
		if !ok {
			Logger.Error("Discarding volume; Path field is required")
			continue
		}

		vPath := vPathI.(string)

		var curKInd, curDirInd, curDirBlockNums int
		var err error

		var totalBlocksCount, totalBlocksSize uint64
		if shouldDelete {
			if err = os.RemoveAll(vPath); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err = os.MkdirAll(vPath, 0644); err != nil {
				Logger.Error(err.Error())
				continue
			}

			if err = updateCurIndexes(filepath.Join(vPath, IndexStateFileName), 0, 0); err != nil {
				Logger.Error(err.Error())
				continue
			}
		} else {
			curKInd, curDirInd, err = getCurIndexes(filepath.Join(vPath, IndexStateFileName))
			if err != nil {
				Logger.Error(err.Error())
				continue
			}

			bDir := filepath.Join(vPath, fmt.Sprintf("%v%v", CK, curKInd), fmt.Sprint(curDirInd))
			curDirBlockNums, err = getCurrentDirBlockNums(bDir)
			if err != nil {
				Logger.Error(err.Error())
				continue
			}

			totalBlocksCount, totalBlocksSize = countBlocksInVolumes(vPath, CK, CDCL)
			if err != nil {
				Logger.Error(err.Error())
				continue
			}
		}

		availableSize, totalInodes, availableInodes, err := getAvailableSizeAndInodes(vPath)

		if err != nil {
			Logger.Error(err.Error())
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
			Logger.Error(ErrSizeLimit(vPath, sizeToMaintain).Error())
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
			Logger.Error(ErrInodesLimit(vPath, inodesToMaintain).Error())
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

		cTier.ColdStorages = append(cTier.ColdStorages, &coldDisk{
			Path:                vPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			SizeToMaintain:      sizeToMaintain,
			BlocksSize:          totalBlocksSize,
			BlocksCount:         totalBlocksCount,
			CurKInd:             curKInd,
			CurDirInd:           curDirInd,
			CurDirBlockNums:     curDirBlockNums,
		})
	}

	if len(cTier.ColdStorages) < len(mVolumes)/2 {
		panic(errors.New("Atleast 50%% volumes must be able to store blocks"))
	}
}

func startColdVolumes(volumes []map[string]interface{}, cTier *coldTier) {
	startcoldVolumes(volumes, cTier, true)
}

func restartColdVolumes(volumes []map[string]interface{}, cTier *coldTier) {
	startcoldVolumes(volumes, cTier, false)
}

func recoverColdVolumeMetaData(mVolumes []map[string]interface{}, cTier *coldTier) {
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
		shouldRecoverI, ok := mVolume["recovery"]
		if ok {
			shouldRecover = shouldRecoverI.(bool)
		}

		if shouldRecover {
			for i := 0; i < CDCL; i++ {
				hotIndexPath := filepath.Join(volPath, fmt.Sprintf("%v%v", CK, i))
				if _, err := os.Stat(hotIndexPath); err != nil {
					Logger.Debug(fmt.Sprintf("Error while recovering metadata for index %v; Full path: %v; err: %v", i, hotIndexPath, err))
					continue
				}

				for j := 0; j < CDCL; j++ {
					blockSubDirPath := filepath.Join(hotIndexPath, fmt.Sprintf("%v", j))
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

			if err := updateCurIndexes(filepath.Join(volPath, IndexStateFileName), 0, 0); err != nil {
				Logger.Error(err.Error())
				continue
			}
		}

		curKInd, curDirInd, err := getCurIndexes(filepath.Join(volPath, IndexStateFileName))
		if err != nil {
			Logger.Error(err.Error())
			continue
		}
		bDir := filepath.Join(volPath, fmt.Sprintf("%v%v", CK, curKInd), fmt.Sprint(curDirInd))
		curDirBlockNums, err := getCurrentDirBlockNums(bDir)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}
		//Check available size and inodes and add volume to volume pool
		availableSize, totalInodes, availableInodes, err := getAvailableSizeAndInodes(volPath)
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

			sizeToMaintain *= GB
		}

		if availableSize <= sizeToMaintain {
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

			allowedBlockSize *= GB
		}

		if allowedBlockSize != 0 && grandCount.totalBlocksSize > allowedBlockSize {
			Logger.Error(ErrAllowedSizeLimit(volPath, allowedBlockSize).Error())
			continue
		}

		cTier.ColdStorages = append(cTier.ColdStorages, &coldDisk{
			Path:                volPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			SizeToMaintain:      sizeToMaintain,
			BlocksCount:         uint64(grandCount.totalBlocksCount),
			CurKInd:             curKInd,
			CurDirInd:           curDirInd,
			CurDirBlockNums:     curDirBlockNums,
		})
	}

	if len(cTier.ColdStorages) < len(mVolumes)/2 {
		panic(errors.New("Atleast 50%% volumes must be able to store blocks"))
	}
}

func startcloudstorages(cloudStorages []map[string]interface{}, cTier *coldTier, shouldDelete bool) {
	coldStoragesMap = make(map[string]*minioClient)
	for _, cloudStorageI := range cloudStorages {
		servUrlI, ok := cloudStorageI["storage_service_url"]
		if !ok {
			Logger.Error("Discarding cloud storage; Service url is required")
			continue
		}

		accessIdI, ok := cloudStorageI["access_id"]
		if !ok {
			Logger.Error("Discarding cloud storage; Access Id is required")
			continue
		}

		secretKeyI, ok := cloudStorageI["secret_access_key"]
		if !ok {
			Logger.Error("Discarding cloud storage; Secred Access Key is required")
			continue
		}

		bucketNameI, ok := cloudStorageI["bucket_name"]
		if !ok {
			Logger.Error("Discarding cloud storage; Bucket name is required")
			continue
		}

		servUrl := servUrlI.(string)
		accessId := accessIdI.(string)
		secretKey := secretKeyI.(string)
		bucketName := bucketNameI.(string)

		var err error
		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := cloudStorageI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers, err = getUint64ValueFromYamlConfig(allowedBlockNumbersI)
			if err != nil {
				panic(err)
			}
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := cloudStorageI["allowed_block_size"]
		if ok {
			allowedBlockSize, err = getUint64ValueFromYamlConfig(allowedBlockSizeI)
			if err != nil {
				panic(err)
			}

			allowedBlockSize *= GB
		}

		var useSSL bool
		useSSLI, ok := cloudStorageI["use_ssl"]
		if ok {
			useSSL = useSSLI.(bool)
		}

		mc := &minioClient{
			storageServiceURL:   servUrl,
			accessId:            accessId,
			secretAccessKey:     secretKey,
			bucketName:          bucketName,
			useSSL:              useSSL,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
		}

		if err := mc.initialize(); err != nil {
			Logger.Error(fmt.Sprintf("Error while initializing %v. Error: %v", servUrl, err))
			continue
		}

		if shouldDelete {
			if err := mc.Client.RemoveBucket(context.Background(), mc.bucketName); err != nil {
				Logger.Error(fmt.Sprintf("Error while removing bucket %v. Error: %v", mc.bucketName, err))
				continue
			}
		}

		coldStoragesMap[fmt.Sprintf("%v:%v", servUrl, bucketName)] = mc

		cTier.ColdStorages = append(cTier.ColdStorages, mc)
	}

	if len(cTier.ColdStorages)/2 < len(cloudStorages) {
		panic("At least 50%% cloud storages must be able to store blocks")
	}
}

func startCloudStorages(cloudStorages []map[string]interface{}, cTier *coldTier) {
	startcloudstorages(cloudStorages, cTier, true)
}

func restartCloudStorages(cloudStorages []map[string]interface{}, cTier *coldTier) {
	startcloudstorages(cloudStorages, cTier, false)
}

func recoverCloudMetaData(cloudStorages []map[string]interface{}, cTier *coldTier) { //Can run upto 100 goroutines
	guideChannel := make(chan struct{}, 10)
	wg := sync.WaitGroup{}
	for _, cloudStorageI := range cloudStorages {
		servUrlI, ok := cloudStorageI["storage_service_url"]
		if !ok {
			Logger.Error("Discarding cloud storage; Service url is required")
			continue
		}

		accessIdI, ok := cloudStorageI["access_id"]
		if !ok {
			Logger.Error("Discarding cloud storage; Access Id is required")
			continue
		}

		secretKeyI, ok := cloudStorageI["secret_access_key"]
		if !ok {
			Logger.Error("Discarding cloud storage; Secred Access Key is required")
			continue
		}

		bucketNameI, ok := cloudStorageI["bucket_name"]
		if !ok {
			Logger.Error("Discarding cloud storage; Bucket name is required")
			continue
		}

		servUrl := servUrlI.(string)
		accessId := accessIdI.(string)
		secretKey := secretKeyI.(string)
		bucketName := bucketNameI.(string)

		var err error
		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := cloudStorageI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers, err = getUint64ValueFromYamlConfig(allowedBlockNumbersI)
			if err != nil {
				panic(err)
			}
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := cloudStorageI["allowed_block_size"]
		if ok {
			allowedBlockSize, err = getUint64ValueFromYamlConfig(allowedBlockSizeI)
			if err != nil {
				panic(err)
			}

			allowedBlockSize *= GB
		}

		var useSSL bool
		useSSLI, ok := cloudStorageI["use_ssl"]
		if ok {
			useSSL = useSSLI.(bool)
		}

		mc := &minioClient{
			storageServiceURL:   servUrl,
			accessId:            accessId,
			secretAccessKey:     secretKey,
			bucketName:          bucketName,
			useSSL:              useSSL,
			allowedBlockNumbers: allowedBlockNumbers,
			allowedBlockSize:    allowedBlockSize,
		}

		if err := mc.initialize(); err != nil {
			Logger.Error(fmt.Sprintf("Error while initializing %v. Error: %v", servUrl, err))
			continue
		}

		guideChannel <- struct{}{}
		wg.Add(1)

		go func(m *minioClient) {
			defer func() {
				<-guideChannel
				wg.Done()
			}()
			recoverMetaDataFromCloudStorage(m, cTier)
		}(mc)

	}

	wg.Wait()
}

func recoverMetaDataFromCloudStorage(mc *minioClient, cTier *coldTier) {
	opts := minio.ListObjectsOptions{
		Recursive: true,
	}

	listChan := mc.Client.ListObjects(context.Background(), mc.bucketName, opts)

	recoveredCount := &struct {
		count uint64
		mu    sync.Mutex
	}{}

	guideChannel := make(chan struct{}, 10)
	wg := sync.WaitGroup{}

	for hashInfo := range listChan {
		mc.blocksCount++
		mc.blocksSize += uint64(hashInfo.Size)

		hash := hashInfo.Key

		bwr := &BlockWhereRecord{
			Hash:      hash,
			Tiering:   ColdTier,
			BlockPath: fmt.Sprintf("%v:%v", mc.storageServiceURL, mc.bucketName),
		}

		guideChannel <- struct{}{}
		wg.Add(1)

		go func(b *BlockWhereRecord) {
			defer func() {
				<-guideChannel
				wg.Done()
			}()
			if err := bwr.AddOrUpdate(); err != nil {
				Logger.Error(fmt.Sprintf("Error: %v, while adding hash %v", err, hash))
			} else {
				recoveredCount.mu.Lock()
				recoveredCount.count++
				recoveredCount.mu.Unlock()
			}
		}(bwr)

	}

	wg.Wait()

	Logger.Debug(fmt.Sprintf("Recovered %v blocks out of %v blocks from cloud: %v, bucket: %v", recoveredCount.count, mc.blocksCount, mc.storageServiceURL, mc.bucketName))

	cTier.ColdStorages = append(cTier.ColdStorages, mc)
}
