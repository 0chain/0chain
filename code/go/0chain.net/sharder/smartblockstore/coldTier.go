package smartblockstore

import (
	"bufio"
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

	. "0chain.net/core/logging"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sys/unix"
)

var coldTier cTier
var coldStoragesMap map[string]*minioClient

type cTier struct { //Cold tier
	strategy          string
	coldStorages      []coldStorageProvider
	selectedStorage   chan coldStorageProvider
	selectNextStorage func(coldStorageProviders []coldStorageProvider, prevInd int) (coldStorageProvider, int)
	prevInd           int
	deleteLocal       bool

	mu sync.Mutex
}

func (ct *cTier) removeColdStorage(i int) {
	ct.coldStorages = append(ct.coldStorages[:i], ct.coldStorages[i+1:]...)
	ct.prevInd--
}

type coldStorageProvider interface {
	moveBlock(hash, blockPath string, deleteLocal bool) error
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

func (ct *minioClient) initialize() (err error) {
	ct.Client, err = minio.New(ct.storageServiceURL, &minio.Options{
		Creds:  credentials.NewStaticV4(ct.accessId, ct.secretAccessKey, ""),
		Secure: ct.useSSL,
	})

	if err != nil {
		Logger.Error(err.Error())
	}

	return
}

func (ct *minioClient) moveBlock(hash, blockPath string, deleteLocal bool) error {
	ctx := context.Background()
	_, err := ct.Client.FPutObject(ctx, ct.bucketName, hash, blockPath, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	if deleteLocal {
		Logger.Info(fmt.Sprintf("Removing block file: %v", blockPath))
		return os.Remove(blockPath)
	}
	return nil
}

func (ct *minioClient) getBlock(hash string) ([]byte, error) {
	ctx := context.Background()
	objInfo, err := ct.Client.StatObject(ctx, ct.bucketName, hash, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	obj, err := ct.Client.GetObject(ctx, ct.bucketName, hash, minio.GetObjectOptions{})
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

func (ct *minioClient) getBlocks(cfo *coldFilterOptions) ([][]byte, error) {
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
	ColdFileExt = "dat"
)

type coldDisk struct {
	Path string

	AllowedBlockSize    uint64
	BlocksSize          uint64
	AllowedBlockNumbers uint64
	BlocksCount         uint64

	CurKInd         int
	CurDirInd       int
	CurDirBlockNums int

	SizeToMaintain   uint64
	InodesToMaintain uint64
}

func (d *coldDisk) moveBlock(hash, oldBlockPath string, deleteLocal bool) error {
	r, err := os.Open(oldBlockPath)
	if err != nil {
		return err
	}
	defer r.Close()

	rStat, err := r.Stat()
	if err != nil {
		return err
	}

	err = d.selectDir()
	if err != nil {
		return err
	}

	blockPathDir := filepath.Join(d.Path, fmt.Sprintf("%v%v/%v", CK, d.CurKInd, d.CurDirInd))
	blockPath := filepath.Join(blockPathDir, fmt.Sprintf("%v.%v", hash, ColdFileExt))
	f, err := os.Create(blockPath)
	if err != nil {
		return err
	}

	defer f.Close()

	bf := bufio.NewWriterSize(f, 64*1024)
	w, err := zlib.NewWriterLevel(bf, zlib.BestCompression)
	if err != nil {
		return err
	}

	defer w.Close()

	n, err := io.Copy(w, r)
	if err != nil {
		return err
	}
	if n != rStat.Size() {
		os.Remove(blockPath)
		return fmt.Errorf("Could not write all data. Data length: %v, write length: %v", rStat.Size(), n)
	}

	if deleteLocal {
		Logger.Info(fmt.Sprintf("Removing block file: %v", oldBlockPath))
		return os.Remove(oldBlockPath)
	}
	return nil
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
		return nil
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

		return nil
	} else if err != nil {
		return err
	}

	if blocksCount >= CDCL {
		return ErrVolumeFull(d.Path)
	}

	d.CurKInd = kInd
	d.CurDirInd = dirInd
	d.CurDirBlockNums = blocksCount
	return nil
}

func (d *coldDisk) isAbleToStoreBlock() (ableToStore bool) {
	var volStat unix.Statfs_t
	err := unix.Statfs(d.Path, &volStat)
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	if d.BlocksSize >= d.AllowedBlockSize {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block size. Allowed: %v, Total block size: %v", d.AllowedBlockSize, d.BlocksSize))
		return
	}

	if d.BlocksCount >= d.AllowedBlockNumbers {
		Logger.Error(fmt.Sprintf("Storage limited by allowed block numbers. Allowed: %v, Total block size: %v", d.AllowedBlockNumbers, d.BlocksCount))
		return
	}

	if volStat.Ffree < d.InodesToMaintain {
		Logger.Error(fmt.Sprintf("Available Inodes for volume %v is less than inodes to maintain(%v)", d.Path, d.InodesToMaintain))
		return
	}

	availableSize := volStat.Bfree * uint64(volStat.Bsize)
	if availableSize/(1024*1024*1024) < uint64(d.SizeToMaintain) {
		Logger.Error(fmt.Sprintf("Available size for volume %v is less than size to maintain(%v)", d.Path, d.SizeToMaintain))
		return
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

// //*****************************Strategy*************************************

func coldInit(cConf map[string]interface{}, mode string) {
	storageI, ok := cConf["storage"]
	if !ok {
		panic(errors.New("Cold storage config not available"))
	}

	storage := storageI.(map[string]interface{})
	storageTypeI, ok := storage["type"]
	if !ok {
		panic(errors.New("Cold storage type is required"))
	}
	storageType := storageTypeI.(string)

	coldStorageI, ok := storage[storageType]
	if !ok {
		panic(fmt.Errorf("Storage type is %v but it config is not available", storageType))
	}

	coldStorage := coldStorageI.(map[string]interface{})

	var f func(coldVolumes []coldStorageProvider, prevInd int) (coldStorageProvider, int)

	switch storageType {
	default:
		panic(fmt.Errorf("Unknown storageType %v", storageType))
	case "disk":
		volumesI, ok := coldStorage["volumes"]
		if !ok {
			panic(errors.New("Volumes Config is not available"))
		}

		var strategy string
		strategyI, ok := coldStorage["strategy"]
		if !ok {
			strategy = DefaultColdStrategy
		} else {
			strategy = strategyI.(string)
		}

		volumes := volumesI.([]map[string]interface{})

		Logger.Info(fmt.Sprintf("Running coldInit in %v mode", mode))
		switch mode {
		default:
			panic(fmt.Errorf("%v mode is not supported", mode))
		case "start":
			startColdVolumes(volumes)
		case "restart":
			restartColdVolumes(volumes)
		case "recover":
			recoverColdVolumeMetaData(volumes)
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
			f = func(coldStorageProviders []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
				var selectedVolume *coldDisk
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

				coldTier.coldStorages = coldStorageProviders
				return selectedVolume, selectedIndex
			}
		}

	case "minio":
		cloudStoragesI, ok := coldStorage["cloud_storages"]
		if !ok {
			panic(errors.New("Cloud storages config is not available"))
		}

		var strategy string
		strategyI, ok := coldStorage["strategy"]
		if !ok {
			strategy = DefaultColdStrategy
		} else {
			strategy = strategyI.(string)
		}

		cloudStorages := cloudStoragesI.([]map[string]interface{})
		Logger.Info(fmt.Sprintf("Running coldInit in %v mode", mode))
		switch mode {
		default:
			panic(fmt.Errorf("%v mode is not supported", mode))
		case "start":
			startCloudStorages(cloudStorages)
		case "restart":
			restartCloudStorages(cloudStorages)
		case "recover":
			recoverCloudMetaData(cloudStorages)
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
			f = func(coldStorageProviders []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
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

				return selectedCloudStorage, selectedIndex
			}
		}
	}

	coldTier.selectNextStorage = f
}

func startcoldVolumes(mVolumes []map[string]interface{}, shouldDelete bool) {
	for _, volI := range mVolumes {
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

		coldTier.coldStorages = append(coldTier.coldStorages, &coldDisk{
			Path:                vPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			SizeToMaintain:      sizeToMaintain,
		})
	}

	if len(coldTier.coldStorages) < len(mVolumes)/2 {
		panic(errors.New("Atleast 50%% volumes must be able to store blocks"))
	}
}

func startColdVolumes(volumes []map[string]interface{}) {
	startcoldVolumes(volumes, true)
}

func restartColdVolumes(volumes []map[string]interface{}) {
	startcoldVolumes(volumes, false)
}

func recoverColdVolumeMetaData(mVolumes []map[string]interface{}) {
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
		Logger.Info("Completed meta data recovery")
		//Check available size and inodes and add volume to volume pool
		availableSize, availableInodes, err := getAvailableSizeAndInodes(volPath)
		if err != nil {
			Logger.Error(err.Error())
			continue
		}

		var sizeToMaintain uint64
		sizeToMaintainI, ok := mVolume["size_to_maintain"]
		if ok {
			sizeToMaintain = sizeToMaintainI.(uint64)
		}

		if availableSize/(1024^3) <= sizeToMaintain {
			Logger.Error(ErrSizeLimit(volPath, sizeToMaintain).Error())
			continue
		}

		var inodesToMaintain uint64
		inodesToMaintainI, ok := mVolume["inodes_to_maintain"]
		if ok {
			inodesToMaintain = inodesToMaintainI.(uint64)
		}
		if availableInodes <= inodesToMaintain {
			Logger.Error(ErrInodesLimit(volPath, inodesToMaintain).Error())
			continue
		}

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := mVolume["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		if allowedBlockNumbers != 0 && grandCount.totalBlocksCount > allowedBlockNumbers {
			Logger.Error(ErrAllowedCountLimit(volPath, allowedBlockNumbers).Error())
			continue
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := mVolume["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
		}

		if allowedBlockSize != 0 && grandCount.totalBlocksSize > allowedBlockSize {
			Logger.Error(ErrAllowedSizeLimit(volPath, allowedBlockSize).Error())
			continue
		}

		coldTier.coldStorages = append(coldTier.coldStorages, &coldDisk{
			Path:                volPath,
			AllowedBlockNumbers: allowedBlockNumbers,
			AllowedBlockSize:    allowedBlockSize,
			SizeToMaintain:      sizeToMaintain,
			BlocksCount:         uint64(grandCount.totalBlocksCount),
		})
	}

	if len(coldTier.coldStorages) < len(mVolumes)/2 {
		panic(errors.New("Atleast 50%% volumes must be able to store blocks"))
	}
}

func startcloudstorages(cloudStorages []map[string]interface{}, shouldDelete bool) {
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

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := cloudStorageI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := cloudStorageI["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
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

		coldTier.coldStorages = append(coldTier.coldStorages, mc)
	}

	if len(coldTier.coldStorages)/2 < len(cloudStorages) {
		panic("At least 50%% cloud storages must be able to store blocks")
	}
}

func startCloudStorages(cloudStorages []map[string]interface{}) {
	startcloudstorages(cloudStorages, true)
}

func restartCloudStorages(cloudStorages []map[string]interface{}) {
	startcloudstorages(cloudStorages, false)
}

func recoverCloudMetaData(cloudStorages []map[string]interface{}) { //Can run upto 100 goroutines
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

		var allowedBlockNumbers uint64
		allowedBlockNumbersI, ok := cloudStorageI["allowed_block_numbers"]
		if ok {
			allowedBlockNumbers = allowedBlockNumbersI.(uint64)
		}

		var allowedBlockSize uint64
		allowedBlockSizeI, ok := cloudStorageI["allowed_block_size"]
		if ok {
			allowedBlockSize = allowedBlockSizeI.(uint64)
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
			recoverMetaDataFromCloudStorage(m)
		}(mc)

	}

	wg.Wait()
}

func recoverMetaDataFromCloudStorage(mc *minioClient) {
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

	coldTier.mu.Lock()
	coldTier.coldStorages = append(coldTier.coldStorages, mc)
	coldTier.mu.Unlock()
}
