package blockstore

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

const (
	averageBlockSize = 15 * KB
	// minimumInodesRequired is minimum inodes requirements of a disk.
	/// Here 3 is number of years and 80M is expected maximum number of block generation
	expectedTotalBlocksIn3Years = 3 * 80000000
	extension                   = "dat.zlib"
	// subDirs will determine the number of subdirs that should be created to store a block.
	// For example if a block hash is `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
	// and subDirs is 5, then block's path will be:
	//
	// BasePath/e/3/b/0/c/44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.dat.zlib
	// subDirs 5 will create total of 16^5 + 16^4 + ..+ 16 =  1118480 directories which will store all the finalized
	// blocks.
	subDirs = 5
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
	minimumInodesRequired := uint64(expectedTotalBlocksIn3Years)
	for i := 0; i < subDirs; i++ {
		minimumInodesRequired += uint64(math.Pow(16, float64(i+1)))
	}

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

func getBlockFilePath(hash string) string {
	var s string
	for i := 0; i < subDirs; i++ {
		s += string(hash[i]) + string(os.PathSeparator)
	}
	return filepath.Join(s, fmt.Sprintf("%s.%s", hash[subDirs:], extension))
}

type BlockStore struct {
	basePath              string
	blockMetadataProvider datastore.EntityMetadata
	write                 func(bStore *BlockStore, hash string, b *block.Block) error
	read                  func(bStore *BlockStore, hash string) (*block.Block, error)
	cache                 cacher
}

func (bStore *BlockStore) writeBlockToCache(hash string, b *block.Block) error {
	buffer := new(bytes.Buffer)
	err := datastore.WriteMsgpack(buffer, b)
	if err != nil {
		return err
	}

	return bStore.cache.Write(hash, buffer.Bytes())
}

func (bStore *BlockStore) writeToDisk(hash string, b *block.Block) error {
	bPath := filepath.Join(bStore.basePath, getBlockFilePath(hash))
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
	err := bStore.write(bStore, b.Hash, b)
	if err != nil {
		return err
	}

	if b.MagicBlock != nil && b.Round == b.MagicBlock.StartingRound {
		logging.Logger.Debug("save magic block",
			zap.Int64("round", b.Round),
			zap.String("mb hash", b.MagicBlock.Hash),
		)
		return bStore.write(bStore, b.MagicBlock.Hash, b)
	}
	return nil
}

func (bStore *BlockStore) Read(hash string) (*block.Block, error) {
	return bStore.read(bStore, hash)
}

func (bStore *BlockStore) readFromDisk(hash string) (*block.Block, error) {
	bPath := filepath.Join(bStore.basePath, getBlockFilePath(hash))
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
	return bStore.read(bStore, bs.Hash)
}

// Init checks for minimum disk size, inodes requirement and assigns
// block storer to a variable. If any error occurs during initialization
// it will panic.
// It will register read and write function based on whether cache config is provided or not.
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
		bStore.write = func(bStore *BlockStore, hash string, b *block.Block) error {
			return bStore.writeToDisk(hash, b)
		}

		bStore.read = func(bStore *BlockStore, hash string) (*block.Block, error) {
			return bStore.readFromDisk(hash)
		}
	case cViper != nil:
		bStore.write = func(bStore *BlockStore, hash string, b *block.Block) error {
			err := bStore.writeToDisk(hash, b)
			if err != nil {
				return err
			}

			go func() {
				if err := bStore.writeBlockToCache(hash, b); err != nil {
					logging.Logger.Error(err.Error())
				}
			}()

			return nil
		}

		bStore.read = func(bStore *BlockStore, hash string) (*block.Block, error) {
			b := bStore.blockMetadataProvider.Instance().(*block.Block)
			data, err := bStore.cache.Read(hash)
			if err == nil {
				r := bytes.NewReader(data)
				err = datastore.ReadMsgpack(r, b)
				if err == nil {
					return b, nil
				}
			}

			b, err = bStore.readFromDisk(hash)
			if err != nil {
				return nil, err
			}

			go func() {
				err := bStore.writeBlockToCache(b.Hash, b)
				if err != nil {
					logging.Logger.Error(err.Error())
				}
			}()
			return b, nil
		}
	}

	SetupStore(bStore)
}
