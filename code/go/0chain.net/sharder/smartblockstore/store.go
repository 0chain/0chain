package smartblockstore

import (
	"errors"

	"0chain.net/chaincore/block"
)

type Tiering uint8

const (
	H   Tiering = iota // Hot only
	W                  // Warm only
	HW                 // Hot and Warm
	C                  // Cold only but only cold tiering is not used.
	HC                 // Hot and cold
	WC                 // Warm and Cold
	HWC                // Hot, Warm and Cold
)

/*
1. Use only hot tier
2. Use only warm tier
3. Use hot and cold tier
4. Use hot and warm tier
5. Use warm and cold tier
6. Use hot, warm and cold tier
*/

var smartStore SmartStore

type SmartStore struct {
	CacheEnabled bool
	Cache        cache
	Tiering      Tiering
	HotTier      hTier
	WarmTier     wTier
	ColdTier     cTier
	//fields with registered functions as per the config files
	write  func(b *block.Block) error
	read   func(hash string, round int64) (b *block.Block, err error)
	delete func(hash string) error
}

func (sm *SmartStore) Write(b *block.Block) error {
	return sm.write(b)
}

func (sm *SmartStore) Read(hash string, round int64) (b *block.Block, err error) {
	return sm.read(hash, round)
}

func (sm *SmartStore) Delete(hash string) error {
	return nil // Not implemented
}

//TODO provide only one path for one volume(partition)

func InitializeSmartStore(sConf map[string]interface{}) error {
	InitMetaRecordDB()
	var mode, storageType string

	storageTypeI, ok := sConf["storage_type"]
	if !ok {
		panic(errors.New("Storage Type is a required field"))
	}
	storageType = storageTypeI.(string)

	modeI, ok := sConf["mode"]
	if !ok {
		mode = "start"
	} else {
		mode = modeI.(string)
	}

	switch mode {
	case "start": // Clean volume paths and start storing blocks
		//
	case "repair": // Get and store missing blocks and start storing blocks
		//
	case "recover": // Recover metadata and start storing blocks
		//
	}

	switch storageType {
	case "hot_only":
		hotI, ok := sConf["hot"]
		if !ok {
			panic(errors.New("Storage type includes hot tier but hot tier config not provided"))
		}
		hotMap := hotI.(map[string]interface{})

		hotInit(hotMap)

	// case W:
	// 	//
	// case HW:
	// 	//
	// case HC:
	// 	//
	// case WC:
	// 	//
	// case HWC:
	//
	default:
		panic(errors.New("Unknown Tiering"))
	}

	return nil
}

//Each tier will have its own implementation

//Hot only
func hotOnly() {

	//SetUpHotVolumes
}

//Hot and Warm
func hotAndWarm() {
	//SetupHotAndWarmVolumes
}

//Hot and Cold
func hotAndCold() {
	//Setup hot and cold tiering
}

//Warm only
func warmOnly() {
	//SetUp Warm volumes
}

//Hot, Warm and Cold
func hotWarmAndCold() {
	//Setup hot warm and cold tiering
}

//Possibly usable code
/*
func (fs *FSStore) furtherTiering(b *block.Block, bmr *BlockMetaRecord, blockData []byte, subDir, cachePath string) {
	if len(fs.Volumes) > 0 {
		ableVolumes := make([]Volume, len(fs.Volumes))
		copy(ableVolumes, fs.Volumes)
		for {
			if len(ableVolumes) == 0 {
				//log error
				//stop sharder, panic
			}
			v, prevInd := fs.pickVolume(&ableVolumes, fs.prevVolInd)
			if v == nil {
				//Log error
				//stop sharder, panic
			}
			fs.prevVolInd = prevInd
			bPath, err := v.Write(b, blockData)
			if err == nil {
				bmr.Tiering = int(HotAndWarmTier)
				bmr.VolumePath = bPath
				bmr.AddOrUpdate()
				break
			} else {
				ableVolumes[prevInd] = ableVolumes[len(ableVolumes)-1]
				ableVolumes = ableVolumes[:len(ableVolumes)-1]
				fs.prevVolInd--
			}

		}
	} else if fs.minioEnabled {
		// Add to minio if warm tiering is not available else minio can be used for cold tiering
		_, err := fs.Minio.FPutObject(fs.Minio.BucketName(), b.Hash, cachePath, minio.PutObjectOptions{})
		if err == nil {
			bmr.Tiering = int(HotAndColdTier)
			bmr.AddOrUpdate()
		}
	}
}

*/
