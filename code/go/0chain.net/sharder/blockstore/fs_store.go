package blockstore

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
	"golang.org/x/sys/unix"
)

const (
	averageBlockSize      = 100 * KB
	minimumInodesRequired = 5 * 365 * 24 * 3600 * 2
	roundRange            = 1000
	twoMillion            = 2000000
	mPrefix               = "M"
	extension             = "dat.zlib"
)

var (
	store BlockStoreI
)

func hasEnoughInodesAndSize(p string) error {
	var diskStat unix.Statfs_t
	err := unix.Statfs(p, &diskStat)
	if err != nil {
		return err
	}

	availableSize := diskStat.Bavail * uint64(diskStat.Bsize)
	freeInodes := diskStat.Ffree

	if freeInodes < minimumInodesRequired {
		return fmt.Errorf("insufficient inodes. Required %d, available %d",
			minimumInodesRequired, freeInodes)
	}

	requiredAvgSize := minimumInodesRequired * averageBlockSize

	if availableSize < uint64(requiredAvgSize) {
		return fmt.Errorf("insufficient disk space. Required %d, available %d",
			requiredAvgSize, availableSize)
	}
	return nil
}

func getMPrefixDir(round int64) (string, int64) {
	i := round / twoMillion
	r := round % twoMillion
	return fmt.Sprintf("%s%d", mPrefix, i), r
}

func getBlockFilePath(hash string, round int64) string {
	mPref, roundRemainder := getMPrefixDir(round)

	dirNum := roundRemainder / roundRange

	return filepath.Join(mPref, fmt.Sprint(dirNum), fmt.Sprintf("%s.%s", hash, extension))
}

type BlockStore struct {
	basePath              string
	blockMetadataProvider datastore.EntityMetadata
	write                 func(bStore *BlockStore, b *block.Block) error
	read                  func(bStore *BlockStore, hash string, round int64) (*block.Block, error)
	cache                 cacher
}

func (bStore *BlockStore) writeBlockToCache(b *block.Block) error {
	buffer := new(bytes.Buffer)
	err := datastore.WriteMsgpack(buffer, b)
	if err != nil {
		return err
	}

	return bStore.cache.Write(b.Hash, buffer.Bytes())
}

func (bStore *BlockStore) writeToDisk(b *block.Block) error {
	bPath := filepath.Join(bStore.basePath, getBlockFilePath(b.Hash, b.Round))
	err := os.MkdirAll(filepath.Dir(bPath), 0700)
	if err != nil {
		return err
	}

	f, err := os.Create(bPath)
	if err != nil {
		return err
	}
	defer f.Close()

	bf := bufio.NewWriterSize(f, 64*1024)
	w, err := zlib.NewWriterLevel(bf, zlib.BestCompression)
	if err != nil {
		return err
	}
	if err := datastore.WriteMsgpack(w, b); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return bf.Flush()
}

func (bStore *BlockStore) Write(b *block.Block) error {
	return bStore.write(bStore, b)
}

func (bStore *BlockStore) Read(hash string, round int64) (*block.Block, error) {
	return bStore.read(bStore, hash, round)
}

func (bStore *BlockStore) readFromDisk(hash string, round int64) (*block.Block, error) {
	bPath := filepath.Join(bStore.basePath, getBlockFilePath(hash, round))
	f, err := os.Open(bPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b := bStore.blockMetadataProvider.Instance().(*block.Block)
	err = datastore.ReadMsgpack(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// ReadWithBlockSummary - read the block given the block summary
func (bStore *BlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	return bStore.Read(bs.Hash, bs.Round)
}

// This function should check for minimum disk size, inodes requirement.
// Other reasonable parameters as well.
func Init(ctx context.Context, sViper *viper.Viper) {
	logging.Logger.Info("Initializing storage")

	basePath := sViper.GetString("root_dir")
	if basePath == "" {
		panic("root dir cannot be empty")
	}

	err := hasEnoughInodesAndSize(basePath)
	if err != nil {
		panic(err)
	}

	bStore := new(BlockStore)
	bStore.blockMetadataProvider = datastore.GetEntityMetadata("block")
	bStore.basePath = basePath

	cViper := sViper.Sub("cache")
	if cViper != nil {
		bStore.cache = initCache(cViper)
	}

	switch {
	default:
		bStore.write = func(bStore *BlockStore, b *block.Block) error {
			return bStore.writeToDisk(b)
		}

		bStore.read = func(bStore *BlockStore, hash string, round int64) (*block.Block, error) {
			return bStore.readFromDisk(hash, round)
		}
	case cViper != nil:
		bStore.write = func(bStore *BlockStore, b *block.Block) error {
			err := bStore.writeToDisk(b)
			if err != nil {
				return err
			}

			go func() {
				if err := bStore.writeBlockToCache(b); err != nil {
					logging.Logger.Error(err.Error())
				}
			}()

			return nil
		}

		bStore.read = func(bStore *BlockStore, hash string, round int64) (*block.Block, error) {
			b := bStore.blockMetadataProvider.Instance().(*block.Block)
			data, err := bStore.cache.Read(hash)
			if err == nil {
				r := bytes.NewReader(data)
				err = datastore.ReadMsgpack(r, b)
				if err == nil {
					return b, nil
				}
			}

			b, err = bStore.readFromDisk(hash, round)
			if err != nil {
				return nil, err
			}

			go func() {
				err := bStore.writeBlockToCache(b)
				if err != nil {
					logging.Logger.Error(err.Error())
				}
			}()
			return b, nil
		}
	}

	SetupStore(bStore)
}
