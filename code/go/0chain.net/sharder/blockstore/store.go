package blockstore

import (
	"0chain.net/chaincore/block"
)

/*BlockStore - an interface to read and write blocks to some storage */
type BlockStore interface {
	Write(b *block.Block) error
	Read(hash string) (*block.Block, error)
	ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error)
	Delete(hash string) error
	DeleteBlock(b *block.Block) error
	UploadToCloud(filePath string) (int64, error)
	DownloadFromCloud(filePath string) error
}

var Store BlockStore

/*GetStore - get the block store that's is setup */
func GetStore() BlockStore {
	return Store
}

/*SetupStore - Setup a file system based block storage */
func SetupStore(store BlockStore) {
	Store = store
}
