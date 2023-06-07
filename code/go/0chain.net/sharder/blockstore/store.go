package blockstore

import (
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

/*BlockStoreI - an interface to read and write blocks to some storage */
type BlockStoreI interface {
	Write(b *block.Block) error
	Read(hash string) (*block.Block, error)
	ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error)
}

/*GetStore - get the block store that's is setup */
//func GetStore() BlockStoreI {
//	return store
//}

/*SetupStore - Setup a file system based block storage */
func SetupStore(s BlockStoreI) {
	store = s
}

func Write(b *block.Block) error {
	blockEntityMeta := datastore.GetEntityMetadata("block")
	tctx := ememorystore.WithEntityConnection(common.GetRootContext(), blockEntityMeta)
	defer ememorystore.Close(tctx)

	err := blockEntityMeta.GetStore().Write(tctx, b)
	if err != nil {
		return err
	}

	// Write the transactions, keyspace the hash
	tCon := ememorystore.GetEntityCon(tctx, blockEntityMeta)
	return tCon.Commit()
}

func Read(hash string) (*block.Block, error) {
	t := time.Now()
	blockEntityMeta := datastore.GetEntityMetadata("block")
	tctx := ememorystore.WithEntityConnection(common.GetRootContext(), blockEntityMeta)
	defer ememorystore.Close(tctx)

	b := blockEntityMeta.Instance().(*block.Block)
	err := blockEntityMeta.GetStore().Read(tctx, hash, b)
	if err != nil {
		return nil, err
	}
	fmt.Println("## read block:", time.Since(t))
	return b, nil
}

func MultipleRead(hashes []string) ([]*block.Block, error) {
	//t := time.Now()
	blockEntityMeta := datastore.GetEntityMetadata("block")
	tctx := ememorystore.WithEntityConnection(common.GetRootContext(), blockEntityMeta)
	defer ememorystore.Close(tctx)

	bes := make([]datastore.Entity, len(hashes))
	for i := 0; i < len(hashes); i++ {
		bes[i] = blockEntityMeta.Instance()
	}
	err := blockEntityMeta.GetStore().MultiRead(tctx, blockEntityMeta, hashes, bes)
	if err != nil {
		return nil, err
	}

	blocks := make([]*block.Block, len(hashes))
	for i := 0; i < len(hashes); i++ {
		blocks[i] = bes[i].(*block.Block)
	}

	//fmt.Println("## read blocks:", time.Since(t))
	return blocks, nil
}
