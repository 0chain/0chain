package blockstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/0chain/common/core/logging"

	"0chain.net/core/viper"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	Timeout = time.Minute
)

var coldStoragesMap map[string]coldStorageProvider

type selectedColdStorage struct {
	coldStorage coldStorageProvider
	prevInd     int
	err         error
}

// coldTier manages all the cold storages with the coldStorageProvider interface
// Currently only minio compatible server is supported.
// We can obviously extend this to magnetic tapes(a much slower, cheaper and durable)
// storage, blobber, etc.
type coldTier struct { //Cold tier
	// Strategy: How to select next storage to store cold blocks.
	Strategy     string
	ColdStorages []coldStorageProvider
	// SelectedStorageChan will provide channel for selected storage
	// as per strategy
	StorageSelectorChan <-chan selectedColdStorage
	// SelectNextStorage will select storage based on strategy and put
	// the selected storage in SelectedStorageChan channel
	SelectNextStorage func(coldStorageProviders []coldStorageProvider, prevInd int)
	// PrevInd is index of previously selected storage
	PrevInd int
	// DeleteLocal: Either to delete local file
	DeleteLocal bool

	// Mu: Mutex used to select storage or remove storage from the list
	Mu Mutex
}

func (ct *coldTier) read(coldPath, hash string) ([]byte, error) {
	mc, ok := coldStoragesMap[coldPath]
	if !ok {
		return nil, fmt.Errorf("Invalid cold path %v", coldPath)
	}

	return mc.getBlock(hash)
}

func (ct *coldTier) moveBlock(hash, blockPath string) (movedPath string, err error) {
	defer func() {
		logging.Logger.Info("Selecting next cold storage")
		go ct.SelectNextStorage(ct.ColdStorages, ct.PrevInd)
	}()

	for {
		logging.Logger.Info("Waiting for channel to get selected cold storage")
		sc := <-ct.StorageSelectorChan
		if sc.err != nil {
			return "", sc.err
		}

		ct.PrevInd = sc.prevInd

		if movedPath, err = sc.coldStorage.moveBlock(hash, blockPath); err != nil {
			logging.Logger.Error(err.Error())
			ct.removeSelectedColdStorage()
			go ct.SelectNextStorage(ct.ColdStorages, ct.PrevInd)
			continue
		}

		if ct.DeleteLocal {
			volume := volumesMap[getVolumePathFromBlockPath(blockPath)]
			if err := volume.delete(hash, blockPath); err != nil {
				logging.Logger.Error(fmt.Sprintf("Error occurred while deleting %v; Error: %v", blockPath, err))
				return movedPath, nil
			}
		}
		return
	}
}

func (ct *coldTier) removeSelectedColdStorage() {
	if !ct.Mu.TryLock() {
		return
	}
	defer ct.Mu.Unlock()

	ct.ColdStorages = append(ct.ColdStorages[:ct.PrevInd], ct.ColdStorages[ct.PrevInd+1:]...)
	ct.PrevInd--
}

type coldStorageProvider interface {
	moveBlock(hash, blockPath string) (string, error)
	getBlock(hash string) ([]byte, error)
}

type minioClient struct {
	*minio.Client
	storageServiceURL string
	accessId          string
	secretAccessKey   string
	bucketName        string
	useSSL            bool

	allowedBlockNumbers uint64
	allowedBlockSize    uint64

	countMu     *sync.Mutex
	blocksCount uint64
	blocksSize  uint64
}

func (mc *minioClient) initialize(delete bool) (err error) {
	mc.Client, err = minio.New(mc.storageServiceURL, &minio.Options{
		Creds:  credentials.NewStaticV4(mc.accessId, mc.secretAccessKey, ""),
		Secure: mc.useSSL,
	})

	if err != nil {
		logging.Logger.Error(err.Error())
		return err
	}

	if delete {
		err := mc.deleteAll()
		if err != nil {
			return err
		}
	}
	return mc.calculateBucketStats()
}

func (mc *minioClient) calculateBucketStats() error {
	ctx := context.Background()
	ch := mc.Client.ListObjects(ctx, mc.bucketName, minio.ListObjectsOptions{})

	for obj := range ch {
		if obj.Err != nil {
			return obj.Err
		}
		// Not required to lock countMU
		mc.blocksCount++
		mc.blocksSize += uint64(obj.Size)
	}
	return nil
}

func (mc *minioClient) deleteAll() error {
	ctx := context.Background()
	ch := mc.Client.ListObjects(ctx, mc.bucketName, minio.ListObjectsOptions{})

	errCh := mc.Client.RemoveObjects(ctx, mc.bucketName, ch, minio.RemoveObjectsOptions{})

	removeErr := <-errCh
	if removeErr.Err != nil {
		return fmt.Errorf("Error: %s, object name: %s", removeErr.Err.Error(), removeErr.ObjectName)
	}

	return nil
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
	ctx := context.Background()
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

func initCold(cViper *viper.Viper, mode string) *coldTier {

	cloudStoragesI := cViper.Get("cloud_storages")
	if cloudStoragesI == nil {
		panic("Config is not available")
	}

	cTier := new(coldTier)

	storageSelectorChan := make(chan selectedColdStorage, 1)
	var f func(coldVolumes []coldStorageProvider, prevInd int)

	cloudStoragesMapI := cloudStoragesI.([]interface{})
	var cloudStoragesMap []map[string]interface{}
	for _, cloudI := range cloudStoragesMapI {
		m := cloudI.(map[string]interface{})
		cloudStoragesMap = append(cloudStoragesMap, m)
	}

	strategy := cViper.GetString("strategy")
	if strategy == "" {
		strategy = DefaultColdStrategy
	}

	switch mode {
	default:
		panic(fmt.Sprintf("%v mode is not supported", mode))
	case "start", "recover":
		startCloudStorages(cloudStoragesMap, cTier, true)
	case "restart":
		startCloudStorages(cloudStoragesMap, cTier, false)
	}

	logging.Logger.Info(fmt.Sprintf("Successfully ran coldInit in %v mode", mode))

	logging.Logger.Info(fmt.Sprintf("Registering function for strategy: %v", strategy))

	switch strategy {
	default:
		panic(ErrStrategyNotSupported(strategy))
	case RoundRobin:
		f = getColdRBStrategyFunc(cTier, storageSelectorChan)
	}

	cTier.DeleteLocal = cViper.GetBool("delete_local")
	cTier.SelectNextStorage = f
	cTier.Mu = make(Mutex, 1)
	cTier.StorageSelectorChan = storageSelectorChan

	logging.Logger.Info("Selecting first cold storage")
	go cTier.SelectNextStorage(cTier.ColdStorages, cTier.PrevInd)

	return cTier
}

func startCloudStorages(cloudStorages []map[string]interface{},
	cTier *coldTier, shouldDelete bool) {

	coldStoragesMap = make(map[string]coldStorageProvider)

	wg := &sync.WaitGroup{}
	coldMu := &sync.Mutex{}

	for _, cloudStorageI := range cloudStorages {
		wg.Add(1)
		go func(cloudStorageI map[string]interface{}) {
			defer wg.Done()

			servUrlI, ok := cloudStorageI["storage_service_url"]
			if !ok {
				logging.Logger.Error("Discarding cloud storage; Service url is required")
				return
			}

			accessIdI, ok := cloudStorageI["access_id"]
			if !ok {
				logging.Logger.Error("Discarding cloud storage; Access Id is required")
				return
			}

			secretKeyI, ok := cloudStorageI["secret_access_key"]
			if !ok {
				logging.Logger.Error("Discarding cloud storage; Secret Access Key is required")
				return
			}

			bucketNameI, ok := cloudStorageI["bucket_name"]
			if !ok {
				logging.Logger.Error("Discarding cloud storage; Bucket name is required")
				return
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
				countMu:             &sync.Mutex{},
			}

			if err := mc.initialize(shouldDelete); err != nil {
				logging.Logger.Error(fmt.Sprintf("Error while initializing %v. Error: %v", servUrl, err))
				return
			}

			if mc.blocksCount >= mc.allowedBlockNumbers {
				logging.Logger.Debug(
					fmt.Sprintf("%v:%v has reached its blocks number limit. Has %v, Allowed: %v ",
						mc.storageServiceURL, mc.bucketName, mc.blocksCount, mc.allowedBlockNumbers))
				return
			}

			if mc.blocksSize >= mc.allowedBlockSize {
				logging.Logger.Debug(
					fmt.Sprintf("%v:%v has reached its blocks size limit. Has %v, Allowed: %v ",
						mc.storageServiceURL, mc.bucketName, mc.blocksSize, mc.allowedBlockSize))
				return
			}

			coldMu.Lock()
			coldStoragesMap[fmt.Sprintf("%v:%v", servUrl, bucketName)] = mc
			cTier.ColdStorages = append(cTier.ColdStorages, mc)
			coldMu.Unlock()
		}(cloudStorageI)
	}

	wg.Wait()
	if len(cTier.ColdStorages) == 0 || len(cTier.ColdStorages) < len(cloudStorages)/2 {
		panic("At least 50%% cloud storages must be able to store blocks")
	}
}

func getColdRBStrategyFunc(
	cTier *coldTier,
	ch chan selectedColdStorage,
) func(
	coldStorageProviders []coldStorageProvider,
	prevInd int,
) {

	return func(coldStorageProviders []coldStorageProvider, prevInd int) {
		cTier.Mu.Lock()
		defer cTier.Mu.Unlock()

		var selectedStorage coldStorageProvider

		if prevInd < 0 {
			prevInd = -1
		}
		i := prevInd + 1
		if len(coldStorageProviders) > 0 {
			if i >= len(coldStorageProviders) {
				i = len(coldStorageProviders) - i
			}
			if i < 0 {
				i = 0
			}

			selectedStorage = coldStorageProviders[i]
			prevInd = i
		}

		if selectedStorage == nil {
			ch <- selectedColdStorage{
				err: ErrUnableToSelectColdStorage,
			}
		} else {
			ch <- selectedColdStorage{
				coldStorage: selectedStorage,
				prevInd:     prevInd,
			}
		}
	}
}
