package blockstore

import (
	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/store"
)

type blockStore interface {
	Put(hash string, b *block.Block) error
	Get(hash string) (*block.Block, error)
	Delete(hash string) error
	IsExist(hash string) bool
}

func newFSBlockStore(dir string) (blockStore, error) {
	fs, err := store.NewFSStore(dir)
	if err != nil {
		return nil, err
	}
	return newBlkStore(fs), nil
}

func newBlkStore(store store.Store) blockStore {
	return &blkStore{
		store: store,
		newBlock: func() *block.Block {
			return datastore.GetEntityMetadata("block").Instance().(*block.Block)
		},
	}
}

type blkStore struct {
	store    store.Store
	newBlock func() *block.Block
}

func (store *blkStore) Put(hash string, b *block.Block) error {
	return store.store.Put([]byte(hash), b.Encode())
}

func (store *blkStore) Get(hash string) (*block.Block, error) {
	data, err := store.store.Get([]byte(hash))
	if err != nil {
		return nil, err
	}
	b := store.newBlock()
	if err := b.Decode(data); err != nil {
		return nil, err
	}
	return b, nil
}

func (store *blkStore) Delete(hash string) error {
	return store.store.Delete([]byte(hash))
}

func (store *blkStore) IsExist(hash string) bool {
	return store.store.IsExist([]byte(hash))
}
