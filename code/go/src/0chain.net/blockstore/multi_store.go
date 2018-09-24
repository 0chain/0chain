package blockstore

import "0chain.net/block"

//MultiBlockStore - a block store backed by multiple other block stores - useful to experiment different block stores
type MultiBlockStore struct {
	BlockStores []BlockStore
}

//NewMultiBlockStore - create a new multi block store
func NewMultiBlockStore(blockstores []BlockStore) *MultiBlockStore {
	mbs := &MultiBlockStore{BlockStores: blockstores}
	return mbs
}

//Write - implement interface
func (mbs *MultiBlockStore) Write(b *block.Block) error {
	for _, bs := range mbs.BlockStores {
		err := bs.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

//Read - implement interface
func (mbs *MultiBlockStore) Read(hash string) (*block.Block, error) {
	var b *block.Block
	var err error
	for _, bs := range mbs.BlockStores {
		b, err = bs.Read(hash)
		if err == nil {
			break
		}
	}
	return b, err
}

//ReadWithBlockSummary - implement interface
func (mbs *MultiBlockStore) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	var b *block.Block
	var err error
	for _, bstore := range mbs.BlockStores {
		b, err = bstore.ReadWithBlockSummary(bs)
		if err == nil {
			break
		}
	}
	return b, err
}

//Delete - implement interface
func (mbs *MultiBlockStore) Delete(hash string) error {
	for _, bs := range mbs.BlockStores {
		err := bs.Delete(hash)
		if err != nil {
			return err
		}
	}
	return nil
}

//DeleteBlock - implement interface
func (mbs *MultiBlockStore) DeleteBlock(b *block.Block) error {
	for _, bs := range mbs.BlockStores {
		err := bs.DeleteBlock(b)
		if err != nil {
			return err
		}
	}
	return nil
}
