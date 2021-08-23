package blockstore

import (
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

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
	pickVolume   func(volumes []Volume) *Volume
}

const (
	Random        = "random"
	RoundRobin    = "round_robin"
	MinSizeFirst  = "min_size_first"
	MinCountFirst = "min_count_first"
)

func NewFSStore(dir string) (*FSStore, error) {
	err := os.MkdirAll(dir, 0644)
	if err != nil {
		return nil, err
	}
	return &FSStore{RootDir: dir}, nil
}

func (fs *FSStore) write() {

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
	if err := bmr.Add(); err != nil {
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
		return
	}
	switch bmr.Tiering {
	case int(WarmTier):
		//its only in warm tier
		volumePath := bmr.VolumePath
		var f *os.File
		f, err = os.Open(volumePath)
		if err != nil {
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
		go fs.addToHotTier(hash, round, data)
		return

	case int(ColdTier):
		//
	default:
		//its in hot tier
		fileName := path.Join(fs.RootDir, getSubdirName(round), fmt.Sprintf("%v.%v", hash, fileExt))
		var r *os.File
		r, err = os.Open(fileName)
		if err != nil {
			return
		}
		err = datastore.ReadJSON(r, b)
		return
	}
	return
}

func (fs *FSStore) Delete(hash string) error {
	return nil
}

// implement multiwriter; One to default path and other to among multiple disk
// implement tiering logic. probably support for s3 compatible storage server with minio
// use key-value store db to keep record of blocks. consider between bolt and rocksdb
// Check Why is size kept int64 while it cannot be negative?

func (fs *FSStore) addToHotTier(hash string, round int64, data []byte) {
	//
}
func (fs *FSStore) furtherTiering(b *block.Block, bmr *BlockMetaRecord, blockData []byte, subDir, cachePath string) {
	if len(fs.Volumes) > 0 {
		v := fs.pickVolume(fs.Volumes)
		bPath, err := v.Write(b, blockData, subDir)
		if err == nil {
			bmr.Tiering = int(HotAndWarmTier)
			bmr.VolumePath = bPath
			bmr.Add()
		}
	} else if fs.minioEnabled {
		// Add to minio if warm tiering is not available else minio can be used for cold tiering
		_, err := fs.Minio.FPutObject(fs.Minio.BucketName(), b.Hash, cachePath, minio.PutObjectOptions{})
		if err == nil {
			bmr.Tiering = int(HotAndColdTier)
			bmr.Add()
		}
	}
}

func getSubdirName(round int64) string {
	dirRoundRange := chain.GetServerChain().RoundRange
	return strconv.Itoa(int(round) / int(dirRoundRange))
}
