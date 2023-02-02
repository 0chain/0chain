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
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

const (
	averageBlockSize = 15 * KB
	// minimumInodesRequired is minimum inodes requirements of a disk.
	/// Here 3 is number of years and 80M is expected maximum number of block generation
	minimumInodesRequired = 3 * 80000000
	// roundRange will determine to create sub-directory inside Mprefixed directory.
	// Round 1-999 blocks will be put into "0" directory, round 1000-1999 will be put into "1" directory
	// and so on ...
	roundRange = 1000
	twoMillion = 2000000
	// mPrefix is prefix of a directory that will contain two million blocks. If round number is TwoMillion,
	// then new directory will be created as:
	// mPrefix + string(round_number/TwoMillion). So in above case "M1"
	mPrefix   = "M"
	extension = "dat.zlib"
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
	write                 func(bStore *BlockStore, hash string, rount int64, b *block.Block) error
	read                  func(bStore *BlockStore, hash string, round int64) (*block.Block, error)
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

func (bStore *BlockStore) writeToDisk(hash string, round int64, b *block.Block) error {
	bPath := filepath.Join(bStore.basePath, getBlockFilePath(hash, round))
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
	err := bStore.write(bStore, b.Hash, b.Round, b)
	if err != nil {
		return err
	}

	if b.MagicBlock != nil && b.Round == b.MagicBlock.StartingRound {
		logging.Logger.Debug("save magic block",
			zap.Int64("round", b.Round),
			zap.String("mb hash", b.MagicBlock.Hash),
		)
		return bStore.write(bStore, b.MagicBlock.Hash, b.MagicBlock.StartingRound, b)
	}
	return nil
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
	return bStore.read(bStore, bs.Hash, bs.Round)
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
		bStore.write = func(bStore *BlockStore, hash string, round int64, b *block.Block) error {
			return bStore.writeToDisk(hash, round, b)
		}

		bStore.read = func(bStore *BlockStore, hash string, round int64) (*block.Block, error) {
			return bStore.readFromDisk(hash, round)
		}
	case cViper != nil:
		bStore.write = func(bStore *BlockStore, hash string, round int64, b *block.Block) error {
			err := bStore.writeToDisk(hash, round, b)
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
