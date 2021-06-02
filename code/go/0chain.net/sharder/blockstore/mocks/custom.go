package mocks

import (
	"errors"

	"0chain.net/chaincore/block"
	"0chain.net/sharder/blockstore"
)

type BlockStoreCustom struct {
	cloud  map[string]struct{} // map to store cloud objects
	blocks map[string]*block.Block
}

var (
	_ blockstore.BlockStore = (*BlockStoreCustom)(nil)
)

func NewBlockStoreMock() *BlockStoreCustom {
	return &BlockStoreCustom{
		cloud:  make(map[string]struct{}),
		blocks: make(map[string]*block.Block),
	}
}

func (b2 BlockStoreCustom) Write(b *block.Block) error {
	if len(b.Hash) != 64 {
		return errors.New("hash must be 64 size")
	}
	b2.blocks[b.Hash] = b
	return nil
}

func (b2 BlockStoreCustom) Read(hash string, _ int64) (*block.Block, error) {
	v, ok := b2.blocks[hash]
	if !ok {
		return nil, errors.New("unknown block")
	}
	return v, nil
}

func (b2 BlockStoreCustom) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	v, ok := b2.blocks[bs.Hash]
	if !ok {
		return nil, errors.New("unknown block")
	}
	return v, nil
}

func (b2 BlockStoreCustom) Delete(hash string) error {
	if len(hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	return nil
}

func (b2 BlockStoreCustom) DeleteBlock(b *block.Block) error {
	if len(b.Hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	return nil
}

func (b2 BlockStoreCustom) UploadToCloud(hash string, _ int64) error {
	if len(hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	b2.cloud[hash] = struct{}{}
	return nil
}

func (b2 BlockStoreCustom) DownloadFromCloud(_ string, _ int64) error {
	return nil
}

func (b2 BlockStoreCustom) CloudObjectExists(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	_, ok := b2.cloud[hash]
	return ok
}
