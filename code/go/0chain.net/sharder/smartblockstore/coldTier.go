package smartblockstore

var coldTier cTier

type cTier struct { //Cold tier
	strategy          string
	coldStorages      []coldStorageProvider
	selectedStorage   chan coldStorageProvider
	selectNextStorage func(coldStorageProviders []coldStorageProvider, prevInd int) (coldStorageProvider, int)
}

type coldStorageProvider interface {
	moveBlock()
	getBlock()
	getBlocks()
	isAbleToStoreBlock() bool
}

//***************************S3 compatible storage*****************

// type minio struct {
// 	storageServiceURL string
// 	accessId          string
// 	secretAccessKey   string
// 	bucketName        string
// }

// func (ct *minio) moveBlock() {
// 	//
// }
// func (ct *minio) getBlock() {
// 	//
// }
// func (ct *minio) getBlocks() {
// 	//
// }
// func (ct *minio) isAbleToStoreBlock() (ableToStore bool) {
// 	return
// }

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

// //******************************Disk*******************************************
// const (
// 	cKilo = 10000
// 	CK    = "CK"
// )

// type disk struct {
// 	path                string
// 	allowedBlockNumbers uint64
// 	allowedBlockSize    uint64
// 	availableSize       uint64
// 	blocksCount         uint64
// }

// func (d *disk) moveBlock() {
// 	//
// }

// func (d *disk) getBlock() {
// 	//
// }
// func (d *disk) getBlocks() {
// 	//
// }
// func (d *disk) selectDir() error {
// 	return nil
// }
// func (d *disk) isAbleToStoreBlock() (ableToStore bool) {
// 	return
// }

// //*****************************Strategy*************************************

// func ColdMinioStrategy(strategy string) func(coldStorageProviders []coldStorageProvider) coldStorageProvider {
// 	switch strategy {
// 	case Random:
// 		return func(coldStorageProviders []coldStorageProvider) coldStorageProvider {
// 			return &blobber{}
// 		}
// 	case RoundRobin:
// 		//
// 		return nil
// 	default:
// 		return nil
// 	}

// }

// func ColdDiskStrategy(strategy string) func(coldStorageProviders []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
// 	switch strategy {
// 	default:
// 		panic(ErrStrategyNotSupported(strategy))
// 	case Random:
// 		return func(coldVolumes []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
// 			var selectedStorage coldStorageProvider
// 			var selectedIndex int

// 			r := rand.New(rand.NewSource((time.Now().UnixNano())))

// 			for len(coldVolumes) > 0 {
// 				ind := r.Intn(len(coldVolumes))
// 				selectedStorage = coldVolumes[ind]

// 				if selectedStorage.isAbleToStoreBlock() {
// 					selectedIndex = ind
// 					break
// 				}

// 				coldVolumes[ind] = coldVolumes[len(coldVolumes)-1]
// 				coldVolumes = coldVolumes[:len(coldVolumes)-1]
// 			}

// 			coldTier.coldStorages = coldVolumes
// 			return selectedStorage, selectedIndex
// 		}
// 	case RoundRobin:
// 		return func(coldVolumes []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
// 			var selectedStorage coldStorageProvider
// 			prevStorage := coldVolumes[prevInd]
// 			var selectedIndex int

// 			if prevInd < 0 {
// 				prevInd = -1
// 			}
// 			for i := prevInd + 1; i != prevInd; i++ {
// 				if len(coldVolumes) == 0 {
// 					break
// 				}

// 				if i >= len(coldVolumes) {
// 					i = len(coldVolumes) - i
// 				}

// 				if i < 0 {
// 					i = 0
// 				}

// 				coldVolume := coldVolumes[i]
// 				if coldVolume.isAbleToStoreBlock() {
// 					selectedStorage = coldVolume
// 					selectedIndex = i
// 					break
// 				} else {
// 					coldVolumes = append(coldVolumes[:i], coldVolumes[i+1:]...)
// 					if i < prevInd {
// 						prevInd--
// 					}
// 					i--
// 				}
// 			}

// 			if selectedStorage == nil {
// 				if prevStorage.isAbleToStoreBlock() {
// 					selectedStorage = prevStorage
// 					selectedIndex = 0
// 				} else {
// 					coldVolumes = make([]coldStorageProvider, 0)
// 				}
// 			}

// 			coldTier.coldStorages = coldVolumes
// 			return selectedStorage, selectedIndex

// 		}
// 	case MinCountFirst:
// 		return func(coldVolumes []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
// 			var selectedStorage coldStorageProvider
// 			var selectedIndex int

// 			totalVolumes := len(coldVolumes)

// 			for i := 0; i < totalVolumes && len(coldVolumes) != 0; i++ {
// 				coldVolumeI := coldVolumes[i]
// 				if !coldVolumeI.isAbleToStoreBlock() {
// 					coldVolumes = append(coldVolumes[:i], coldVolumes[i+1:]...)
// 					i--
// 					totalVolumes--
// 					continue
// 				}

// 				if selectedStorage == nil {
// 					selectedStorage = coldVolumeI
// 					selectedIndex = i
// 					continue
// 				}

// 				coldVolume := coldVolumeI.(*disk)
// 				selectedVolume := selectedStorage.(*disk)

// 				if coldVolume.blocksCount < selectedVolume.blocksCount {
// 					selectedStorage = coldVolumeI
// 					selectedIndex = i
// 				}
// 			}
// 			coldTier.coldStorages = coldVolumes
// 			return selectedStorage, selectedIndex
// 		}
// 	case MinSizeFirst:
// 		return func(coldVolumes []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
// 			var selectedStorage coldStorageProvider
// 			var selectedIndex int

// 			totalVolumes := len(coldVolumes)

// 			for i := 0; i < totalVolumes && len(coldVolumes) != 0; i++ {
// 				coldVolumeI := coldVolumes[i]
// 				if !coldVolumeI.isAbleToStoreBlock() {
// 					coldVolumes = append(coldVolumes[:i], coldVolumes[i+1:]...)
// 					i--
// 					totalVolumes--
// 					continue
// 				}

// 				if selectedStorage == nil {
// 					selectedStorage = coldVolumeI
// 					selectedIndex = i
// 					continue
// 				}

// 				coldVolume := coldVolumeI.(*disk)
// 				selectedVolume := selectedStorage.(*disk)

// 				if coldVolume.availableSize > selectedVolume.availableSize {
// 					selectedStorage = coldVolumeI
// 					selectedIndex = i
// 				}
// 			}
// 			coldTier.coldStorages = coldVolumes
// 			return selectedStorage, selectedIndex
// 		}
// 	}
// 	return nil
// }

// func ColdBlobberStrategy(strategy string) func(coldStorageProviders []coldStorageProvider, prevInd int) (coldStorageProvider, int) {
// 	return nil
// }

// func coldInit(cConf map[string]interface{}) {
// 	Logger.Info("Initializing cold tiering")
// 	storageTypeI, ok := cConf["type"]
// 	if !ok {
// 		panic(errors.New("Storage type is required"))
// 	}
// 	storageType := storageTypeI.(string)

// 	switch storageType {
// 	default:
// 		panic(ErrStorageTypeNotSupported(storageType))
// 	case "disk":
// 		diskI, ok := cConf["disk"]
// 		if !ok {
// 			panic(errors.New("Provided storage type is \"disk\" but disk info not provided"))
// 		}
// 		diskConf := diskI.(map[string]interface{})
// 		stragegyI, ok := diskConf["stragegy"]

// 		var strategy string
// 		if !ok {
// 			strategy = DefaultColdStrategy
// 		} else {
// 			strategy = stragegyI.(string)
// 		}

// 		volumesI, ok := cConf["volumes"]
// 		if !ok {
// 			panic(errors.New("Volumes config not available"))
// 		}
// 		volumes := volumesI.([]map[string]interface{})
// 		checkColdVolumes(volumes)

// 		Logger.Info(fmt.Sprintf("Registering function for disk strategy: %v", strategy))

// 		coldTier.selectNextStorage = ColdDiskStrategy(strategy)
// 	case "minio":
// 		//
// 	case "blobber":
// 		blobberI, ok := cConf["blobber"]
// 		if !ok {
// 			panic(errors.New("Provided storage type is \"blobber\" but blobber info is not provided"))
// 		}
// 		blobberConf := blobberI.(map[string]interface{})
// 		strategyI, ok := blobberConf["strategy"]

// 		var strategy string
// 		if !ok {
// 			strategy = DefaultColdStrategy
// 		} else {
// 			strategy = strategyI.(string)
// 		}

// 		blobbersI, ok := cConf["blobbers"]
// 		if !ok {
// 			panic(errors.New("Blobbers config not available"))
// 		}

// 		blobbers := blobbersI.([]map[string]interface{})

// 		checkBlobbers(blobbers)

// 		Logger.Info(fmt.Sprintf("Registering function for blobber strategy: %v", strategy))
// 		coldTier.selectNextStorage = ColdBlobberStrategy(strategy)
// 	}
// }

// //Check if volume is able to store block and if able put it in an array
// //Panic if only less than 50% are able to store the block.
// func checkColdVolumes(volumes []map[string]interface{}) {
// 	//
// }

// //Check if blobber is able to store block and put it in an array
// //Panic if only less than 50% are able to store block.
// // func checkBlobbers(blobbers []map[string]interface{}) {
// // 	for _, blobber := range blobbers {
// // 		walletI, ok := blobber["wallet"]
// // 		if !ok {
// // 			Logger.Error("Wallet information required")
// // 			continue
// // 		}
// // 		wallet := walletI.(string)

// // 		allocationIdI, ok := blobber["allocation_id"]
// // 		if !ok {
// // 			Logger.Error("Allocation Id is required")
// // 			continue
// // 		}
// // 		allocationId := allocationIdI.(string)

// // 	}
// // }
