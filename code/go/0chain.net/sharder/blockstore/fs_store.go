package blockstore

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"0chain.net/chaincore/chain"
	. "0chain.net/core/logging"
	"github.com/minio/minio-go"
	"github.com/spf13/viper"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

/*FSBlockStore - a block store implementation using file system */
type FSBlockStore struct {
	RootDirectory         string
	blockMetadataProvider datastore.EntityMetadata
	Minio                 *minio.Client
}

type MinioConfiguration struct {
	StorageServiceURL string
	AccessKeyID       string
	SecretAccessKey   string
	BucketName        string
	BucketLocation    string
}

var MinioConfig MinioConfiguration

/*NewFSBlockStore - return a new fs block store */
func NewFSBlockStore(rootDir string) *FSBlockStore {
	store := &FSBlockStore{RootDirectory: rootDir}
	store.blockMetadataProvider = datastore.GetEntityMetadata("block")
	store.intializeMinio()
	return store
}

func (fbs *FSBlockStore) intializeMinio() {
	minioClient, err := minio.New(
		MinioConfig.StorageServiceURL,
		MinioConfig.AccessKeyID,
		MinioConfig.SecretAccessKey,
		viper.GetBool("minio.use_ssl"),
	)
	if err != nil {
		Logger.Panic("Unable to initiaze minio cliet", zap.Error(err))
		panic(err)
	}
	err = minioClient.MakeBucket(MinioConfig.BucketName, MinioConfig.BucketLocation)
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(MinioConfig.BucketName)
		if errBucketExists == nil && exists {
			Logger.Info("We already own ", zap.Any("bucket_name", MinioConfig.BucketName))
		} else {
			Logger.Panic("Minio bucket error", zap.Error(err), zap.Any("bucket_name", MinioConfig.BucketName))
			panic(err)
		}
	} else {
		Logger.Info(MinioConfig.BucketName + " bucket successfully created")
	}
	fbs.Minio = minioClient
}

func (fbs *FSBlockStore) getFileWithoutExtension(hash string, round int64) string {
	var file bytes.Buffer
	var dirRoundRange = chain.GetServerChain().RoundRange
	fmt.Fprintf(&file, "%s%s%v", fbs.RootDirectory, string(os.PathSeparator), round/dirRoundRange)
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&file, "%s%s", string(os.PathSeparator), hash[3*i:3*i+3])
	}
	fmt.Fprintf(&file, "%s%s", string(os.PathSeparator), hash[9:])
	return file.String()
}

func (fbs *FSBlockStore) getFileName(hash string, round int64) string {
	return fbs.getFileWithoutExtension(hash, round) + ".dat.zlib"
}

/*Write - write the block to the file system */
func (fbs *FSBlockStore) Write(b *block.Block) error {
	fileName := fbs.getFileName(b.Hash, b.Round)
	dir := filepath.Dir(fileName)
	os.MkdirAll(dir, 0755)
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	bf := bufio.NewWriterSize(f, 64*1024)
	w, _ := zlib.NewWriterLevel(bf, zlib.BestCompression)
	datastore.WriteJSON(w, b)
	w.Close()
	bf.Flush()
	f.Close()
	return nil
}

/*ReadWithBlockSummary - read the block given the block summary */
func (fbs *FSBlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	return fbs.read(bs.Hash, bs.Round)
}

/*Read - read the block from the file system */
func (fbs *FSBlockStore) Read(hash string) (*block.Block, error) {
	return nil, common.NewError("interface_not_implemented", "FSBlockStore cannot provide this interface")
}

func (fbs *FSBlockStore) read(hash string, round int64) (*block.Block, error) {
	if len(hash) != 64 {
		return nil, encryption.ErrInvalidHash
	}
	fileName := fbs.getFileName(hash, round)
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			err = fbs.DownloadFromCloud(hash, round)
			if err != nil {
				return nil, err
			}
			f, err = os.Open(fileName)
			if err != nil {
				return nil, err

			}
		} else {
			return nil, err
		}
	}
	defer f.Close()
	r, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b := fbs.blockMetadataProvider.Instance().(*block.Block)
	err = datastore.ReadJSON(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

/*Delete - delete from the hash of the block*/
func (fbs *FSBlockStore) Delete(hash string) error {
	return common.NewError("interface_not_implemented", "FSBlockStore cannote provide this interface")
}

/*DeleteBlock - delete the given block from the file system */
func (fbs *FSBlockStore) DeleteBlock(b *block.Block) error {
	fileName := fbs.getFileName(b.Hash, b.Round)
	err := os.Remove(fileName)
	if err != nil {
		return err
	}
	return nil
}

func (fbs *FSBlockStore) UploadToCloud(hash string, round int64) error {
	filePath := fbs.getFileName(hash, round)
	_, err := fbs.Minio.FPutObject(MinioConfig.BucketName, hash, filePath, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return os.Remove(filePath)
}

func (fbs *FSBlockStore) DownloadFromCloud(hash string, round int64) error {
	filePath := fbs.getFileName(hash, round)
	return fbs.Minio.FGetObject(MinioConfig.BucketName, hash, filePath, minio.GetObjectOptions{})
}
