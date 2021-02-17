package blockstore

import (
	"bufio"
	"compress/zlib"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/minio/minio-go"
	"github.com/spf13/viper"

	"0chain.net/chaincore/chain"
	. "0chain.net/core/logging"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

const fileExt = ".dat.zlib"

type (
	// FSBlockStore - a block store implementation using file system.
	FSBlockStore struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
	}
)

var (
	// Make sure FSBlockStore implements BlockStore.
	_ BlockStore = (*FSBlockStore)(nil)
)

// NewFSBlockStore - return a new fs block store.
func NewFSBlockStore(rootDir string, minio MinioClient) *FSBlockStore {
	return &FSBlockStore{
		RootDirectory:         rootDir,
		blockMetadataProvider: datastore.GetEntityMetadata("block"),
		Minio:                 minio,
	}
}

func (fbs *FSBlockStore) getFileWithoutExtension(hash string, round int64) string {
	var file strings.Builder
	var dirRoundRange = chain.GetServerChain().RoundRange

	file.WriteString(fbs.RootDirectory)
	file.WriteString(string(os.PathSeparator))
	file.WriteString(strconv.Itoa(int(round / dirRoundRange)))

	for i := 0; i < 3; i++ {
		file.WriteString(string(os.PathSeparator))
		file.WriteString(hash[3*i : 3*i+3]) // FIXME panics if hash size < 9
	}

	file.WriteString(string(os.PathSeparator))
	file.WriteString(hash[9:])

	return file.String()
}

func (fbs *FSBlockStore) getFileName(hash string, round int64) string {
	return fbs.getFileWithoutExtension(hash, round) + fileExt
}

// Write - write the block to the file system
func (fbs *FSBlockStore) Write(b *block.Block) error {
	fileName := fbs.getFileName(b.Hash, b.Round)
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	bf := bufio.NewWriterSize(f, 64*1024)
	w, err := zlib.NewWriterLevel(bf, zlib.BestCompression)
	if err != nil {
		return err
	}
	if err = datastore.WriteJSON(w, b); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	if err = bf.Flush(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}

// ReadWithBlockSummary - read the block given the block summary
func (fbs *FSBlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	return fbs.read(bs.Hash, bs.Round)
}

// Read a block from the file system by its hash. Walk over round/RoundRange
// directories looking for block with given hash.
func (fbs *FSBlockStore) Read(hash string, round int64) (b *block.Block, err error) {
	// check out hash can be ""
	if len(hash) != 64 {
		return nil, common.NewError("fbs_store_read", "invalid block hash length given")
	}

	return fbs.read(hash, round)

	/*

		// for example
		// 01c/08c/7f5/4c43fb351ebc31161dd9572465ea1640b11b5629aefe3a4937f0394.dat.zlib
		var s1, s2, s3, tail = hash[0:3], hash[3:6], hash[6:9], hash[9:] + fileExt

		// walk over all 'round/RoundRange'
		err = filepath.Walk(fbs.RootDirectory,
			func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !fi.IsDir() {
					return nil
				}
				path = filepath.Join(path, s1, s2, s3, tail) // block path
				fi, err = os.Stat(path)
				if err != nil {
					if os.IsNotExist(err) {
						// can't use errors.Is(err, os.ErrNotExist) with go1.12
						return nil // not an error (continue)
					}
					return err // filesystem error
				}
				// got the file
				if b, err = fbs.read(hash, round); err != nil {
					return err
				}
				return io.EOF // ok (just stop walking loop)
			})

		if err != io.EOF {
			return // unexpected error
		}

		err = nil // reset the io.EOF

		// err is not nil doesn't mean we have the block

		if b == nil {
			return nil, os.ErrNotExist
		}

		return // got it

	*/
}

func (fbs *FSBlockStore) read(hash string, round int64) (*block.Block, error) {
	if len(hash) != 64 {
		return nil, encryption.ErrInvalidHash
	}
	fileName := fbs.getFileName(hash, round)
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			if viper.GetBool("minio.enabled") {
				err = fbs.DownloadFromCloud(hash, round)
				if err != nil {
					return nil, err
				}
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

// Delete - delete from the hash of the block
func (fbs *FSBlockStore) Delete(hash string) error {
	return common.NewError("interface_not_implemented", "FSBlockStore cannote provide this interface")
}

// DeleteBlock - delete the given block from the file system
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
	_, err := fbs.Minio.FPutObject(fbs.Minio.BucketName(), hash, filePath, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	if fbs.Minio.DeleteLocal() {
		err = os.Remove(filePath)
		if err != nil {
			Logger.Error("Failed to delete block which is moved to cloud", zap.Any("round", round), zap.Any("path", filePath))
		}
		Logger.Info("Local block successfully deleted, moved to cloud", zap.Any("round", round), zap.Any("path", filePath))
	}
	return nil
}

func (fbs *FSBlockStore) DownloadFromCloud(hash string, round int64) error {
	filePath := fbs.getFileName(hash, round)
	return fbs.Minio.FGetObject(fbs.Minio.BucketName(), hash, filePath, minio.GetObjectOptions{})
}

func (fbs *FSBlockStore) CloudObjectExists(hash string) bool {
	_, err := fbs.Minio.StatObject(fbs.Minio.BucketName(), hash, minio.StatObjectOptions{})
	if err != nil {
		return false
	}
	return true
}
