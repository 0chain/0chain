package blockstore

import "0chain.net/chaincore/block"

type Tiering uint8

const fileExt = ".dat"

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

type SmartStore struct {
	Mode     string //start or repair
	Tiering  Tiering
	HotTier  interface{} // type not defined
	WarmTier wTier
	ColdTier interface{}
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

var smartStore SmartStore

func InitializeSmartStore(configs map[string]interface{}) error {
	mode := ""
	InitMetaRecordDB()
	switch mode {
	case "start": // Clean volume paths and start storing blocks
		//
	case "repair": // Get and store missing blocks and start storing blocks
		//
	case "recover": // Recover metadata and start storing blocks
		//
	}

	return nil
}

//Each tier will have its own implementation

//Hot only

//Hot and Warm

//Hot and Cold

//Warm only

//Hot, Warm and Cold

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
