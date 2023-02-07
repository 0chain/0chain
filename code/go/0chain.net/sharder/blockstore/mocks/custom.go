package mocks

import (
	"errors"

	"0chain.net/chaincore/block"
	"0chain.net/sharder/blockstore"
)

type BlockStoreCustom struct {
	blocks map[string]*block.Block
}

var (
	_ blockstore.BlockStoreI = (*BlockStoreCustom)(nil)
)

func NewBlockStoreMock() *BlockStoreCustom {
	return &BlockStoreCustom{
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

func (b2 BlockStoreCustom) Read(hash string) (*block.Block, error) {
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
