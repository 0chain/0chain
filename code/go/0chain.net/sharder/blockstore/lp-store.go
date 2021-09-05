package blockstore

import (
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	. "0chain.net/core/logging"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/datastore"
	"github.com/minio/minio-go"
)

// This struct considers that volumes are already mounted before starting sharders and storing is done
// on the basis of config options: round robin, random, etc.

type FSStore struct {
	RootDir      string
	tier         bool
	Volumes      []Volume
	minioEnabled bool
	Minio        MinioClient
	nextVolume   chan *Volume
	prevVolInd   int
	pickVolume   func(volumes *[]Volume, prevVolInd int) (*Volume, int)
}

func NewFSStore(dir, strategy string) (*FSStore, error) {
	err := os.MkdirAll(dir, 0644)
	if err != nil {
		return nil, err
	}
	volumes := checkVolumes([]string{})
	volumePicker := volumeStrategy(strategy)
	nextVolume, prevVolInd := volumePicker(&volumes, -1)

	fsStore := FSStore{
		RootDir:    dir,
		Volumes:    volumes,
		pickVolume: volumePicker,
	}

	fsStore.prevVolInd = prevVolInd
	fsStore.nextVolume <- nextVolume
	return &fsStore, nil
}

func (fs *FSStore) Write(b *block.Block) error {
	dirRoundRange := chain.GetServerChain().RoundRange
	subDir := strconv.Itoa(int(b.Round / dirRoundRange))
	bPath := path.Join(fs.RootDir, subDir, fileExt)
	cacheWriter, err := os.Create(bPath) //No compression for cache disk
	// cacheWriter, err := getCacheWriter(fs.RootDir, b)
	if err != nil {
		cacheWriter.Close()
		os.Remove(bPath)
		return err
	}
	data, err := json.Marshal(b)
	if err != nil {
		cacheWriter.Close()
		return err
	}
	_, err = cacheWriter.Write(data)
	if err != nil {
		cacheWriter.Close()
		os.Remove(bPath)
		return err
	}
	cacheWriter.Close()
	bmr := BlockMetaRecord{Hash: b.Hash, Tiering: int(HotTier)}
	if err := bmr.AddOrUpdate(); err != nil {
		os.Remove(bPath)
		return err
	}

	go fs.furtherTiering(b, &bmr, data, subDir, bPath) //May need to inform main goroutine about this goroutine
	return nil
}

func (fs *FSStore) Read(hash string, round int64) (b *block.Block, err error) {
	var bmr *BlockMetaRecord
	bmr, err = GetBlockMetaRecord(hash)
	if err != nil {
		Logger.Debug(fmt.Sprintf("Block meta record not found: hash = %v, round = %v", hash, round))
		return
	}

	switch bmr.Tiering {
	case int(WarmTier):
		//its only in warm tier
		volumePath := bmr.VolumePath
		var f *os.File
		f, err = os.Open(volumePath)
		if err != nil {
			Logger.Error(fmt.Sprintf("Block with hash = %v and round = %v not found in warm tier but its meta data found; volume path = %v", hash, round, bmr.VolumePath))
			return
		}
		defer f.Close()
		var r io.ReadCloser
		r, err = zlib.NewReader(f)
		if err != nil {
			return
		}
		var data []byte
		data, err = io.ReadAll(r)
		if err != nil {
			return
		}
		err = json.Unmarshal(data, b)
		if err != nil {
			return
		}

		//Update meta record
		bmr.LatestAccessCount++
		bmr.LastAccessTime = time.Now()
		bmr.AddOrUpdate()
		// go fs.addToHotTier(hash, round, data)

	case int(ColdTier): //its only in cold tier
		filePath := path.Join(fs.RootDir, getSubdirName(round), fmt.Sprintf("%v.%v", hash, fileExt))
		err = fs.Minio.FGetObject(fs.Minio.BucketName(), hash, filePath, minio.GetObjectOptions{})
		if err != nil {
			Logger.Error(fmt.Sprintf("Block with hash = %v and round = %v not found in cold tier but its meta data found", hash, round))
			return nil, err
		}

		var f *os.File
		f, _ = os.Open(filePath)
		defer f.Close()
		var r io.ReadCloser
		r, err = zlib.NewReader(f)
		if err != nil {
			return
		}
		var data []byte
		data, err = io.ReadAll(r)
		if err != nil {
			return
		}
		err = json.Unmarshal(data, b)
		if err != nil {
			return
		}

		//block is in hot tier now; either delete it or let it be
		bmr.LatestAccessCount++
		bmr.LastAccessTime = time.Now()
		bmr.Tiering = int(HotAndColdTier)
		bmr.AddOrUpdate()
	default:
		//its in hot tier
		fileName := path.Join(fs.RootDir, getSubdirName(round), fmt.Sprintf("%v.%v", hash, fileExt))
		var r *os.File
		r, err = os.Open(fileName)
		if err != nil {
			Logger.Error(fmt.Sprintf("Block with hash = %v and round = %v not found in hot tier but its meta data found", hash, round))
			return
		}
		err = datastore.ReadJSON(r, b)
		if err != nil {
			return
		}
		bmr.LatestAccessCount++
		bmr.LastAccessTime = time.Now()
		bmr.AddOrUpdate()
	}
	return
}

//***********************************Delete functions are kept to comply with store interface*****************************
func (fs *FSStore) Delete(hash string) error {
	return nil
}

func (fs *FSStore) DeleteBlock(b *block.Block) error {
	return nil
}

//*************************************************************************************************************************

// implement multiwriter; One to default path and other to among multiple disk
// implement tiering logic. probably support for s3 compatible storage server with minio
// use key-value store db to keep record of blocks. consider between bolt and rocksdb
// Check Why is size kept int64 while it cannot be negative?

func (fs *FSStore) addToHotTier(hash string, round int64, data []byte) {
	//
}

func (fs *FSStore) furtherTiering(b *block.Block, bmr *BlockMetaRecord, blockData []byte, subDir, cachePath string) {
	if len(fs.Volumes) > 0 {
		v, prevInd := fs.pickVolume(&fs.Volumes, fs.prevVolInd)
		fs.prevVolInd = prevInd
		bPath, err := v.Write(b, blockData)
		if err == nil {
			bmr.Tiering = int(HotAndWarmTier)
			bmr.VolumePath = bPath
			bmr.AddOrUpdate()
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

func getSubdirName(round int64) string {
	dirRoundRange := chain.GetServerChain().RoundRange
	return strconv.Itoa(int(round) / int(dirRoundRange))
}
